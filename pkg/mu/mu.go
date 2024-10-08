package mu

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	Mutexes  = make(map[string]*SharedMutex)
	globalMu sync.RWMutex
)

const defaultDelay = time.Duration(1) * time.Second

func GetMutex(name string) *SharedMutex {
	globalMu.RLock()
	mu, ok := Mutexes[name]
	globalMu.RUnlock()
	if !ok {
		mu = NewSharedMutex(name)
	}
	return mu
}

type SharedMutex struct {
	name         string
	delay        time.Duration
	sig          chan os.Signal
	sigMu        *sync.RWMutex
	sigListening *atomic.Bool
	mu           *sync.RWMutex
}

func NewSharedMutex(name string) *SharedMutex {
	globalMu.Lock()
	sm := &SharedMutex{
		name:         name,
		mu:           &sync.RWMutex{},
		sigMu:        &sync.RWMutex{},
		delay:        defaultDelay,
		sig:          make(chan os.Signal, 1),
		sigListening: &atomic.Bool{},
	}
	sm.sigListening.Store(false)

	Mutexes[name] = sm
	globalMu.Unlock()

	return sm
}

func (sm *SharedMutex) WithSIGHUPUnlock() *SharedMutex {
	sm.mu.Lock()
	sm.sigMu.Lock()
	signal.Notify(sm.sig, syscall.SIGHUP)
	sm.sigMu.Unlock()
	sm.mu.Unlock()
	log.Printf("[mutex] SIGHUP signal unlock enabled for '%s'", sm.name)
	return sm
}

func (sm *SharedMutex) WithDelayedUnlock(delay time.Duration) *SharedMutex {
	sm.mu.Lock()
	sm.delay = delay
	sm.mu.Unlock()
	return sm
}

func (sm *SharedMutex) watchSIGHUP() {
	if sm.sig == nil {
		return
	}
	if sm.sigListening.Load() {
		return
	}
	sm.sigListening.Store(true)
	select {
	case <-sm.sig:
		if !sm.sigMu.TryLock() {
			log.Printf("[mutex][race] SIGHUP signal received, but already unlocking '%s'", sm.name)
		}
		if !sm.sigListening.Load() {
			sm.sigMu.Unlock()
			return
		}
		sm.unlock("by SIGGUP signal")
		sm.sigMu.Unlock()
	}
	sm.sigListening.Store(false)
}

func (sm *SharedMutex) Lock() {
	sm.mu.Lock()
	log.Printf("[mutex] '%s' locked", sm.name)
	go sm.watchSIGHUP()
}

func (sm *SharedMutex) Unlock(ctx ...context.Context) {
	if sm.delay > 0 {
		time.Sleep(sm.delay)
	}
	if !sm.sigMu.TryLock() {
		log.Printf("[mutex][race] unlock '%s' already in progress", sm.name)
		return
	}

	if sm.sigListening.Load() && len(ctx) == 0 {
		ctx1, cancel := context.WithTimeout(context.Background(), time.Duration(20)*time.Second)
		defer cancel()
		ctx = append(ctx, ctx1)
	}

	if len(ctx) == 1 {
		select {
		case <-ctx[0].Done():
			log.Printf("[mutex] '%s' unlock context cancelled", sm.name)
			sm.sigMu.Unlock()
			return
		default:
		}
	}
	sm.unlock()
	sm.sigMu.Unlock()
}

func (sm *SharedMutex) unlock(details ...string) {
	if sm.mu.TryRLock() {
		sm.mu.RUnlock()
		return
	}
	sm.mu.Unlock()
	log.Printf("[mutex] '%s' unlocked %s", sm.name, strings.Join(details, " "))
}

func (sm *SharedMutex) RLock() {
	sm.mu.RLock()
	// log.Printf("[mutex] '%s' rlocked", sm.name)
}

func (sm *SharedMutex) RUnlock() {
	sm.mu.RUnlock()
	// log.Printf("[mutex] '%s' runlocked", sm.name)
}

func (sm *SharedMutex) Name() string {
	return sm.name
}

func (sm *SharedMutex) String() string {
	return sm.name
}

package mu

import (
	"log"
	"sync"
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
	name  string
	delay time.Duration
	mu    *sync.RWMutex
}

func NewSharedMutex(name string) *SharedMutex {
	globalMu.Lock()
	sm := &SharedMutex{name: name, mu: &sync.RWMutex{}, delay: defaultDelay}
	Mutexes[name] = sm
	globalMu.Unlock()
	return sm
}

func (sm *SharedMutex) WithDelayedUnlock(delay time.Duration) *SharedMutex {
	sm.mu.Lock()
	sm.delay = delay
	sm.mu.Unlock()
	return sm
}

func (sm *SharedMutex) Lock() {
	sm.mu.Lock()
	log.Printf("[mutex] '%s' locked", sm.name)
}

func (sm *SharedMutex) Unlock() {
	if sm.delay > 0 {
		time.Sleep(sm.delay)
	}
	sm.mu.Unlock()
	log.Printf("[mutex] '%s' unlocked", sm.name)
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

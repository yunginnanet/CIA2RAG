package mu

import (
	"log"
	"sync"
)

var (
	Mutexes  = make(map[string]*SharedMutex)
	globalMu sync.RWMutex
)

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
	name string
	mu   *sync.RWMutex
}

func NewSharedMutex(name string) *SharedMutex {
	globalMu.Lock()
	sm := &SharedMutex{name: name, mu: &sync.RWMutex{}}
	Mutexes[name] = sm
	globalMu.Unlock()
	return sm
}

func (sm *SharedMutex) Lock() {
	sm.mu.Lock()
	log.Printf("[mutex] '%s' locked", sm.name)
}

func (sm *SharedMutex) Unlock() {
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

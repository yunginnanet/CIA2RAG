package mu

import (
	"syscall"
	"testing"
	"time"
)

func TestGetMutex_ReturnsExistingMutex(t *testing.T) {
	NewSharedMutex(t.Name())
	mutex := GetMutex(t.Name())
	if mutex.Name() != t.Name() {
		t.Errorf("expected mutex name to be '%s', got %s", t.Name(), mutex.Name())
	}
}

func TestGetMutex_CreatesNewMutex(t *testing.T) {
	mutex := GetMutex(t.Name())
	if mutex.Name() != t.Name() {
		t.Errorf("expected mutex name to be '%s', got %s", t.Name(), mutex.Name())
	}
}

func TestWithSIGHUPUnlock_EnablesSignalHandling(t *testing.T) {
	mutex := NewSharedMutex(t.Name()).WithSIGHUPUnlock()
	if mutex == nil {
		t.Errorf("expected mutex to be initialized, got nil")
	}
}

func TestWithDelayedUnlock_SetsDelay(t *testing.T) {
	delay := 2 * time.Second
	mutex := NewSharedMutex(t.Name()).WithDelayedUnlock(delay)
	if mutex == nil || mutex.delay != delay {
		if mutex == nil {
			t.Fatalf("expected mutex to be initialized, got nil")
		}
		t.Errorf("expected delay to be %v, got %v", delay, mutex.delay)
	}
}

func TestLock_SetsLock(t *testing.T) {
	mutex := NewSharedMutex(t.Name())
	mutex.Lock()
	if mutex.mu.TryLock() {
		t.Errorf("expected mutex to be locked")
	}
	mutex.Unlock()
}

func TestUnlock_ReleasesLock(t *testing.T) {
	mutex := NewSharedMutex(t.Name())
	mutex.Lock()
	mutex.Unlock()
	if !mutex.mu.TryLock() {
		t.Errorf("expected mutex to be unlocked")
	}
	mutex.Unlock()
}

func TestRLock_SetsReadLock(t *testing.T) {
	mutex := NewSharedMutex(t.Name())
	mutex.RLock()
	if !mutex.mu.TryRLock() {
		t.Errorf("expected mutex to be read-locked")
	}
	mutex.RUnlock()
}

func TestRUnlock_ReleasesReadLock(t *testing.T) {
	mutex := NewSharedMutex(t.Name())
	mutex.RLock()
	mutex.RUnlock()
	if !mutex.mu.TryRLock() {
		t.Errorf("expected mutex to be read-unlocked")
	}
	mutex.RUnlock()
}

func TestWatchSIGHUP_HandlesSignal(t *testing.T) {
	mutex := NewSharedMutex(t.Name()).WithSIGHUPUnlock()
	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGHUP); err != nil {
			t.Errorf("failed to send SIGHUP signal: %v", err)
		}
	}()
	mutex.Lock()
	mutex.Unlock()
	if !mutex.mu.TryLock() {
		t.Errorf("expected mutex to be unlocked after SIGHUP")
	}
	mutex.Unlock()
}

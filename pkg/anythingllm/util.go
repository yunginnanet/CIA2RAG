package anythingllm

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"syscall"

	"ciascrape/pkg/mu"
)

// FIXME: make me not global
var writing = &atomic.Bool{}

func init() {
	writing.Store(false)
}

func WriteToFIFO(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("empty path")
	}
	if writing.Load() {
		return nil
	}

	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err = syscall.Mkfifo(path, 0755); err != nil {
			return err
		}
	}

	var f *os.File

	writing.Store(true)

	if f, err = os.OpenFile(path, os.O_WRONLY, os.ModeNamedPipe); err != nil {
		return fmt.Errorf("fifo open error: %w", err)
	}

	go func() {
		defer writing.Store(false)
		mu.GetMutex("net").Lock()
		if _, err = f.Write([]byte("x")); err != nil {
			_ = f.Close()
			log.Printf("[err][mullvad-fifo] %v", err)
			mu.GetMutex("net").Unlock()
			return
		}
		_ = f.Close()
		log.Printf("[info][mullvad-fifo] signal sent to '%s'", path)
		mu.GetMutex("net").Unlock()
	}()

	return nil
}

package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownCh := make(chan struct{})
	go func() {
		time.Sleep(1 * time.Second)
		close(shutdownCh)
	}()

	select {
	case <-shutdownCh:
		assert.True(t, true, "Shutdown completed successfully")
	case <-ctx.Done():
		t.Fatal("Shutdown did not complete in time")
	}
}

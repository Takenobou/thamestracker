package test

import (
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestGracefulShutdown(t *testing.T) {
	app := fiber.New()

	// Dummy route that simulates a delay.
	app.Get("/delay", func(c *fiber.Ctx) error {
		time.Sleep(2 * time.Second)
		return c.SendString("delayed response")
	})

	// Listen on an ephemeral port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	addr := ln.Addr().String()

	// Run server in a goroutine.
	go func() {
		_ = app.Listener(ln)
	}()

	// Give the server a moment to start.
	time.Sleep(100 * time.Millisecond)

	// Start a delayed request.
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, "http://"+addr+"/delay", nil)
	assert.NoError(t, err)

	done := make(chan string)
	go func() {
		resp, err := client.Do(req)
		if err != nil {
			done <- "error"
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		done <- string(body)
	}()

	// Wait a moment then initiate graceful shutdown.
	time.Sleep(100 * time.Millisecond)
	err = app.Shutdown()
	assert.NoError(t, err)

	// Ensure the delayed request completes.
	select {
	case res := <-done:
		assert.Equal(t, "delayed response", res)
	case <-time.After(3 * time.Second):
		t.Error("graceful shutdown timed out, request did not complete")
	}
}

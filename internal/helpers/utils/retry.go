package utils

import (
	"time"
)

// Retry calls the provided function up to maxAttempts times, with exponential backoff starting at initial.
// It returns nil on first success, or the last error if all attempts fail.
func Retry(maxAttempts int, initialBackoff time.Duration, fn func() error) error {
	backoff := initialBackoff
	var err error
	for i := 0; i < maxAttempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(backoff)
		backoff *= 2
	}
	return err
}

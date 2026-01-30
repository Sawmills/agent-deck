package ai

import (
	"fmt"
	"log"
	"time"
)

// WithRetry executes fn with exponential backoff
// Retries: 3 attempts with 1s, 2s, 4s delays
func WithRetry(fn func() error) error {
	delays := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
	}

	var err error
	for attempt := 0; attempt < len(delays)+1; attempt++ {
		if attempt > 0 {
			delay := delays[attempt-1]
			log.Printf("Retry attempt %d/%d after %v", attempt, len(delays), delay)
			time.Sleep(delay)
		}

		err = fn()
		if err == nil {
			return nil
		}

		log.Printf("Attempt %d failed: %v", attempt+1, err)
	}

	return fmt.Errorf("failed after %d attempts: %w", len(delays)+1, err)
}

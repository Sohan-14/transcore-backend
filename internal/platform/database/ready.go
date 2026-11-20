package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const maxReadyBackoff = 2 * time.Second

// waitReady calls ping until it succeeds or ctx ends, backing off from
// initialDelay up to 2s. On failure the error says WHY the last attempt failed
// ===============================================================
func waitReady(ctx context.Context, ping func(context.Context) error, log *slog.Logger, initialDelay time.Duration) error {
	delay := initialDelay
	timer := time.NewTimer(delay)
	defer timer.Stop()

	for attempt := 1; ; attempt++ {
		err := ping(ctx)
		if err == nil {
			return nil
		}

		if log != nil {
			log.WarnContext(ctx, "database not ready",
				"attempt", attempt,
				"retry_in", delay.String(),
				"error", err,
			)
		}

		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(delay)

		select {
		case <-ctx.Done():
			return fmt.Errorf("database not reachable after %d attempts: %w", attempt, err)
		case <-timer.C:
		}

		delay = min(delay*2, maxReadyBackoff)
	}
}

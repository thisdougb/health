package config

import (
	"context"
	"fmt"
	"time"
)

// Public methods
func LogInfo(ctx context.Context, msg string) {
	writeToLog(ctx, "INFO", msg)
}

func LogError(ctx context.Context, msg string) {
	writeToLog(ctx, "ERROR", msg)
}

func LogDebug(ctx context.Context, msg string) {
	if GetContextDebug(ctx) {
		writeToLog(ctx, "DEBUG", msg)
	}
}

// Private methods
func writeToLog(ctx context.Context, severity string, msg string) {

	// Always do normal logging
	fmt.Printf("%s (health) %s +%s [%s] %s\n",
		time.Now().UTC().Format("2006/01/02 15:04:05"),
		severity,
		sinceCreated(ctx),
		GetContextCorrelationId(ctx),
		msg)

	// Additionally collect if enabled
	if IsLogCollectionEnabled(ctx) {
		if logs := ctx.Value(CollectedLogsContextKey("logs")); logs != nil {
			logSlice := logs.(*[]CollectedLog)
			createdTime := time.Unix(GetContextTimeCreated(ctx), 0)
			elapsedMs := time.Since(createdTime).Seconds() * 1000

			*logSlice = append(*logSlice, CollectedLog{
				Timestamp: time.Now().UTC(),
				Severity:  severity,
				Message:   msg,
				CID:       GetContextCorrelationId(ctx),
				ElapsedMs: elapsedMs,
			})
		}
	}
}

func sinceCreated(ctx context.Context) string {

	createdTime := time.Unix(GetContextTimeCreated(ctx), 0)
	t := time.Since(createdTime).Seconds()

	return fmt.Sprintf("%.1fs", t)
}

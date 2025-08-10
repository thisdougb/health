package config

import (
	"context"
	"fmt"
	"log"
	"runtime"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if BoolValue("HEALTH_DEBUG") {
		LogInfo(context.Background(), fmt.Sprintf("health config.init(): arch: %v", runtime.GOOS))
		LogInfo(context.Background(), "health config initialized with environment variable defaults")
	}
}


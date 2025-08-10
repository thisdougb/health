package config

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

type (
	CorrelationContextKey    string
	DebugContextKey          string
	TimeCreatedContextKey    string
	SimulationModeContextKey string
	LogCollectionContextKey  string
	CollectedLogsContextKey  string
)

type CollectedLog struct {
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	CID       string    `json:"correlation_id"`
	ElapsedMs float64   `json:"elapsed_ms"`
}

const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func SetContextCorrelationId(ctx context.Context, value string) context.Context {

	id := make([]byte, 8)
	for idx := range 8 {
		n := rand.Intn(len(chars))
		id[idx] = chars[n]
	}

	newctx := context.WithValue(ctx, CorrelationContextKey("cid"), fmt.Sprintf("%s-%s", string(id), value))

	// if the created time is unset then set it. test for -1 as 0 could be
	// a symptom of a default unset value
	t := GetContextTimeCreated(ctx)
	if t == -1 {
		newctx = context.WithValue(
			newctx,
			TimeCreatedContextKey("timeCreated"),
			time.Now().Unix())
	}

	newctx = context.WithValue(newctx, DebugContextKey("debug"), BoolValue("DEBUG"))

	return newctx
}
func GetContextTimeCreated(ctx context.Context) int64 {

	key := TimeCreatedContextKey("timeCreated")

	if v := ctx.Value(key); v != nil {
		return v.(int64)
	}
	return -1
}
func AppendToContextCorrelationId(ctx context.Context, value string) context.Context {
	key := CorrelationContextKey("cid")
	id := GetContextCorrelationId(ctx)
	newctx := context.WithValue(ctx, key, id+"-"+value)
	return newctx
}
func GetContextCorrelationId(ctx context.Context) string {

	key := CorrelationContextKey("cid")

	if v := ctx.Value(key); v != nil {
		return v.(string)
	}

	return "no-id"
}

func GetContextDebug(ctx context.Context) bool {

	key := DebugContextKey("debug")

	if v := ctx.Value(key); v != nil {
		return v.(bool)
	}

	return false
}

// Simulation Mode Functions
func EnableSimulationMode(ctx context.Context) context.Context {
	return context.WithValue(ctx, SimulationModeContextKey("simulation_mode"), true)
}

func IsSimulationMode(ctx context.Context) bool {
	if v := ctx.Value(SimulationModeContextKey("simulation_mode")); v != nil {
		return v.(bool)
	}
	return false
}

// Log Collection Functions
func EnableLogCollection(ctx context.Context) context.Context {
	logs := make([]CollectedLog, 0)
	ctx = context.WithValue(ctx, LogCollectionContextKey("collect"), true)
	ctx = context.WithValue(ctx, CollectedLogsContextKey("logs"), &logs)
	return ctx
}

func IsLogCollectionEnabled(ctx context.Context) bool {
	if v := ctx.Value(LogCollectionContextKey("collect")); v != nil {
		return v.(bool)
	}
	return false
}

func DumpLogsAsJSON(ctx context.Context) (string, error) {
	if logs := ctx.Value(CollectedLogsContextKey("logs")); logs != nil {
		jsonData, err := json.Marshal(logs.(*[]CollectedLog))
		if err != nil {
			return "", err
		}
		return string(jsonData), nil
	}
	return "[]", nil
}

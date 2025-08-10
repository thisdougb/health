package config

import (
	"os"
	"strconv"
)

var defaultValues = map[string]interface{}{
	// Health package configuration
	"HEALTH_SAMPLE_RATE":               60,     // Sample rate in seconds for system metrics
	"HEALTH_PERSISTENCE_ENABLED":       false,  // Enable SQLite persistence
	"HEALTH_DB_PATH":                   "/tmp/health.db", // SQLite database path
	"HEALTH_FLUSH_INTERVAL":            "60s",  // How often to flush metrics to storage
	"HEALTH_BATCH_SIZE":                100,    // Number of metrics to batch before writing
	"HEALTH_BACKUP_ENABLED":            false,  // Enable automatic backups
	"HEALTH_BACKUP_DIR":                "./backups", // Directory for backup files
	"HEALTH_BACKUP_RETENTION_DAYS":     30,     // Days to retain backup files
	"HEALTH_DEBUG":                     false,  // Enable debug logging
}

func StringValue(key string) string {
	if defaultValue, ok := defaultValues[key]; ok {
		return getEnvVar(key, defaultValue.(string)).(string)
	}
	return ""
}

// ValueAsInt gets a string value from the env or default
func Int64Value(key string) int64 {

	if defaultValue, ok := defaultValues[key]; ok {
		return getEnvVar(key, defaultValue.(int64)).(int64)
	}
	return 0
}

// ValueAsInt gets a string value from the env or default
func Int32Value(key string) int32 {

	if defaultValue, ok := defaultValues[key]; ok {
		return getEnvVar(key, defaultValue.(int32)).(int32)
	}
	return 0
}

// ValueAsInt gets a string value from the env or default
func IntValue(key string) int {

	if defaultValue, ok := defaultValues[key]; ok {
		return getEnvVar(key, defaultValue.(int)).(int)
	}
	return 0
}

// ValueAsBool gets a string value from the env or default
func BoolValue(key string) bool {

	if defaultValue, ok := defaultValues[key]; ok {
		return getEnvVar(key, defaultValue.(bool)).(bool)
	}
	return false
}

func getEnvVar(key string, fallback interface{}) interface{} {

	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}

	switch fallback.(type) {
	case string:
		return value
	case bool:
		valueAsBool, err := strconv.ParseBool(value)
		if err != nil {
			return fallback
		}
		return valueAsBool
	case int:
		valueAsInt, err := strconv.Atoi(value)
		if err != nil {
			return fallback
		}
		return valueAsInt
	}
	return fallback
}

// Package config provides application configuration loaded from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration settings for the Transverse backend service.
type Config struct {
	// Port is the HTTP server listening port.
	Port string `json:"port"`

	// DatabaseURL is the PostgreSQL connection string (including credentials, host, db).
	DatabaseURL string `json:"database_url"`

	// DBPoolMinConns is the minimum number of idle connections maintained in the pool.
	DBPoolMinConns int `json:"db_pool_min_conns"`

	// DBPoolMaxConns is the maximum number of connections allowed in the pool.
	DBPoolMaxConns int `json:"db_pool_max_conns"`

	// ONNXModelPath is the filesystem path to the BAAI/bge-small-en-v1.5 ONNX model.
	ONNXModelPath string `json:"onnx_model_path"`

	// TopicsGraphPath is the filesystem path to the topics knowledge graph JSON file.
	TopicsGraphPath string `json:"topics_graph_path"`

	// Judge0BaseURL is the base URL for the Judge0 code execution API.
	Judge0BaseURL string `json:"judge0_base_url"`

	// Judge0APIKey is the RapidAPI / Judge0 authentication key.
	Judge0APIKey string `json:"judge0_api_key"`

	// Judge0TimeoutMs is the timeout in milliseconds for Judge0 API calls.
	Judge0TimeoutMs int `json:"judge0_timeout_ms"`

	// CacheEnabled toggles in-memory caching for hot queries and session state.
	CacheEnabled bool `json:"cache_enabled"`

	// JWTSecret is the secret key used for signing and validating JWT tokens.
	JWTSecret string `json:"jwt_secret"`
}

// Load reads all configuration from environment variables, applying defaults
// for non-required settings and panicking via mustGetenv if required variables are absent.
func Load() *Config {
	return &Config{
		Port:            getEnvWithDefault("PORT", "8080"),
		DatabaseURL:     mustGetenv("DATABASE_URL"),
		DBPoolMinConns:  getEnvAsInt("DB_POOL_MIN_CONNS", 4),
		DBPoolMaxConns:  getEnvAsInt("DB_POOL_MAX_CONNS", 20),
		ONNXModelPath:   getEnvWithDefault("ONNX_MODEL_PATH", "./models/bge-small-en-v1.5.onnx"),
		TopicsGraphPath: getEnvWithDefault("TOPICS_GRAPH_PATH", "./data/topics.json"),
		Judge0BaseURL:   getEnvWithDefault("JUDGE0_BASE_URL", "https://judge0-ce.p.rapidapi.com"),
		Judge0APIKey:    getEnvWithDefault("JUDGE0_API_KEY", ""),
		Judge0TimeoutMs: getEnvAsInt("JUDGE0_TIMEOUT_MS", 5000),
		CacheEnabled:    getEnvAsBool("CACHE_ENABLED", true),
		JWTSecret:       getEnvWithDefault("JWT_SECRET", "change-me-in-production"),
	}
}

// mustGetenv retrieves the environment variable value for the given key,
// or panics with an informative message if the variable is unset or empty.
func mustGetenv(key string) string {
	val := os.Getenv(key)
	if strings.TrimSpace(val) == "" {
		panic(fmt.Sprintf("critical configuration error: environment variable %q is required but not set", key))
	}
	return val
}

// getEnvWithDefault retrieves an environment variable value or falls back to defaultValue.
func getEnvWithDefault(key, defaultValue string) string {
	val := os.Getenv(key)
	if strings.TrimSpace(val) == "" {
		return defaultValue
	}
	return val
}

// getEnvAsInt parses an environment variable as an integer or returns defaultValue.
func getEnvAsInt(key string, defaultValue int) int {
	valStr := os.Getenv(key)
	if strings.TrimSpace(valStr) == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}

// getEnvAsBool parses an environment variable as a boolean or returns defaultValue.
func getEnvAsBool(key string, defaultValue bool) bool {
	valStr := os.Getenv(key)
	if strings.TrimSpace(valStr) == "" {
		return defaultValue
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}

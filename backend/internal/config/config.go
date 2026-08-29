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

	// Connector configurations
	GithubAPIBase            string `json:"github_api_base"`
	LeetcodeGraphQLURL       string `json:"leetcode_graphql_url"`
	CodeforcesAPIBase        string `json:"codeforces_api_base"`
	ConnectorTimeoutSeconds  int    `json:"connector_timeout_seconds"`
	ConnectorMaxReposScanned int    `json:"connector_max_repos_scanned"`
	GithubToken              string `json:"github_token"` // Optional PAT for higher rate limits

	// OAuth2
	OAuthGithubClientID     string `json:"oauth_github_client_id"`
	OAuthGithubClientSecret string `json:"oauth_github_client_secret"`
	OAuthGithubRedirectURL  string `json:"oauth_github_redirect_url"`
	OAuthGoogleClientID     string `json:"oauth_google_client_id"`
	OAuthGoogleClientSecret string `json:"oauth_google_client_secret"`
	OAuthGoogleRedirectURL  string `json:"oauth_google_redirect_url"`
	JWTAccessTTLMinutes     int    `json:"jwt_access_ttl_minutes"`
	JWTRefreshTTLDays       int    `json:"jwt_refresh_ttl_days"`

	// Redis
	RedisAddr string `json:"redis_addr"`
	RedisDB   int    `json:"redis_db"`

	// LLM
	ZaiAPIKey      string `json:"zai_api_key"`
	ZaiBaseURL     string `json:"zai_base_url"`
	ZaiModel       string `json:"zai_model"`
	ZaiTimeoutSecs int    `json:"zai_timeout_secs"`
	ZaiMaxRetries  int    `json:"zai_max_retries"`
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
		TopicsGraphPath: getEnvWithDefault("TOPICS_GRAPH_PATH", "../data/topics.json"),
		Judge0BaseURL:   getEnvWithDefault("JUDGE0_BASE_URL", "https://judge0-ce.p.rapidapi.com"),
		Judge0APIKey:    getEnvWithDefault("JUDGE0_API_KEY", ""),
		Judge0TimeoutMs: getEnvAsInt("JUDGE0_TIMEOUT_MS", 5000),
		CacheEnabled:    getEnvAsBool("CACHE_ENABLED", true),
		JWTSecret:       getEnvWithDefault("JWT_SECRET", "change-me-in-production"),
		
		GithubAPIBase:            getEnvWithDefault("GITHUB_API_BASE", "https://api.github.com"),
		LeetcodeGraphQLURL:       getEnvWithDefault("LEETCODE_GRAPHQL_URL", "https://leetcode.com/graphql"),
		CodeforcesAPIBase:        getEnvWithDefault("CODEFORCES_API_BASE", "https://codeforces.com/api"),
		ConnectorTimeoutSeconds:  getEnvAsInt("CONNECTOR_TIMEOUT_SECONDS", 10),
		ConnectorMaxReposScanned: getEnvAsInt("CONNECTOR_MAX_REPOS_SCANNED", 15),
		GithubToken:              os.Getenv("GITHUB_TOKEN"),

		OAuthGithubClientID:     os.Getenv("OAUTH_GITHUB_CLIENT_ID"),
		OAuthGithubClientSecret: os.Getenv("OAUTH_GITHUB_CLIENT_SECRET"),
		OAuthGithubRedirectURL:  os.Getenv("OAUTH_GITHUB_REDIRECT_URL"),
		OAuthGoogleClientID:     os.Getenv("OAUTH_GOOGLE_CLIENT_ID"),
		OAuthGoogleClientSecret: os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET"),
		OAuthGoogleRedirectURL:  os.Getenv("OAUTH_GOOGLE_REDIRECT_URL"),
		JWTAccessTTLMinutes:     getEnvAsInt("JWT_ACCESS_TTL_MINUTES", 15),
		JWTRefreshTTLDays:       getEnvAsInt("JWT_REFRESH_TTL_DAYS", 30),

		RedisAddr: getEnvWithDefault("REDIS_ADDR", "redis:6379"),
		RedisDB:   getEnvAsInt("REDIS_DB", 0),
		
		ZaiAPIKey:      os.Getenv("ZAI_API_KEY"),
		ZaiBaseURL:     getEnvWithDefault("ZAI_BASE_URL", "https://api.z.ai/api/paas/v4"),
		ZaiModel:       getEnvWithDefault("ZAI_MODEL", "glm-4.7-flash"),
		ZaiTimeoutSecs: getEnvAsInt("ZAI_TIMEOUT_SECONDS", 30),
		ZaiMaxRetries:  getEnvAsInt("ZAI_MAX_RETRIES", 2),
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

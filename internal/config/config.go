package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Env, Port, Key, AdminUser, AdminPass                   string
	DBConn, DBHost, DBPort, DBName, DBUser, DBPass, DBSSL string
	SearchEndpoint, OpenAIEndpoint, OpenAIKey, OpenAIModel string
}

func get(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		Env: get("APP_ENV", "development"), Port: get("APP_PORT", "8081"),
		Key: get("APP_KEY", "secret"), AdminUser: get("ADMIN_USERNAME", "admin"), AdminPass: get("ADMIN_PASSWORD", "admin"),
		DBConn: get("DB_CONNECTION", "pgsql"), DBHost: get("DB_HOST", "127.0.0.1"), DBPort: get("DB_PORT", "5432"),
		DBName: get("DB_DATABASE", "pabwe_p9_blog"), DBUser: get("DB_USERNAME", "postgres"),
		DBPass: get("DB_PASSWORD", "postgres"), DBSSL: get("DB_SSLMODE", "disable"),
		SearchEndpoint: get("SEARCH_ENGINE_ENDPOINT", ""), OpenAIEndpoint: get("OPENAI_ENDPOINT", ""),
		OpenAIKey: get("OPENAI_KEY", ""), OpenAIModel: get("OPENAI_MODEL", ""),
	}
}

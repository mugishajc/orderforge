package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	ServerPort  int
	DatabaseURL string
	RedisURL    string
	NatsURL     string
	LogLevel    string
}

func Load() (*Config, error) {
	port, err := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_PORT: %w", err)
	}

	cfg := &Config{
		ServerPort:  port,
		DatabaseURL: getEnv("DATABASE_URL", "postgres://orderforge:orderforge@localhost:5432/orderforge?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		NatsURL:     getEnv("NATS_URL", "nats://localhost:4222"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

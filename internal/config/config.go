package config

import (
	"log"
	"os"
)

type Config struct {
	AppPort    string
	LogLevel   string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
}

func Load() *Config {
	return &Config{
		DBUser:     mustGetEnv("DB_USER"),
		DBPassword: mustGetEnv("DB_PASSWORD"),
		DBHost:     mustGetEnv("DB_HOST"),
		DBPort:     mustGetEnv("DB_PORT"),
		DBName:     mustGetEnv("DB_NAME"),
		DBSSLMode:  mustGetEnv("DB_SSLMODE"),
		AppPort:    mustGetEnv("APP_PORT"),
		LogLevel:   mustGetEnv("LOG_LEVEL"),
	}
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("environment variable %s not set", key)
	}
	return v
}

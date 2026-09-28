package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	AppPort    string

	// RedisHost is host:port; empty disables the Redis cache.
	RedisHost string
	RedisDB   int

	// WeatherAPIKey is the weatherapi.com key; empty disables weather lookups.
	WeatherAPIKey string
}

func Load() *Config {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
	return &Config{
		DBHost:        os.Getenv("DB_HOST"),
		DBPort:        os.Getenv("DB_PORT"),
		DBUser:        os.Getenv("DB_USER"),
		DBPassword:    os.Getenv("DB_PASSWORD"),
		DBName:        os.Getenv("DB_NAME"),
		AppPort:       port,
		RedisHost:     os.Getenv("REDIS_HOST"),
		RedisDB:       redisDB,
		WeatherAPIKey: os.Getenv("WEATHERAPI_KEY"),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

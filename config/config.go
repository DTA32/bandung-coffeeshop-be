package config

import (
	"fmt"
	"net"
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

	// RedisHost empty disables the Redis cache; RedisPort defaults to 6379.
	RedisHost string
	RedisPort string
	RedisDB   int

	// WeatherAPIKey is the weatherapi.com key; empty disables weather lookups.
	WeatherAPIKey string
}

func Load() *Config {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
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
		RedisPort:     redisPort,
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

// RedisAddr is the host:port the Redis client dials.
func (c *Config) RedisAddr() string {
	return net.JoinHostPort(c.RedisHost, c.RedisPort)
}

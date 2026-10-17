package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	AppPort    string

	RedisHost string
	RedisPort string
	RedisDB   int

	WeatherAPIKey string

	GoogleClientID     string
	GoogleClientSecret string
	PublicBaseURL      string
	AuthCallbackURL    string
	AllowedOrigins     []string
	CookieSecure       bool
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
	publicBaseURL := strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
	callbackURL := os.Getenv("AUTH_CALLBACK_URL")
	if callbackURL == "" && publicBaseURL != "" {
		callbackURL = publicBaseURL + "/api/v1/auth/google/callback"
	}
	allowedOrigins := splitCSV(os.Getenv("ALLOWED_ORIGINS"))
	if len(allowedOrigins) == 0 && publicBaseURL != "" {
		allowedOrigins = []string{publicBaseURL}
	}
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:3000"}
	}
	cookieSecure := true
	if v, err := strconv.ParseBool(os.Getenv("COOKIE_SECURE")); err == nil {
		cookieSecure = v
	}
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

		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		PublicBaseURL:      publicBaseURL,
		AuthCallbackURL:    callbackURL,
		AllowedOrigins:     allowedOrigins,
		CookieSecure:       cookieSecure,
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

func (c *Config) RedisAddr() string {
	return net.JoinHostPort(c.RedisHost, c.RedisPort)
}

func (c *Config) GoogleAuthEnabled() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != "" && c.PublicBaseURL != ""
}

func splitCSV(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if s := strings.TrimRight(strings.TrimSpace(p), "/"); s != "" {
			out = append(out, s)
		}
	}
	return out
}

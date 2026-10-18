package main

import (
	"context"
	"log"

	"github.com/dta32/bandung-coffeeshop-be/cache"
	"github.com/dta32/bandung-coffeeshop-be/config"
	"github.com/dta32/bandung-coffeeshop-be/handler"
	"github.com/dta32/bandung-coffeeshop-be/middleware"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	cfg := config.Load()

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.AllowedOrigins
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE"}

	pool, err := pgxpool.New(context.Background(), cfg.DSN())
	if err != nil {
		log.Fatalf("failed to create db pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Printf("db ping failed (continuing): %v", err)
	} else {
		log.Println("db connected")
	}

	locationRepo := repository.NewLocationRepository(pool)
	locationSvc := service.NewLocationService(locationRepo)
	locationHdlr := handler.NewLocationHandler(locationSvc)

	// Redis is an optional, best-effort cache: without REDIS_HOST (or with
	// Redis down) callers fall back to their source.
	var redisCache *cache.Redis
	if cfg.RedisHost != "" {
		redisCache = cache.NewRedis(cfg.RedisAddr(), cfg.RedisDB)
		defer redisCache.Close()
		if err := redisCache.Ping(context.Background()); err != nil {
			log.Printf("redis ping failed (continuing): %v", err)
		} else {
			log.Println("redis connected")
		}
	}
	if cfg.WeatherAPIKey == "" {
		log.Println("WEATHERAPI_KEY not set; weather=current searches skip the weather filter")
	}
	weatherRepo := repository.NewWeatherRepository(cfg.WeatherAPIKey)
	weatherSvc := service.NewWeatherService(weatherRepo, redisCache)

	cafeRepo := repository.NewCafeRepository(pool)
	cafeSvc := service.NewCafeService(cafeRepo, weatherSvc)
	cafeHdlr := handler.NewCafeHandler(cafeSvc)

	filterRepo := repository.NewFilterRepository(pool)
	filterSvc := service.NewFilterService(filterRepo)
	filterHdlr := handler.NewFilterHandler(filterSvc)

	quicksearchRepo := repository.NewQuicksearchRepository(pool)
	quicksearchSvc := service.NewQuicksearchService(quicksearchRepo)
	quicksearchHdlr := handler.NewQuicksearchHandler(quicksearchSvc)

	// Auth needs Google credentials and Redis (sessions); without either the
	// auth routes answer 503 and everyone browses anonymously.
	var googleRepo *repository.GoogleRepository
	if cfg.GoogleAuthEnabled() {
		googleRepo = repository.NewGoogleRepository(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.AuthCallbackURL)
	} else {
		log.Println("GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET/PUBLIC_BASE_URL not set; auth routes respond 503")
	}
	if redisCache == nil {
		log.Println("REDIS_HOST not set; auth routes respond 503 (sessions need redis)")
	}
	userRepo := repository.NewUserRepository(pool)
	sessionRepo := repository.NewSessionRepository(redisCache)
	authSvc := service.NewAuthService(userRepo, sessionRepo, googleRepo)
	authHdlr := handler.NewAuthHandler(authSvc, cfg.PublicBaseURL, cfg.CookieSecure)

	r := gin.Default()
	r.Use(cors.New(corsConfig))
	r.GET("/health", handler.Health)

	v1 := r.Group("/v1")
	v1.Use(middleware.RequireAllowedOrigin(cfg.AllowedOrigins), middleware.LoadUser(authSvc))
	{
		v1.GET("/auth/:provider", authHdlr.Login)
		v1.GET("/auth/:provider/callback", authHdlr.Callback)
		v1.POST("/auth/logout", authHdlr.Logout)
		v1.GET("/me", middleware.RequireUser, authHdlr.Me)

		v1.GET("/quicksearch", quicksearchHdlr.Quicksearch)
		v1.GET("/location", locationHdlr.List)
		v1.GET("/location/:id", locationHdlr.GetByID)
		v1.GET("/search/cafes", cafeHdlr.Search)
		v1.GET("/cafe/random", cafeHdlr.Random)
		v1.GET("/cafe/:id", cafeHdlr.GetByID)
		v1.GET("/cafe/:id/review", cafeHdlr.GetReview)
		v1.GET("/cafe/:id/feed", cafeHdlr.GetFeed)
		v1.GET("/filters", filterHdlr.Get)
	}

	log.Printf("starting server on :%s", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

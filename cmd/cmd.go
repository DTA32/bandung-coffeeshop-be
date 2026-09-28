package main

import (
	"context"
	"log"

	"github.com/dta32/bandung-coffeeshop-be/cache"
	"github.com/dta32/bandung-coffeeshop-be/config"
	"github.com/dta32/bandung-coffeeshop-be/handler"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE"}

	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	cfg := config.Load()

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
	var weatherCache service.WeatherCache
	if cfg.RedisHost != "" {
		rdb := cache.NewRedis(cfg.RedisAddr(), cfg.RedisDB)
		defer rdb.Close()
		if err := rdb.Ping(context.Background()); err != nil {
			log.Printf("redis ping failed (continuing): %v", err)
		} else {
			log.Println("redis connected")
		}
		weatherCache = rdb
	}
	if cfg.WeatherAPIKey == "" {
		log.Println("WEATHERAPI_KEY not set; weather=current searches skip the weather filter")
	}
	weatherRepo := repository.NewWeatherRepository(cfg.WeatherAPIKey)
	weatherSvc := service.NewWeatherService(weatherRepo, weatherCache)

	cafeRepo := repository.NewCafeRepository(pool)
	cafeSvc := service.NewCafeService(cafeRepo, weatherSvc)
	cafeHdlr := handler.NewCafeHandler(cafeSvc)

	filterRepo := repository.NewFilterRepository(pool)
	filterSvc := service.NewFilterService(filterRepo)
	filterHdlr := handler.NewFilterHandler(filterSvc)

	quicksearchRepo := repository.NewQuicksearchRepository(pool)
	quicksearchSvc := service.NewQuicksearchService(quicksearchRepo)
	quicksearchHdlr := handler.NewQuicksearchHandler(quicksearchSvc)

	r := gin.Default()
	r.Use(cors.New(corsConfig))
	r.GET("/health", handler.Health)

	v1 := r.Group("/v1")
	{
		v1.GET("/quicksearch", quicksearchHdlr.Quicksearch)
		v1.GET("/location", locationHdlr.List)
		v1.GET("/location/:id", locationHdlr.GetByID)
		v1.GET("/search/cafes", cafeHdlr.Search)
		v1.GET("/cafe/random", cafeHdlr.Random)
		v1.GET("/cafe/:id", cafeHdlr.GetByID)
		v1.GET("/cafe/:id/review", cafeHdlr.GetReview)
		v1.GET("/filters", filterHdlr.Get)
	}

	log.Printf("starting server on :%s", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

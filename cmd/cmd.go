package main

import (
	"context"
	"log"

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

	cafeRepo := repository.NewCafeRepository(pool)
	cafeSvc := service.NewCafeService(cafeRepo)
	cafeHdlr := handler.NewCafeHandler(cafeSvc)

	r := gin.Default()
	r.Use(cors.New(corsConfig))
	r.GET("/health", handler.Health)

	v1 := r.Group("/v1")
	{
		v1.GET("/quicksearch", locationHdlr.Quicksearch)
		v1.GET("/location", locationHdlr.List)
		v1.GET("/location/:id", locationHdlr.GetByID)
		v1.GET("/search/cafes", cafeHdlr.Search)
		v1.GET("/cafe/:id", cafeHdlr.GetByID)
		v1.GET("/cafe/:id/review", cafeHdlr.GetReview)
	}

	log.Printf("starting server on :%s", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

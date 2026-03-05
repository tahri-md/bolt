package main

import (
	"bolt/routes"
	"bolt/service"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	nodeService := service.NewNodeService()
	// LRU eviction
	cacheService := service.NewCacheService(1024*1024, "LRU")

	// LFU eviction
	//cacheService := service.NewCacheService(1024*1024, "LFU")
	routes.RegisterNodeRoutes(r, nodeService)
	routes.RegisterCacheRoutes(r, cacheService)

	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

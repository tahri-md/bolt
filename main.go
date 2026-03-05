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
	cacheEntryService := service.NewCacheEntryService()

	routes.RegisterNodeRoutes(r, nodeService)
	routes.RegisterCacheEntryRoutes(r, cacheEntryService)

	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

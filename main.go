package main

import (
	"bolt/models"
	"bolt/routes"
	"bolt/service"
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create coordinator: 150 virtual nodes, replication factor 2, 1MB per node, LRU
	coordinator := service.NewCoordinatorService(150, 2, 1024*1024, "LRU")

	// register nodes
	coordinator.AddNode(&models.Node{
		ID:            "node-1",
		Address:       "localhost",
		Port:          8081,
		LastHeartbeat: time.Now(),
	})
	coordinator.AddNode(&models.Node{
		ID:            "node-2",
		Address:       "localhost",
		Port:          8082,
		LastHeartbeat: time.Now(),
	})
	coordinator.AddNode(&models.Node{
		ID:            "node-3",
		Address:       "localhost",
		Port:          8083,
		LastHeartbeat: time.Now(),
	})

	// start TTL cleanup for all nodes
	coordinator.StartAllTTLCleanup(ctx)

	routes.RegisterNodeRoutes(r, service.NewNodeService())
	routes.RegisterCacheRoutes(r, coordinator)

	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

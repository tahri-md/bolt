package routes

import (
	"bolt/models"
	"bolt/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AddNodeRequest struct {
	ID      string `json:"id" binding:"required"`
	Address string `json:"address" binding:"required"`
	Port    int    `json:"port" binding:"required"`
}

func RegisterNodeRoutes(r *gin.Engine, coordinator *service.CoordinatorService) {
	nodeGroup := r.Group("/nodes")

	nodeGroup.GET("/", func(c *gin.Context) {
		nodes := coordinator.GetAllNodes()
		c.JSON(http.StatusOK, nodes)
	})

	nodeGroup.GET("/:id", func(c *gin.Context) {
		id := c.Param("id")
		nodes := coordinator.GetAllNodes()
		for _, node := range nodes {
			if node.ID == id {
				c.JSON(http.StatusOK, node)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
	})

	nodeGroup.POST("/", func(c *gin.Context) {
		var req AddNodeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		node := &models.Node{
			ID:            req.ID,
			Address:       req.Address,
			Port:          req.Port,
			Status:        "active",
			LastHeartbeat: time.Now(),
		}

		if err := coordinator.AddNode(node); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, node)
	})

	nodeGroup.DELETE("/:id", func(c *gin.Context) {
		id := c.Param("id")
		if err := coordinator.RemoveNode(id); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "node removed"})
	})

	nodeGroup.PUT("/:id/heartbeat", func(c *gin.Context) {
		id := c.Param("id")
		if err := coordinator.UpdateHeartbeat(id); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "heartbeat received"})
	})
}

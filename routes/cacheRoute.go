package routes

import (
	"bolt/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type SetRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
	TTL   int64  `json:"ttl"`
}

func RegisterCacheRoutes(r *gin.Engine, coordinator *service.CoordinatorService) {
	cacheGroup := r.Group("/cache")

	cacheGroup.GET("/:key", func(c *gin.Context) {
		key := c.Param("key")
		value, err := coordinator.Get(key)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"key": key, "value": value})
	})

	cacheGroup.POST("/", func(c *gin.Context) {
		var req SetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ttl := time.Duration(req.TTL) * time.Second
		if err := coordinator.Set(req.Key, req.Value, ttl); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"key": req.Key, "value": req.Value, "ttl": req.TTL})
	})

	cacheGroup.PUT("/:key", func(c *gin.Context) {
		key := c.Param("key")
		var req struct {
			Value string `json:"value" binding:"required"`
			TTL   int64  `json:"ttl"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ttl := time.Duration(req.TTL) * time.Second
		if err := coordinator.Set(key, req.Value, ttl); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"key": key, "value": req.Value, "ttl": req.TTL})
	})

	cacheGroup.DELETE("/:key", func(c *gin.Context) {
		key := c.Param("key")
		if err := coordinator.Delete(key); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNoContent, nil)
	})

	cacheGroup.DELETE("/", func(c *gin.Context) {
		coordinator.Flush()
		c.JSON(http.StatusOK, gin.H{"message": "cache flushed"})
	})

	r.GET("/cache-metrics", func(c *gin.Context) {
		metrics := coordinator.GetAllMetrics()
		c.JSON(http.StatusOK, metrics)
	})

	r.GET("/cache-metrics/:nodeID", func(c *gin.Context) {
		nodeID := c.Param("nodeID")
		metrics, err := coordinator.GetNodeMetrics(nodeID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, metrics)
	})

	r.GET("/cache-node/:key", func(c *gin.Context) {
		key := c.Param("key")
		node, err := coordinator.GetNodeForKey(key)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"key": key, "node": node.ID})
	})
}

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
	TTL   int64  `json:"ttl"` // seconds, 0 means no expiry
}

func RegisterCacheRoutes(r *gin.Engine, cacheService *service.CacheService) {
	cacheGroup := r.Group("/cache")

	// GET /cache/ — get all entries
	cacheGroup.GET("/", func(c *gin.Context) {
		entries, err := cacheService.GetAllEntries()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, entries)
	})

	// GET /cache/:key — get a single entry
	cacheGroup.GET("/:key", func(c *gin.Context) {
		key := c.Param("key")
		value, err := cacheService.Get(key)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"key": key, "value": value})
	})

	// POST /cache/ — set an entry
	cacheGroup.POST("/", func(c *gin.Context) {
		var req SetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ttl := time.Duration(req.TTL) * time.Second
		if err := cacheService.Set(req.Key, req.Value, ttl); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"key": req.Key, "value": req.Value, "ttl": req.TTL})
	})

	// PUT /cache/:key — update an entry
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
		if err := cacheService.Set(key, req.Value, ttl); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"key": key, "value": req.Value, "ttl": req.TTL})
	})

	// DELETE /cache/:key — delete an entry
	cacheGroup.DELETE("/:key", func(c *gin.Context) {
		key := c.Param("key")
		if err := cacheService.Delete(key); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNoContent, nil)
	})

	// DELETE /cache/ — flush all entries
	cacheGroup.DELETE("/", func(c *gin.Context) {
		cacheService.Flush()
		c.JSON(http.StatusOK, gin.H{"message": "cache flushed"})
	})

	// GET /cache-metrics — get cache metrics
	r.GET("/cache-metrics", func(c *gin.Context) {
		metrics := cacheService.GetMetrics()
		c.JSON(http.StatusOK, metrics)
	})
}

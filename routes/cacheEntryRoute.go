package routes

import (
	"bolt/models"
	"bolt/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterCacheEntryRoutes(r *gin.Engine, cacheEntryService *service.CacheEntryService) {
	cacheGroup := r.Group("/cache")

	cacheGroup.GET("/", func(c *gin.Context) {
		entries, err := cacheEntryService.GetAllCacheEntries()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, entries)
	})

	cacheGroup.GET("/:key", func(c *gin.Context) {
		key := c.Param("key")
		entry, err := cacheEntryService.GetCacheEntryByKey(key)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, entry)
	})

	cacheGroup.POST("/", func(c *gin.Context) {
		var entry models.CacheEntry
		if err := c.ShouldBindJSON(&entry); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := cacheEntryService.CreateCacheEntry(&entry); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, entry)
	})

	cacheGroup.PUT("/:key", func(c *gin.Context) {
		key := c.Param("key")
		var entry models.CacheEntry
		if err := c.ShouldBindJSON(&entry); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		entry.Key = key
		if err := cacheEntryService.UpdateCacheEntry(&entry); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, entry)
	})

	cacheGroup.DELETE("/:key", func(c *gin.Context) {
		key := c.Param("key")
		if err := cacheEntryService.DeleteCacheEntry(key); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNoContent, nil)
	})
}

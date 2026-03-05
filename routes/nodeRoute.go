package routes

import (
	"bolt/models"
	"bolt/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterNodeRoutes(r *gin.Engine, nodeService *service.NodeService) {
	nodeGroup := r.Group("/nodes")

	nodeGroup.GET("/", func(c *gin.Context) {
		nodes, err := nodeService.GetAllNodes()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, nodes)
	})

	nodeGroup.GET("/:id", func(c *gin.Context) {
		id := c.Param("id")
		node, err := nodeService.GetNodeByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
			return
		}
		c.JSON(http.StatusOK, node)
	})

	nodeGroup.POST("/", func(c *gin.Context) {
		var node models.Node
		if err := c.ShouldBindJSON(&node); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := nodeService.CreateNode(&node); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, node)
	})

	nodeGroup.PUT("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var node models.Node
		if err := c.ShouldBindJSON(&node); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		node.ID = id
		if err := nodeService.UpdateNode(&node); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, node)
	})

	nodeGroup.DELETE("/:id", func(c *gin.Context) {
		id := c.Param("id")
		if err := nodeService.DeleteNode(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Node deleted"})
	})
}

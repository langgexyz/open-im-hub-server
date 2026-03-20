package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/langgexyz/open-im-hub-server/internal/store"
)

type DirectoryHandler struct{ nodes *store.NodeStore }

func NewDirectoryHandler(nodes *store.NodeStore) *DirectoryHandler {
	return &DirectoryHandler{nodes: nodes}
}

func (h *DirectoryHandler) List(c *gin.Context) {
	nodes, err := h.nodes.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

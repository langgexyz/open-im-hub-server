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

type nodeResponse struct {
	AppID          string `json:"app_id"`
	Name           string `json:"name"`
	Avatar         string `json:"avatar"`
	Description    string `json:"description"`
	NodeServerAddr string `json:"node_server_addr"`
	NodeWebAddr    string `json:"node_web_addr"`
	AdminUID       string `json:"admin_uid"`
}

func toNodeResponse(n *store.Node) nodeResponse {
	return nodeResponse{
		AppID:          n.AppID,
		Name:           n.Name,
		Avatar:         n.Avatar,
		Description:    n.Description,
		NodeServerAddr: n.NodeServerAddr,
		NodeWebAddr:    n.NodeWebAddr,
		AdminUID:       n.AdminUID,
	}
}

// List GET /nodes — returns all active nodes (status=1)
func (h *DirectoryHandler) List(c *gin.Context) {
	nodes, err := h.nodes.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]nodeResponse, 0, len(nodes))
	for _, n := range nodes {
		resp = append(resp, toNodeResponse(n))
	}
	c.JSON(http.StatusOK, gin.H{"nodes": resp})
}

// Get GET /nodes/:app_id
func (h *DirectoryHandler) Get(c *gin.Context) {
	appID := c.Param("app_id")
	n, err := h.nodes.GetByAppID(appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	c.JSON(http.StatusOK, toNodeResponse(n))
}

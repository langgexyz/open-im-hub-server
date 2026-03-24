package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/langgexyz/open-im-hub-server/internal/handler"
	"github.com/langgexyz/open-im-hub-server/internal/store"
)

func TestDirectoryList(t *testing.T) {
	db := openTestDB(t)
	s, _ := store.New(db)
	// 插入一个 status=1 的节点
	_ = s.Nodes.Upsert(&store.Node{AppID: "n1", AppPublicKey: "0xA", NodeServerAddr: "http://node:8080", NodeWebAddr: "http://node.com", AdminUID: "1", Status: 0})
	_ = s.Nodes.Activate("n1")
	_ = s.Nodes.UpdateProfile("n1", "My Node", "http://avatar.png", "desc")

	gin.SetMode(gin.TestMode)
	h := handler.NewDirectoryHandler(s.Nodes)
	r := gin.New()
	r.GET("/nodes", h.List)
	r.GET("/nodes/:app_id", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/nodes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct{ Nodes []map[string]any }
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.Len(t, resp.Nodes, 1)
	require.Equal(t, "n1", resp.Nodes[0]["app_id"])
	require.Equal(t, "My Node", resp.Nodes[0]["name"])
}

func TestDirectoryGet(t *testing.T) {
	db := openTestDB(t)
	s, _ := store.New(db)
	_ = s.Nodes.Upsert(&store.Node{AppID: "n2", AppPublicKey: "0xB", AdminUID: "2", Status: 0})
	_ = s.Nodes.Activate("n2")

	gin.SetMode(gin.TestMode)
	h := handler.NewDirectoryHandler(s.Nodes)
	r := gin.New()
	r.GET("/nodes/:app_id", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/nodes/n2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.Equal(t, "n2", resp["app_id"])
	require.Equal(t, "2", resp["admin_uid"])
}

package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	hubauth "github.com/langgexyz/open-im-hub-server/internal/auth"
	"github.com/langgexyz/open-im-hub-server/internal/handler"
	"github.com/langgexyz/open-im-hub-server/internal/store"
)

func TestNodeActivate(t *testing.T) {
	// Mock Node Server
	activated := false
	mockNode := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/node/info" {
			json.NewEncoder(w).Encode(map[string]any{"status": "ok", "activated": false})
			return
		}
		if r.URL.Path == "/node/activate" {
			activated = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockNode.Close()

	db := openTestDB(t)
	s, err := store.New(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Exec("DELETE FROM nodes WHERE app_id LIKE 'a1b2c3%'")
		db.Close()
	})
	h := handler.NewActivateHandler(s.Nodes, testPrivHex, "hub.example.com:50051", "https://hub.example.com")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/node/activate", func(c *gin.Context) {
		c.Set(hubauth.ContextUID, "10001")
		h.Activate(c)
	})

	code := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2" // 64 hex chars
	body, _ := json.Marshal(map[string]string{
		"code":             code,
		"node_server_addr": mockNode.URL,
		"node_web_addr":    "http://node.example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/node/activate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.True(t, activated, "node server should have received activation")

	// 验证幂等：再次激活同一 code 应返回 200
	req2 := httptest.NewRequest(http.MethodPost, "/node/activate", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
}

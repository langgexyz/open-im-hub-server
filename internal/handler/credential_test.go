package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	hubauth "github.com/langgexyz/open-im-hub-server/internal/auth"
	"github.com/langgexyz/open-im-hub-server/internal/handler"
	"github.com/stretchr/testify/require"
)

const testPrivHex = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

func setupCredentialRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := handler.NewCredentialHandler(testPrivHex)
	r := gin.New()
	// 模拟 JWT 中间件：直接注入 uid
	r.POST("/user/credential", func(c *gin.Context) {
		c.Set(hubauth.ContextUID, "10001")
		h.Issue(c)
	})
	return r
}

func TestCredentialIssue(t *testing.T) {
	r := setupCredentialRouter(t)
	body, _ := json.Marshal(map[string]string{"target_app_id": "app-abc123"})
	req := httptest.NewRequest(http.MethodPost, "/user/credential", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.NotEmpty(t, resp["credential"])

	// 验证 credential 可以被解析
	pub, err := hubauth.PubKeyFromHex(testPrivHex)
	require.NoError(t, err)
	uid, appID, err := hubauth.VerifyCredential(resp["credential"], pub)
	require.NoError(t, err)
	require.Equal(t, "10001", uid)
	require.Equal(t, "app-abc123", appID)
}

func TestCredentialIssueMissingAppID(t *testing.T) {
	r := setupCredentialRouter(t)
	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/user/credential", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

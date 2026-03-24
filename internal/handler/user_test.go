package handler_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/langgexyz/open-im-hub-server/internal/handler"
	"github.com/langgexyz/open-im-hub-server/internal/store"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	return db
}

func setupUserRouter(t *testing.T) (*gin.Engine, *store.Store) {
	gin.SetMode(gin.TestMode)
	db := openTestDB(t)
	s, err := store.New(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Exec("TRUNCATE TABLE users")
		db.Close()
	})
	privHex := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	h := handler.NewUserHandler(s.Users, privHex)
	r := gin.New()
	r.POST("/user/register", h.Register)
	r.POST("/user/login", h.Login)
	return r, s
}

func TestUserRegister(t *testing.T) {
	r, _ := setupUserRouter(t)
	body, _ := json.Marshal(map[string]string{"email": "alice@example.com", "password": "secret123"})
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.NotEmpty(t, resp["uid"])
	require.NotEmpty(t, resp["hub_token"])
}

func TestUserRegisterDuplicate(t *testing.T) {
	r, _ := setupUserRouter(t)
	body, _ := json.Marshal(map[string]string{"email": "dup@example.com", "password": "pass12"})
	req1 := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusOK, w1.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusConflict, w2.Code)
}

func TestUserLogin(t *testing.T) {
	r, _ := setupUserRouter(t)
	// 先注册
	body, _ := json.Marshal(map[string]string{"email": "bob@example.com", "password": "mypassword"})
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	// 再登录
	req2 := httptest.NewRequest(http.MethodPost, "/user/login", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req2)
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.NotEmpty(t, resp["hub_token"])
}

func TestUserLoginWrongPassword(t *testing.T) {
	r, _ := setupUserRouter(t)
	body, _ := json.Marshal(map[string]string{"email": "carol@example.com", "password": "correct"})
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	wrong, _ := json.Marshal(map[string]string{"email": "carol@example.com", "password": "wrong"})
	req2 := httptest.NewRequest(http.MethodPost, "/user/login", bytes.NewReader(wrong))
	req2.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req2)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

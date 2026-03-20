package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/langgexyz/open-im-hub-server/internal/handler"
)

func TestRegisterMissingNodeParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewRegisterHandler(nil, "")
	r.GET("/register", h.Register)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/register", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRegisterInvalidNodeURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewRegisterHandler(nil, "")
	r.GET("/register", h.Register)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/register?node=not-a-url", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

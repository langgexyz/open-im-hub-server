package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	hubauth "github.com/langgexyz/open-im-hub-server/internal/auth"
	"github.com/langgexyz/open-im-hub-server/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const (
	hubTokenTTL = 7 * 24 * 3600 // 7 days
	minPassword = 6
)

type UserHandler struct {
	users     *store.UserStore
	jwtSecret string // = HUB_PRIVATE_KEY hex string
}

func NewUserHandler(users *store.UserStore, hubPrivKeyHex string) *UserHandler {
	return &UserHandler{users: users, jwtSecret: hubPrivKeyHex}
}

// Register POST /user/register { email, password }
func (h *UserHandler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Password) < minPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("password must be at least %d characters", minPassword)})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	uid, err := h.users.Create(req.Email, string(hash))
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create user failed"})
		return
	}

	uidStr := fmt.Sprintf("%d", uid)
	token, err := hubauth.IssueHubToken(uidStr, req.Email, h.jwtSecret, hubTokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"uid": uidStr, "hub_token": token})
}

// Login POST /user/login { email, password }
func (h *UserHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	uidStr := fmt.Sprintf("%d", user.ID)
	token, err := hubauth.IssueHubToken(uidStr, user.Email, h.jwtSecret, hubTokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"uid": uidStr, "hub_token": token})
}

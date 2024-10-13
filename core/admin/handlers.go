package admin

import (
	"github.com/dryack/gDiceRoll/core/session"
	"github.com/dryack/gDiceRoll/core/user"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	userManager    *user.UserManager
	sessionManager *session.SessionManager
}

func NewAdminHandler(userManager *user.UserManager, sessionManager *session.SessionManager) (*AdminHandler, error) {
	return &AdminHandler{
		userManager:    userManager,
		sessionManager: sessionManager,
	}, nil
}

func (h *AdminHandler) LoginPage(c *gin.Context) {
	// log.Println("Rendering login page")
	c.HTML(http.StatusOK, "layout.html", gin.H{
		"content": "login.html",
	})
}

func (h *AdminHandler) Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	user, err := h.userManager.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"error": "Invalid credentials",
		})
		return
	}

	match, err := h.userManager.VerifyPassword(user, password)
	if err != nil || !match {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"error": "Invalid credentials",
		})
		return
	}

	// Create a new session with JWTs
	session, accessToken, refreshToken, err := h.sessionManager.CreateSession(c.Request.Context(), user.ID)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"error": "Failed to create session",
		})
		return
	}

	// Set refresh token as an HTTP-only cookie
	c.SetCookie("refresh_token", refreshToken, int(time.Until(session.ExpiresAt).Seconds()), "/", "", false, true)

	// Set access token as a cookie (this might not be the best practice for security)
	c.SetCookie("access_token", accessToken, int(15*time.Minute.Seconds()), "/", "", false, false)

	c.Redirect(http.StatusSeeOther, "/admin/dashboard")
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	log.Printf("Dashboard function called. Method: %s, Path: %s", c.Request.Method, c.Request.URL.Path)

	c.HTML(http.StatusOK, "layout.html", gin.H{
		"content": "dashboard.html",
		"message": "Welcome to the dashboard!",
	})
}

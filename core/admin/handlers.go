package admin

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct{}

func NewAdminHandler() (*AdminHandler, error) {
	return &AdminHandler{}, nil
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
	log.Printf("Login attempt: username=%s, password=%s", username, password)

	if username == "admin" && password == "password" {
		// log.Println("Login successful, redirecting to dashboard")
		c.Redirect(http.StatusSeeOther, "/admin/dashboard")
	} else {
		// log.Println("Login failed, re-rendering login page with error")
		c.HTML(http.StatusUnauthorized, "layout.html", gin.H{
			"content": "login.html",
			"error":   "Invalid credentials",
		})
	}
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	// log.Println("Rendering dashboard page")
	c.HTML(http.StatusOK, "layout.html", gin.H{
		"content": "dashboard.html",
		"message": "This is a test message from the dashboard handler",
	})
}

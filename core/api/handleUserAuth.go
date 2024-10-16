package api

import (
	"github.com/dryack/gDiceRoll/core/user"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

func (s *Server) handleRegister(c *gin.Context) {
	var registerData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&registerData); err != nil {
		log.Printf("Error binding JSON: %v", err)
		log.Printf("Request body: %v", c.Request.Body)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Attempting to create user with username: %s", registerData.Username)

	newUser, err := s.userManager.CreateUser(c.Request.Context(), registerData.Username, registerData.Password, user.User)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	log.Printf("User created successfully with ID: %d", newUser.ID)
	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully", "user_id": newUser.ID})
}

func (s *Server) handleLogin(c *gin.Context) {
	var loginData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := s.userManager.GetUserByUsername(c.Request.Context(), loginData.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	match, err := s.userManager.VerifyPassword(user, loginData.Password)
	if err != nil || !match {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Create a new session with JWTs
	session, accessToken, refreshToken, err := s.sessionManager.CreateSession(c.Request.Context(), user.ID)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Set refresh token as an HTTP-only cookie
	c.SetCookie("refresh_token", refreshToken, int(time.Until(session.ExpiresAt).Seconds()), "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Login successful",
		"access_token": accessToken,
		"session_id":   session.ID,
	})
}

func (s *Server) handleLogout(c *gin.Context) {
	sessionID, err := c.Cookie("session_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No session found"})
		return
	}

	err = s.sessionManager.DeleteSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete session"})
		return
	}

	c.SetCookie("session_id", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

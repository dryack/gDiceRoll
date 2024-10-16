package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func (s *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for session ID cookie first
		sessionID, err := c.Cookie("session_id")
		if err == nil {
			// Session ID found, verify it
			session, err := s.sessionManager.GetSession(c.Request.Context(), sessionID)
			if err == nil {
				// Session found, check if it's expired
				if time.Now().After(session.ExpiresAt) {
					// Session has expired, try to refresh
					refreshToken, _ := c.Cookie("refresh_token")
					if refreshToken != "" {
						newSession, newAccessToken, newRefreshToken, err := s.sessionManager.RefreshSession(c.Request.Context(), refreshToken)
						if err == nil {
							// Successfully refreshed, update cookies and continue
							c.SetCookie("session_id", newSession.ID, int(time.Until(newSession.ExpiresAt).Seconds()), "/", "", false, true)
							c.SetCookie("access_token", newAccessToken, int(15*time.Minute.Seconds()), "/", "", false, false)
							c.SetCookie("refresh_token", newRefreshToken, int(24*time.Hour.Seconds()), "/", "", false, true)
							c.Set("user_id", newSession.UserID)
							c.Next()
							return
						}
					}
				} else {
					// Session is valid and not expired
					c.Set("user_id", session.UserID)
					c.Next()
					return
				}
			}
		}

		// If session check failed, try JWT
		accessToken, err := c.Cookie("access_token")
		if err == nil {
			claims, err := s.sessionManager.VerifyAccessToken(accessToken)
			if err == nil {
				// JWT is valid
				c.Set("user_id", claims.UserID)
				c.Next()
				return
			}
		}

		// If we've reached here, authentication has failed
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		c.Abort()
	}
}

package api

import (
	"github.com/dryack/gDiceRoll/core/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (s *Server) handleEncodeExpression(c *gin.Context) {
	expression := c.Query("expression")
	if expression == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing dice expression"})
		return
	}

	encoded := utils.EncodeExpression(expression)

	c.JSON(http.StatusOK, gin.H{
		"original": expression,
		"encoded":  encoded,
	})
}

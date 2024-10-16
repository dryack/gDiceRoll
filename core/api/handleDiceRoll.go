package api

import (
	"github.com/dryack/gDiceRoll/core/dsl"
	"github.com/dryack/gDiceRoll/core/utils"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

func (s *Server) handleDiceRoll(c *gin.Context) {
	start := time.Now()

	encodedExpression := c.Query("expr")
	if encodedExpression == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing encoded dice expression"})
		return
	}

	expression, err := utils.DecodeExpression(encodedExpression)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := dsl.NewPostgresDB(s.db)

	// Try to get from cache
	_, err = s.cache.Get(c.Request.Context(), expression)
	if err == nil {
		// Cache hit
		roll, err := dsl.Parse(c.Request.Context(), expression, s.cache, db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		s.sendRollResponse(c, roll, expression, time.Since(start), "cache")
		return
	}

	// Cache miss, try database
	dbResult, err := db.Get(c.Request.Context(), expression)
	if err == nil {
		// Database hit, add to cache
		err = s.cache.Set(c.Request.Context(), expression, dbResult)
		if err != nil {
			log.Printf("Error setting cache: %v", err)
		}
		roll, err := dsl.Parse(c.Request.Context(), expression, s.cache, db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		s.sendRollResponse(c, roll, expression, time.Since(start), "database")
		return
	}

	// Both cache and database miss, calculate
	roll, err := dsl.Parse(c.Request.Context(), expression, s.cache, db)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Add to cache
	err = s.cache.Set(c.Request.Context(), expression, &dsl.CachedResult{Statistics: roll.Statistics})
	if err != nil {
		log.Printf("Error setting cache: %v", err)
	}

	s.sendRollResponse(c, roll, expression, time.Since(start), "calculation")
}

func (s *Server) sendRollResponse(c *gin.Context, roll *dsl.Result, expression string, duration time.Duration, source string) {
	c.JSON(http.StatusOK, gin.H{
		"expression": expression,
		"result":     roll.Value,
		"breakdown":  roll.Breakdown,
		"statistics": gin.H{
			"min":               roll.Statistics.Min,
			"max":               roll.Statistics.Max,
			"mean":              roll.Statistics.Mean,
			"standardDeviation": roll.Statistics.StandardDeviation,
			"variance":          roll.Statistics.Variance,
			"skewness":          roll.Statistics.Skewness,
			"kurtosis":          roll.Statistics.Kurtosis,
			"percentiles":       roll.Statistics.Percentiles,
		},
		"source":           source,
		"request_duration": utils.FormatDuration(duration),
	})
}

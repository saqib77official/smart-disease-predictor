package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/saqibullah/smart-disease-predictor-backend/api"
)

func main() {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://smart-disease-predictor-509.web.app" // Firebase URL
	}

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", frontendURL)
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API Endpoints
	r.POST("/predict", api.PredictHandler)
	// Comment out for now: r.POST("/extract", api.ExtractHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}

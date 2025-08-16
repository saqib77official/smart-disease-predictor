package api

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

type Input struct {
	Pregnancies              int     `json:"pregnancies"`
	Glucose                  float64 `json:"glucose"`
	BloodPressure            float64 `json:"bloodPressure"`
	SkinThickness            float64 `json:"skinThickness"`
	Insulin                  float64 `json:"insulin"`
	BMI                      float64 `json:"bmi"`
	DiabetesPedigreeFunction float64 `json:"diabetesPedigreeFunction"`
	Age                      int     `json:"age"`
}

func PredictHandler(c *gin.Context) {
	var input Input
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client := resty.New()

	// Check environment variable for ML URL
	mlURL := os.Getenv("ML_URL")
	if mlURL == "" {
		// Default for local testing
		mlURL = "https://smart-disease-ml.onrender.com"
	}

	// Structure to store prediction result
	var result struct {
		Prediction string `json:"prediction"`
	}

	// Make POST request to ML service
	resp, err := client.R().
		SetBody(input).
		SetResult(&result).
		Post(mlURL + "/predict")

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to ML server: " + err.Error()})
		return
	}
	if resp.StatusCode() != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("ML server error: status %d", resp.StatusCode())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"prediction": result.Prediction})
}

func ExtractHandler(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image uploaded: " + err.Error()})
		return
	}

	tempPath := "temp.jpg"
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image: " + err.Error()})
		return
	}
	defer os.Remove(tempPath)

	// Verify Tesseract installation
	if _, err := exec.LookPath("tesseract"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tesseract not installed or not in PATH: " + err.Error()})
		return
	}

	// Run OCR with Tesseract
	outputFile := "output"
	cmd := exec.Command("tesseract", tempPath, outputFile, "-l", "eng")
	if err := cmd.Run(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCR failed: " + err.Error()})
		return
	}

	// Read OCR output
	text, err := os.ReadFile(outputFile + ".txt")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read OCR output: " + err.Error()})
		return
	}
	defer os.Remove(outputFile + ".txt")

	ocrText := strings.TrimSpace(string(text))
	if ocrText == "" {
		c.JSON(http.StatusOK, gin.H{"extracted": map[string]float64{}})
		return
	}

	// Extract values using regex
	extracted := make(map[string]float64)
	patterns := map[string]*regexp.Regexp{
		"Pregnancies":              regexp.MustCompile(`(?i)Pregnancies:\s*(\d+)`),
		"Glucose":                  regexp.MustCompile(`(?i)Glucose:\s*(\d+\.?\d*)`),
		"BloodPressure":            regexp.MustCompile(`(?i)BloodPressure:\s*(\d+\.?\d*)`),
		"SkinThickness":            regexp.MustCompile(`(?i)SkinThickness:\s*(\d+\.?\d*)`),
		"Insulin":                  regexp.MustCompile(`(?i)Insulin:\s*(\d+\.?\d*)`),
		"BMI":                      regexp.MustCompile(`(?i)BMI:\s*(\d+\.?\d*)`),
		"DiabetesPedigreeFunction": regexp.MustCompile(`(?i)DiabetesPedigreeFunction:\s*(\d+\.?\d*)`),
		"Age":                      regexp.MustCompile(`(?i)Age:\s*(\d+)`),
	}

	for key, re := range patterns {
		if match := re.FindStringSubmatch(ocrText); len(match) > 1 {
			if f, err := strconv.ParseFloat(match[1], 64); err == nil {
				extracted[key] = f
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"extracted": extracted})
}

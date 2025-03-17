// examples/main.go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/SIM-MBKM/filestorage/storage" // Import storage package directly
	"github.com/gin-gonic/gin"
)

func main() {

	// Load configuration from environment variables
	config, err := storage.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create cache for token management
	cache := storage.NewMemoryCache()

	// Create token manager
	tokenManager := storage.NewCacheTokenManager(config, cache)

	// Initialize file storage manager
	fs := storage.NewFileStorageManager(config, tokenManager)

	// Set up Gin router for examples
	r := gin.Default()

	// Simple upload endpoint
	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Upload file
		result, err := fs.Upload(file)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, result)
	})

	// Example 1: Upload to Google Cloud Storage
	r.POST("/gcs/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Upload to GCS
		result, err := fs.GcsUpload(file, "", "", "")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, result)
	})

	// Example 2: Upload to AWS S3
	r.POST("/s3/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Upload to S3
		result, err := fs.AwsUpload(file, "examples", "")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, result)
	})

	// Example 3: Get temporary link for GCS file
	r.GET("/gcs/link", func(c *gin.Context) {
		// fileId := c.Param("fileId")
		// get from query
		fileId := c.Query("fileId")

		// Create temporary link that expires in 1 hour
		expiry := time.Now().Add(1 * time.Hour)
		result, err := fs.GcsGetTemporaryPublicLink(fileId, expiry, "", "")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, result)
	})

	// Example 4: Get file info from S3
	r.GET("/s3/info/:fileId", func(c *gin.Context) {
		fileId := c.Param("fileId")

		result, err := fs.AwsGetFileById(fileId, "")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, result)
	})

	// gcs file info
	r.GET("/gcs/info", func(c *gin.Context) {
		// fileId := c.Param("fileId")
		// get from query
		fileId := c.Query("fileId")

		result, err := fs.GcsGetFileById(fileId, "", "")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, result)
	})

	// Example 5: Delete file from GCS
	r.DELETE("/gcs/delete", func(c *gin.Context) {
		// fileId := c.Param("fileId")
		// get from query
		fileId := c.Query("fileId")

		result, err := fs.GcsDelete(fileId, "", "")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, result)
	})

	// Example 6: Upload base64 file
	r.POST("/upload-base64", func(c *gin.Context) {
		var request struct {
			Filename      string `json:"filename"`
			Extension     string `json:"extension"`
			MimeType      string `json:"mime_type"`
			Base64Content string `json:"base64_content"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		result, err := fs.UploadBase64File(
			request.Filename,
			request.Extension,
			request.MimeType,
			request.Base64Content,
		)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, result)
	})

	// Example 7: Get temporary link for S3 file
	r.GET("/s3/link/:fileId", func(c *gin.Context) {
		fileId := c.Param("fileId")

		// Create temporary link that expires in 30 minutes
		expiry := time.Now().Add(30 * time.Minute)
		result, err := fs.AwsGetTemporaryPublicLink(fileId, expiry, "")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, result)
	})

	// Example 8: Delete file from S3
	r.DELETE("/s3/delete/:fileId", func(c *gin.Context) {
		fileId := c.Param("fileId")

		result, err := fs.AwsDelete(fileId, "")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, result)
	})

	// Start the example server
	fmt.Println("Starting example server on :8000")
	r.Run(":8000")
}

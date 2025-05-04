// route/route.go
package route

import (
	"time"

	"github.com/SIM-MBKM/filestorage/middleware"
	"github.com/SIM-MBKM/filestorage/storage"
	"github.com/SIM-MBKM/mod-service/src/helpers"
	"github.com/gin-gonic/gin"
)

// SetupRouter configures all routes and returns the router
func SetupRouter(fs *storage.FileStorageManager, security *helpers.Security, secretKey string, expireSeconds int64) *gin.Engine {
	// Set up Gin router
	r := gin.Default()

	// Add CORS middleware
	r.Use(middleware.CORS())

	// Add security middleware
	// r.Use(middleware.AccessKeyMiddleware(security, secretKey, expireSeconds))

	// Create a route group for file service
	fileService := r.Group("/file-service/api/v1")
	{
		// Simple upload endpoint
		fileService.POST("/upload", func(c *gin.Context) {
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
		fileService.POST("/gcs/upload", func(c *gin.Context) {
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
		fileService.POST("/s3/upload", func(c *gin.Context) {
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
		fileService.GET("/gcs/link", func(c *gin.Context) {
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
		fileService.GET("/s3/info/:fileId", func(c *gin.Context) {
			fileId := c.Param("fileId")

			result, err := fs.AwsGetFileById(fileId, "")
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, result)
		})

		// GCS file info
		fileService.GET("/gcs/info", func(c *gin.Context) {
			fileId := c.Query("fileId")

			result, err := fs.GcsGetFileById(fileId, "", "")
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, result)
		})

		// Example 5: Delete file from GCS
		fileService.DELETE("/gcs/delete", func(c *gin.Context) {
			fileId := c.Query("fileId")

			result, err := fs.GcsDelete(fileId, "", "")
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, result)
		})

		// Example 6: Upload base64 file
		fileService.POST("/upload-base64", func(c *gin.Context) {
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
		fileService.GET("/s3/link/:fileId", func(c *gin.Context) {
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
		fileService.DELETE("/s3/delete/:fileId", func(c *gin.Context) {
			fileId := c.Param("fileId")

			result, err := fs.AwsDelete(fileId, "")
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, result)
		})
	}

	return r
}

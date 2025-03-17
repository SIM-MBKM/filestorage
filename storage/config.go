// pkg/storage/config.go

package storage

import (
	"os"

	"github.com/joho/godotenv"
)

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	godotenv.Load()

	config := &Config{
		// Default REST API Storage Settings
		HostURI:                os.Getenv("FILE_STORAGE_HOST_URI"),
		AuthorizationServerURI: os.Getenv("FILE_STORAGE_AUTHORIZATION_SERVER_URI"),
		ClientID:               os.Getenv("FILE_STORAGE_CLIENT_ID"),
		ClientSecret:           os.Getenv("FILE_STORAGE_CLIENT_SECRET"),

		// AWS S3 Configuration
		AWSKey:    os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecret: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AWSRegion: os.Getenv("AWS_DEFAULT_REGION"),
		AWSBucket: os.Getenv("AWS_BUCKET"),

		// Google Cloud Storage Configuration
		GCSKeyPath:   os.Getenv("GOOGLE_KEY_PATH"),
		GCSProjectID: os.Getenv("GOOGLE_PROJECT_ID"),
		GCSBucket:    os.Getenv("GOOGLE_BUCKET"),
	}

	return config, nil
}

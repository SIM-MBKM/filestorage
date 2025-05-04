package main

import (
	"fmt"
	"log"

	"github.com/SIM-MBKM/filestorage/route"
	"github.com/SIM-MBKM/filestorage/storage"
	"github.com/SIM-MBKM/mod-service/src/helpers"
)

func main() {
	// Load environment variables
	helpers.LoadEnv()
	secretKey := helpers.GetEnv("APP_KEY", "secret")

	// Configure security middleware
	security := helpers.NewSecurity("sha256", secretKey, "aes")
	expireSeconds := int64(9999)

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

	// Set up router with all routes and middleware
	r := route.SetupRouter(fs, security, secretKey, expireSeconds)

	// Start the server
	fmt.Println("Starting server on :8000")
	r.Run(":8000")
}

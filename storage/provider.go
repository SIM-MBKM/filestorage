package storage

import (
	"sync"
)

// FileStorageServiceProvider manages the file storage service
type FileStorageServiceProvider struct {
	config       *Config
	fileStorage  *FileStorageManager
	tokenManager TokenManager
	memoryCache  *MemoryCache
	once         sync.Once
}

// NewFileStorageServiceProvider creates a new service provider
func NewFileStorageServiceProvider(config *Config) *FileStorageServiceProvider {
	return &FileStorageServiceProvider{
		config: config,
	}
}

// FileStorage provides a singleton instance of FileStorageManager
func (p *FileStorageServiceProvider) FileStorage() *FileStorageManager {
	p.once.Do(func() {
		p.memoryCache = NewMemoryCache()
		p.tokenManager = NewCacheTokenManager(p.config, p.memoryCache)
		p.fileStorage = NewFileStorageManager(p.config, p.tokenManager)
	})

	return p.fileStorage
}

// pkg/storage/token_manager.go

package storage

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CacheTokenManager struct {
	config      *Config
	cache       Cache
	tokenKey    string
	tokenExpiry time.Duration
}

// Cache interface for storing tokens
type Cache interface {
	Get(key string) (string, bool)
	Set(key string, value string, expiry time.Duration)
	Delete(key string)
	Has(key string) bool
}

// TokenResponse represents the response from the token server
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// NewCacheTokenManager creates a new token manager that uses a cache
func NewCacheTokenManager(config *Config, cache Cache) *CacheTokenManager {
	return &CacheTokenManager{
		config:      config,
		cache:       cache,
		tokenKey:    "access_token",
		tokenExpiry: 3550 * time.Second, // Almost 1 hour
	}
}

// GenerateToken generates a new token
func (t *CacheTokenManager) GenerateToken() (string, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", t.config.ClientID)
	data.Set("client_secret", t.config.ClientSecret)

	client := &http.Client{}
	req, err := http.NewRequest("POST", t.config.AuthorizationServerURI+"/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return "", err
	}

	// Store token in cache
	t.cache.Set(t.tokenKey, tokenResp.AccessToken, time.Duration(tokenResp.ExpiresIn)*time.Second)

	return tokenResp.AccessToken, nil
}

// GetToken retrieves the current token
func (t *CacheTokenManager) GetToken() (string, error) {
	token, exists := t.cache.Get(t.tokenKey)
	if !exists {
		return t.GenerateToken()
	}
	return token, nil
}

// HasToken checks if a token exists
func (t *CacheTokenManager) HasToken() bool {
	return t.cache.Has(t.tokenKey)
}

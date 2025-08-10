package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const pocketbaseAuthRefreshPath = "/api/collections/users/auth-refresh"

type tokenCacheEntry struct {
	expiresAt time.Time
}

var (
	tokenCache    = make(map[string]tokenCacheEntry)
	tokenCacheMu  sync.RWMutex
	tokenCacheTTL = time.Minute * 1 // cache for 5 minutes
)

func isTokenCachedValid(token string) bool {
	tokenCacheMu.RLock()
	entry, exists := tokenCache[token]
	tokenCacheMu.RUnlock()
	return exists && time.Now().Before(entry.expiresAt)
}

func cacheToken(token string, ttl time.Duration) {
	tokenCacheMu.Lock()
	tokenCache[token] = tokenCacheEntry{
		expiresAt: time.Now().Add(ttl),
	}
	tokenCacheMu.Unlock()
}

func checkAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbAuthToken := r.Header.Get("pb_auth")
		if pbAuthToken != "" && isTokenCachedValid(pbAuthToken) {
			next.ServeHTTP(w, r)
			return
		}

		pbServerURL, err := url.Parse(pocketbaseBaseURL)
		if err != nil {
			log.Printf("Pocketbase Auth Check - Error parsing pocketbase url: %s", err)
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
		pbServerURL = pbServerURL.JoinPath(pocketbaseAuthRefreshPath)

		authRequest, err := http.NewRequest(http.MethodPost, pbServerURL.String(), nil)
		if err != nil {
			log.Printf("Pocketbase Auth Check - Error creating auth request: %s", err)
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
		authRequest.Header.Add("Authorization", pbAuthToken)
		client := http.Client{}
		authResponse, err := client.Do(authRequest)
		if err != nil {
			log.Printf("Pocketbase Auth Check - Error sending auth request: %s", err)
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
		defer authResponse.Body.Close()
		if authResponse.StatusCode != http.StatusOK {
			body, err := io.ReadAll(authResponse.Body)
			if err != nil {
				log.Printf("Pocketbase Auth Check - Error reading response body: %s", err)
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
				return
			}
			var pbResponseData map[string]any

			err = json.Unmarshal(body, &pbResponseData)
			if err != nil {
				log.Printf("Pocketbase Auth Check - Error reading response body: %s", err)
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
				return
			}

			statusMessage := pbResponseData["message"].(string)
			http.Error(w, statusMessage, authResponse.StatusCode)
			log.Printf("Unauthorized request to %s. Pocketbase response status: %s", r.URL, authResponse.Status)
			return
		} else {
			cacheToken(pbAuthToken, tokenCacheTTL)
		}

		next.ServeHTTP(w, r)
	})
}

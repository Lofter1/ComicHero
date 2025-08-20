package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	rdb *redis.Client
	ctx = context.Background()
	mu  sync.RWMutex
)

type CacheEntry struct {
	Data      []byte
	Header    http.Header
	Timestamp int64
}

var (
	apiCache   = make(map[string]CacheEntry)
	mediaCache = make(map[string]CacheEntry)
	cacheTTL   = int64(60 * 60 * 24) // cache for 1 day
)

func InitCaching() {
	if redisAddr == "" {
		log.Printf("No Redis Address set, using in-memory cache\n")
	} else {
		err := initRedis()
		if err != nil {
			log.Printf("Failed to connect to Redis: %v\n Fall back to in-memory cache", err)
			rdb = nil
		} else {
			log.Printf("Using Redis cache at %s\n", redisAddr)
		}
	}
}

func SaveCache(key string, entry CacheEntry, ttl time.Duration, media bool) {
	if rdb != nil {
		saveCacheRedis(key, entry, ttl, media)
	} else {
		saveCacheInMemory(key, entry, media)
	}
}

func saveCacheRedis(key string, entry CacheEntry, ttl time.Duration, media bool) {
	b, _ := json.Marshal(entry)
	if err := rdb.Set(ctx, redisKey(key, media), b, ttl).Err(); err != nil {
		log.Printf("Redis set error: %v", err)
	}
}

func saveCacheInMemory(key string, entry CacheEntry, media bool) {
	mu.Lock()
	defer mu.Unlock()
	if media {
		mediaCache[key] = entry
	} else {
		apiCache[key] = entry
	}
}

func GetCache(key string, media bool) (*CacheEntry, bool) {
	if rdb != nil {
		return getCacheRedis(key, media)
	} else {
		return getCacheInMemory(key, media)
	}
}

func getCacheRedis(key string, media bool) (*CacheEntry, bool) {
	val, err := rdb.Get(ctx, redisKey(key, media)).Result()
	if err == redis.Nil {
		return nil, false
	}
	if err != nil {
		log.Printf("Redis get error: %v", err)
		return nil, false
	}
	var entry CacheEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		log.Printf("Redis unmarshal error: %v", err)
		return nil, false
	}
	if isExpired(entry) {
		return nil, false
	}
	return &entry, true
}

func getCacheInMemory(key string, media bool) (*CacheEntry, bool) {
	mu.RLock()
	defer mu.RUnlock()
	var entry CacheEntry
	var found bool
	if media {
		entry, found = mediaCache[key]
	} else {
		entry, found = apiCache[key]
	}
	if !found || isExpired(entry) {
		return nil, false
	}
	return &entry, true
}

func redisKey(key string, media bool) string {
	if media {
		return "media:" + key
	}
	return "api:" + key
}

func initRedis() error {
	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	return rdb.Ping(ctx).Err()
}

func isExpired(entry CacheEntry) bool {
	return time.Now().Unix()-entry.Timestamp > cacheTTL
}

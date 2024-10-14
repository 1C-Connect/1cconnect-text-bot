package database

import (
	"time"

	"connect-text-bot/internal/logger"

	"github.com/allegro/bigcache/v3"
	"github.com/gin-gonic/gin"
)

func ConnectInMemoryCache() *bigcache.BigCache {
	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig(2 * time.Hour))
	if err != nil {
		logger.Crit(err)
	}
	return cache
}

func InjectInMemoryCache(key string, cache *bigcache.BigCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(key, cache)
	}
}

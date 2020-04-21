package database

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v7"
)

type (
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
	}
)

const (
	PREFIX = "demo_bot:"
	EXPIRE = 7776000
)

func Connect(d Redis) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         d.Addr,     // use default Addr
		Password:     d.Password, // no password set
		DB:           0,          // use default DB
		MinIdleConns: 3,
	})
}

func Inject(key string, redis *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(key, redis)
	}
}

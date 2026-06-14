package database

import (
	"context"
	"fmt"
	"gin-postgre-project/config"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func ConnectRedis() {
	addr := fmt.Sprintf("%s:%s", config.AppConfig.RedisHost, config.AppConfig.RedisPort)
	fmt.Printf("尝试连接 Redis: %s\n", addr)

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
		Network:  "tcp", // 强制使用 TCP IPv4
	})

	ctx := context.Background() // 创建一个上下文, 用于与 Redis 交互

	// 尝试连接 Redis，如果失败则警告但不退出程序
	if err := RedisClient.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️  Redis 连接失败: %v, 地址: %s", err, addr)
		log.Printf("⚠️  程序将继续运行，但缓存功能将不可用")
		RedisClient = nil // 设置为 nil 表示 Redis 不可用
		return
	}
	fmt.Println("✅ Redis 连接成功") // 如果连接成功, 则打印连接成功
}

// CacheGet 安全获取缓存，Redis不可用时返回空结果
func CacheGet(ctx context.Context, key string) *redis.StringCmd {
	if RedisClient == nil {
		return redis.NewStringResult("", redis.Nil)
	}
	return RedisClient.Get(ctx, key)
}

// CacheSet 安全写入缓存，Redis不可用时静默跳过
func CacheSet(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	if RedisClient == nil {
		return
	}
	RedisClient.Set(ctx, key, value, ttl)
}

// CacheDel 安全删除缓存，Redis不可用时静默跳过
func CacheDel(ctx context.Context, keys ...string) {
	if RedisClient == nil {
		return
	}
	RedisClient.Del(ctx, keys...)
}

// CacheFlushDB 安全清空数据库，Redis不可用时静默跳过
func CacheFlushDB(ctx context.Context) {
	if RedisClient == nil {
		return
	}
	RedisClient.FlushDB(ctx)
}

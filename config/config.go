package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	RedisHost  string
	RedisPort  string
	JWTSecret  string
	JWTExpire  string
}

var AppConfig Config

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("没有发现 .env 文件")
	}

	AppConfig = Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "34189"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "abcd001002"),
		DBName:     getEnv("DB_NAME", "machine_info"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
		JWTSecret:  getEnv("JWT_SECRET", "fallback-secret-change-me"),
		JWTExpire:  getEnv("JWT_EXPIRE", "72"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

package config

import (
	"fmt"
	"os"
)

// Config ortam yapılandırmalarını tutar
type Config struct {
	AppEnv string
	Port   string
	DBHost string
	DBPort string
	DBUser string
	DBPass string
	DBName string
}

// yardımcı fonksiyon: ortam değişkeni yoksa default değeri döner
func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// LoadConfig tüm yapılandırmayı yükler
func LoadConfig() *Config {
	return &Config{
		AppEnv: getEnv("APP_ENV", "development"),
		Port:   getEnv("PORT", "8080"),
		DBHost: getEnv("DB_HOST", "localhost"),
		DBPort: getEnv("DB_PORT", "5432"),
		DBUser: getEnv("DB_USER", "ilhan"),
		DBPass: getEnv("DB_PASS", "password"),
		DBName: getEnv("DB_NAME", "paymentdb"),
	}
}

// GetDSN veritabanı bağlantı URL'sini döner
func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort, c.DBName,
	)
}

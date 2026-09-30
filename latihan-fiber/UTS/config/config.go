package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config menyimpan seluruh konfigurasi aplikasi.
type Config struct {
	AppPort      string
	AppEnv       string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	DBSSLMode    string
	DBMaxConns   int
	JWTSecret    string
	JWTIssuer    string
	JWTExpiresIn int
}

var AppConfig Config

// LoadConfig memuat environment variables dari file .env.
func LoadConfig() {
	if err := godotenv.Load(); err != nil {
		// Log info jika .env tidak ditemukan, fallback ke system environment variables
		log.Println(".env file tidak ditemukan, membaca dari environment system")
	}

	AppConfig = Config{
		AppPort:      getEnv("APP_PORT", "3000"),
		AppEnv:       getEnv("APP_ENV", "development"),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "postgres"),
		DBPassword:   getEnv("DB_PASSWORD", "123456"),
		DBName:       getEnv("DB_NAME", "siakad_mini"),
		DBSSLMode:    getEnv("DB_SSLMODE", "disable"),
		DBMaxConns:   getEnvInt("DB_MAX_CONNS", 10),
		JWTSecret:    getEnv("JWT_SECRET", "supersecretjwtkeywithmorethan32bytes123456!"),
		JWTIssuer:    getEnv("JWT_ISSUER", "siakad-mini"),
		JWTExpiresIn: getEnvInt("JWT_EXPIRES_IN", 3600),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

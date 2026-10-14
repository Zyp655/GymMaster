package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type (
	Config struct {
		Env    string
		Server ServerCfg
		DB     DbCfg
		JWT    JwtCfg
		Seed   SeedCfg
		Upload UploadCfg
		Mail   MailCfg
		Cron   CronCfg
	}

	UploadCfg struct {
		Dir       string // thư mục lưu file, được serve tại /uploads
		PublicURL string // tiền tố URL trả về cho client, vd http://localhost:8000/uploads
		MaxSizeMB int
	}

	MailCfg struct {
		Host string
		Port int
		User string
		Pass string
		From string
	}

	CronCfg struct {
		Enabled bool
	}

	ServerCfg struct {
		Port        int
		CorsOrigins []string
	}

	DbCfg struct {
		Host            string
		Port            int
		User            string
		Password        string
		Name            string
		SSLMode         string
		MaxOpenConns    int
		MaxIdleConns    int
		ConnMaxLifetime time.Duration
		LogSQL          bool
	}

	JwtCfg struct {
		Secret     string
		Issuer     string
		AccessTTL  time.Duration
		RefreshTTL time.Duration
	}

	SeedCfg struct {
		AdminEmail    string
		AdminPassword string
	}
)

var (
	cfg  *Config
	once sync.Once
)

// GetConfig trả về cấu hình (singleton), đọc từ .env + biến môi trường.
func GetConfig() *Config {
	once.Do(func() {
		cfg = load()
	})
	return cfg
}

func load() *Config {
	_ = godotenv.Load()

	c := &Config{
		Env: getEnv("APP_ENV", "dev"),
		Server: ServerCfg{
			Port:        parseInt("SERVER_PORT", 8000),
			CorsOrigins: corsHosts(getEnv("CORS_ORIGINS", "localhost:3000,localhost:3001")),
		},
		DB: DbCfg{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            parseInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "gymmaster"),
			Password:        getEnv("DB_PASSWORD", "gymmaster"),
			Name:            getEnv("DB_NAME", "gymmaster"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    parseInt("DB_MAX_OPEN_CONNS", 20),
			MaxIdleConns:    parseInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: parseDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
			LogSQL:          parseBool("DB_LOG_SQL", false),
		},
		JWT: JwtCfg{
			Secret:     getEnv("JWT_SECRET", ""),
			Issuer:     getEnv("JWT_ISSUER", "gymmaster-api"),
			AccessTTL:  parseDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL: parseDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Upload: UploadCfg{
			Dir:       getEnv("UPLOAD_DIR", "resource/uploads"),
			PublicURL: getEnv("UPLOAD_PUBLIC_URL", "http://localhost:8000/uploads"),
			MaxSizeMB: parseInt("UPLOAD_MAX_SIZE_MB", 5),
		},
		Mail: MailCfg{
			Host: getEnv("SMTP_HOST", "localhost"),
			Port: parseInt("SMTP_PORT", 1025),
			User: getEnv("SMTP_USER", ""),
			Pass: getEnv("SMTP_PASS", ""),
			From: getEnv("SMTP_FROM", "GymMaster <no-reply@gymmaster.vn>"),
		},
		Cron: CronCfg{
			Enabled: parseBool("CRON_ENABLED", true),
		},
		Seed: SeedCfg{
			AdminEmail:    getEnv("SEED_ADMIN_EMAIL", "admin@gymmaster.vn"),
			AdminPassword: getEnv("SEED_ADMIN_PASSWORD", "Admin@123"),
		},
	}
	return c
}

// DSN dùng cho GORM (pgx) và golang-migrate.
func (d DbCfg) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func parseInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return def
}

func parseBool(key string, def bool) bool {
	if v, err := strconv.ParseBool(os.Getenv(key)); err == nil {
		return v
	}
	return def
}

func parseDuration(key string, def time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(key)); err == nil {
		return v
	}
	return def
}

// corsHosts tách danh sách origin (ngăn bởi dấu phẩy) và bỏ scheme, vì GoFrame so khớp CORS theo host.
func corsHosts(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		p = strings.TrimSpace(p)
		p = strings.TrimPrefix(strings.TrimPrefix(p, "https://"), "http://")
		if p = strings.TrimSuffix(p, "/"); p != "" {
			out = append(out, p)
		}
	}
	return out
}

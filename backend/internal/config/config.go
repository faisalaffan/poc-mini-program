package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config menyimpan konfigurasi runtime backend OCP.
type Config struct {
	Port                 string
	AppEnv               string
	BuildVersion         string
	DatabaseURL          string
	JWTSecret            string
	AllowedOrigins       []string
	RateLimitUserPerMin  int
	RateLimitDevPerMin   int
	DrawMinIntervalMs    int
	WebDistDir           string
	OpenAPIPath          string
	AdminUsername        string
	AdminPassword        string
}

// Load membaca konfigurasi dari .env (jika ada) dan environment variables.
func Load() (Config, error) {
	loadDotEnv()
	cfg := Config{
		Port:                getEnv("PORT", "8080"),
		AppEnv:              getEnv("APP_ENV", "poc"),
		BuildVersion:        getEnv("BUILD_VERSION", "0.1.0-poc"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://gacha:gacha_poc_password@127.0.0.1:5432/gacha_poc?sslmode=disable"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		AllowedOrigins:      splitCSV(getEnv("ALLOWED_ORIGINS", "https://livin-village.faisalaffan.com,http://127.0.0.1:8080,http://localhost:8080,http://macbook.singapura-parore.ts.net:8080,https://macbook.singapura-parore.ts.net")),
		RateLimitUserPerMin: getEnvInt("RATE_LIMIT_USER_PER_MIN", 60),
		RateLimitDevPerMin:  getEnvInt("RATE_LIMIT_DEVICE_PER_MIN", 90),
		DrawMinIntervalMs:   getEnvInt("DRAW_MIN_INTERVAL_MS", 400),
		WebDistDir:          resolveExistingPath(getEnv("WEB_DIST_DIR", "../web-gacha/dist"), "web-gacha/dist"),
		OpenAPIPath:         resolveExistingPath(getEnv("OPENAPI_PATH", "../docs/api/openapi.yaml"), "docs/api/openapi.yaml"),
		AdminUsername:       getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:       getEnv("ADMIN_PASSWORD", "LivinVillage2026!"),
	}

	if cfg.JWTSecret == "" {
		if cfg.AppEnv == "poc" {
			cfg.JWTSecret = "poc-local-dev-secret-change-in-ocp-32bytes"
		} else {
			return Config{}, fmt.Errorf("JWT_SECRET must be set when APP_ENV=%s", cfg.AppEnv)
		}
	}

	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func resolveExistingPath(configured, repoRootFallback string) string {
	if _, err := os.Stat(configured); err == nil {
		return configured
	}
	if _, err := os.Stat(repoRootFallback); err == nil {
		return repoRootFallback
	}
	return configured
}

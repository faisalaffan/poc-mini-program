package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv_ReadsFileWithoutOverridingExistingEnv(t *testing.T) {
	tmp := t.TempDir()
	envPath := filepath.Join(tmp, ".env")
	content := `
# komentar
PORT=9090
DATABASE_URL="postgres://gacha:secret@vps-malang.singapura-parore.ts.net:5432/gacha_poc?sslmode=disable"
APP_ENV='poc'
JWT_SECRET=test-dotenv-secret-at-least-32-characters
`
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp .env: %v", err)
	}

	t.Setenv("ENV_FILE", envPath)
	t.Setenv("PORT", "7070") // OS env harus tetap menang
	// Pastikan key lain dibersihkan untuk uji baca dari file
	oldDB, hadDB := os.LookupEnv("DATABASE_URL")
	_ = os.Unsetenv("DATABASE_URL")
	t.Cleanup(func() {
		if hadDB {
			_ = os.Setenv("DATABASE_URL", oldDB)
		} else {
			_ = os.Unsetenv("DATABASE_URL")
		}
	})

	oldJWT, hadJWT := os.LookupEnv("JWT_SECRET")
	_ = os.Unsetenv("JWT_SECRET")
	t.Cleanup(func() {
		if hadJWT {
			_ = os.Setenv("JWT_SECRET", oldJWT)
		} else {
			_ = os.Unsetenv("JWT_SECRET")
		}
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.Port != "7070" {
		t.Fatalf("expected OS env PORT 7070 to take precedence, got %s", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://gacha:secret@vps-malang.singapura-parore.ts.net:5432/gacha_poc?sslmode=disable" {
		t.Fatalf("unexpected DatabaseURL from .env: %s", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != "test-dotenv-secret-at-least-32-characters" {
		t.Fatalf("unexpected JWTSecret from .env")
	}
}

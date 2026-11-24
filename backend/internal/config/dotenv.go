package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// loadDotEnv membaca file .env jika ada tanpa menimpa environment variable yang sudah diset OS/container.
func loadDotEnv() {
	candidates := []string{}
	if custom := strings.TrimSpace(os.Getenv("ENV_FILE")); custom != "" {
		candidates = append(candidates, custom)
	}
	candidates = append(candidates, ".env", filepath.Join("..", ".env"))

	for _, path := range candidates {
		if applyEnvFile(path) {
			return
		}
	}
}

func applyEnvFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if key == "" {
			continue
		}
		val = unquoteEnvValue(val)

		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
	return true
}

func unquoteEnvValue(val string) string {
	if len(val) >= 2 {
		if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
			return val[1 : len(val)-1]
		}
	}
	if commentIdx := strings.Index(val, " #"); commentIdx >= 0 {
		return strings.TrimSpace(val[:commentIdx])
	}
	return val
}

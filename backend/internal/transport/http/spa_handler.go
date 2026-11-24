package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// NewSPAHandler menyajikan aset Vue 3 SPA di bawah prefix /webview/ tanpa traversal path.
func NewSPAHandler(distDir string) http.Handler {
	cleanDir := filepath.Clean(distDir)
	fs := http.FileServer(http.Dir(cleanDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, "/webview")
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			rel = "index.html"
		}

		target := filepath.Join(cleanDir, filepath.Clean("/"+rel))
		info, err := os.Stat(target)
		if err != nil || info.IsDir() {
			indexPath := filepath.Join(cleanDir, "index.html")
			if _, idxErr := os.Stat(indexPath); idxErr != nil {
				http.Error(w, "Vue SPA bundle belum di-build. Jalankan npm run build di web-gacha.", http.StatusServiceUnavailable)
				return
			}
			http.ServeFile(w, r, indexPath)
			return
		}

		http.StripPrefix("/webview/", fs).ServeHTTP(w, r)
	})
}

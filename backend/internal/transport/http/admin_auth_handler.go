package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const adminCookieName = "livin_admin_session"

type AdminSessionManager struct {
	username string
	password string
	secret   []byte
}

func NewAdminSessionManager(username, password, secret string) *AdminSessionManager {
	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "LivinVillage2026!"
	}
	return &AdminSessionManager{
		username: username,
		password: password,
		secret:   []byte(secret),
	}
}

func (m *AdminSessionManager) RequireAdminCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.isAuthenticated(r) {
			next.ServeHTTP(w, r)
			return
		}
		target := "/login"
		if nextPath := r.URL.RequestURI(); nextPath != "" && nextPath != "/login" {
			target = "/login?next=" + url.QueryEscape(nextPath)
		}
		http.Redirect(w, r, target, http.StatusFound)
	})
}

func (m *AdminSessionManager) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	if m.isAuthenticated(r) {
		http.Redirect(w, r, safeNextPath(r.URL.Query().Get("next")), http.StatusFound)
		return
	}
	m.renderLogin(w, http.StatusOK, safeNextPath(r.URL.Query().Get("next")), "")
}

func (m *AdminSessionManager) HandleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		m.renderLogin(w, http.StatusBadRequest, "/sdui-admin/", "Format form tidak valid.")
		return
	}

	user := strings.TrimSpace(r.FormValue("username"))
	pass := r.FormValue("password")
	nextPath := safeNextPath(r.FormValue("next"))

	userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(m.username)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(m.password)) == 1
	if !userMatch || !passMatch {
		m.renderLogin(w, http.StatusUnauthorized, nextPath, "Username atau password salah.")
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour).Unix()
	token := m.signToken(user, expiresAt)
	isHTTPS := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")

	http.SetCookie(w, &http.Cookie{
		Name:     adminCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS,
	})
	http.Redirect(w, r, nextPath, http.StatusSeeOther)
}

func (m *AdminSessionManager) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (m *AdminSessionManager) isAuthenticated(r *http.Request) bool {
	c, err := r.Cookie(adminCookieName)
	if err != nil || c.Value == "" {
		return false
	}
	return m.verifyToken(c.Value)
}

func (m *AdminSessionManager) signToken(username string, expiresUnix int64) string {
	payload := fmt.Sprintf("%s|%d", username, expiresUnix)
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	raw := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return raw + "." + sig
}

func (m *AdminSessionManager) verifyToken(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	rawBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write(rawBytes)
	if !hmac.Equal(sigBytes, mac.Sum(nil)) {
		return false
	}
	fields := strings.Split(string(rawBytes), "|")
	if len(fields) != 2 || fields[0] != m.username {
		return false
	}
	exp, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return true
}

func safeNextPath(raw string) string {
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/login") {
		return "/sdui-admin/"
	}
	return raw
}

func (m *AdminSessionManager) renderLogin(w http.ResponseWriter, status int, nextPath, errMsg string) {
	errBlock := ""
	if errMsg != "" {
		errBlock = `<div class="err">` + html.EscapeString(errMsg) + `</div>`
	}
	page := `<!doctype html>
<html lang="id">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Login · Livin' Village Studio</title>
  <style>
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: -apple-system, BlinkMacSystemFont, "Inter", "Segoe UI", sans-serif;
      background: #f1f3f6;
      color: #111318;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 20px;
    }
    .box {
      width: 100%;
      max-width: 360px;
      background: #ffffff;
      border: 1px solid #e4e6eb;
      border-radius: 12px;
      padding: 24px;
      box-shadow: 0 10px 28px rgba(15, 23, 42, 0.08);
    }
    h1 { margin: 0 0 4px; font-size: 17px; font-weight: 600; }
    p { margin: 0 0 18px; font-size: 12.5px; color: #667085; }
    .err {
      margin-bottom: 14px;
      padding: 9px 11px;
      border-radius: 7px;
      background: #fef2f2;
      border: 1px solid #fecaca;
      color: #dc2626;
      font-size: 12.5px;
    }
    label {
      display: block;
      font-size: 11.5px;
      font-weight: 500;
      color: #667085;
      margin-bottom: 5px;
    }
    input {
      width: 100%;
      font: inherit;
      font-size: 13px;
      border: 1px solid #dfe4ee;
      border-radius: 7px;
      padding: 9px 11px;
      margin-bottom: 14px;
      outline: none;
    }
    input:focus { border-color: #005ae2; }
    button {
      width: 100%;
      font: inherit;
      font-size: 13px;
      font-weight: 600;
      border: 0;
      border-radius: 7px;
      padding: 10px 14px;
      background: #005ae2;
      color: #ffffff;
      cursor: pointer;
    }
    button:hover { background: #0049b8; }
  </style>
</head>
<body>
  <form class="box" method="POST" action="/login">
    <h1>Livin' Village Studio</h1>
    <p>Masuk untuk mengelola SDUI &amp; API Docs</p>
    ` + errBlock + `
    <input type="hidden" name="next" value="` + html.EscapeString(nextPath) + `" />
    <label for="username">Username</label>
    <input id="username" name="username" type="text" autocomplete="username" required autofocus />
    <label for="password">Password</label>
    <input id="password" name="password" type="password" autocomplete="current-password" required />
    <button type="submit">Masuk</button>
  </form>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(page))
}

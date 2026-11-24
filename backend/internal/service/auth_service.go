package service

import (
	"fmt"
	"time"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenScopeNative  = "gacha:native"
	TokenScopeWebView = "gacha:webview"
)

type AuthClaims struct {
	UserID   string `json:"uid"`
	DeviceID string `json:"did"`
	Scope    string `json:"scope"`
	jwt.RegisteredClaims
}

type IssuedToken struct {
	Token      string    `json:"token"`
	TokenType  string    `json:"token_type"`
	ExpiresAt  time.Time `json:"expires_at"`
	TTLSeconds int       `json:"ttl_seconds"`
	UserID     string    `json:"user_id"`
	DeviceID   string    `json:"device_id"`
	Scope      string    `json:"scope"`
}

type AuthService struct {
	secret []byte
}

func NewAuthService(secret string) *AuthService {
	return &AuthService{secret: []byte(secret)}
}

func (s *AuthService) IssueMasterSession(userID, deviceID string) (IssuedToken, error) {
	return s.issueToken(userID, deviceID, TokenScopeNative, 24*time.Hour)
}

// IssueWebViewToken mengeluarkan token berumur pendek (120 detik) agar web tidak pernah memegang master token.
func (s *AuthService) IssueWebViewToken(claims AuthClaims) (IssuedToken, error) {
	if claims.Scope != TokenScopeNative {
		return IssuedToken{}, domain.ErrForbiddenScope
	}
	return s.issueToken(claims.UserID, claims.DeviceID, TokenScopeWebView, 120*time.Second)
}

func (s *AuthService) issueToken(userID, deviceID, scope string, ttl time.Duration) (IssuedToken, error) {
	if userID == "" || deviceID == "" {
		return IssuedToken{}, domain.ErrInvalidInput
	}
	now := time.Now().UTC()
	exp := now.Add(ttl)

	claims := AuthClaims{
		UserID:   userID,
		DeviceID: deviceID,
		Scope:    scope,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "gacha-poc-ocp",
			Subject:   userID,
			Audience:  jwt.ClaimStrings{scope},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return IssuedToken{}, fmt.Errorf("sign jwt: %w", err)
	}

	return IssuedToken{
		Token:      signed,
		TokenType:  "Bearer",
		ExpiresAt:  exp,
		TTLSeconds: int(ttl.Seconds()),
		UserID:     userID,
		DeviceID:   deviceID,
		Scope:      scope,
	}, nil
}

func (s *AuthService) VerifyToken(rawToken string) (AuthClaims, error) {
	var claims AuthClaims
	parsed, err := jwt.ParseWithClaims(rawToken, &claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil || !parsed.Valid {
		return AuthClaims{}, domain.ErrUnauthorized
	}
	if claims.UserID == "" || claims.DeviceID == "" {
		return AuthClaims{}, domain.ErrUnauthorized
	}
	return claims, nil
}

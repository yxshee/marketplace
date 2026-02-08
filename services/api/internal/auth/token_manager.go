package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

var (
	ErrInvalidTokenType = errors.New("invalid token type")
	ErrInvalidToken     = errors.New("invalid token")
)

// Claims are the structured claims extracted from a validated token.
type Claims struct {
	UserID    string
	Role      Role
	SessionID string
	VendorID  *string
	TokenType TokenType
	ExpiresAt time.Time
}

type tokenClaims struct {
	Role      string `json:"role"`
	SessionID string `json:"sid"`
	VendorID  string `json:"vendor_id,omitempty"`
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

// TokenPair contains an access/refresh pair with expiries.
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

// TokenManager signs and validates auth tokens.
type TokenManager struct {
	secret         []byte
	issuer         string
	accessTokenTTL time.Duration
	refreshTTL     time.Duration
}

func NewTokenManager(secret, issuer string, accessTokenTTL, refreshTTL time.Duration) (*TokenManager, error) {
	if secret == "" {
		return nil, errors.New("token secret must not be empty")
	}
	if issuer == "" {
		return nil, errors.New("token issuer must not be empty")
	}
	if accessTokenTTL <= 0 || refreshTTL <= 0 {
		return nil, errors.New("token ttl values must be positive")
	}

	return &TokenManager{
		secret:         []byte(secret),
		issuer:         issuer,
		accessTokenTTL: accessTokenTTL,
		refreshTTL:     refreshTTL,
	}, nil
}

func (m *TokenManager) IssueTokenPair(user User, sessionID string) (TokenPair, error) {
	issuedAt := time.Now().UTC()
	accessExpiry := issuedAt.Add(m.accessTokenTTL)
	refreshExpiry := issuedAt.Add(m.refreshTTL)

	accessToken, err := m.sign(user, sessionID, TokenTypeAccess, issuedAt, accessExpiry)
	if err != nil {
		return TokenPair{}, err
	}

	refreshToken, err := m.sign(user, sessionID, TokenTypeRefresh, issuedAt, refreshExpiry)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExpiry,
		RefreshExpiresAt: refreshExpiry,
	}, nil
}

func (m *TokenManager) sign(user User, sessionID string, tokenType TokenType, issuedAt, expiresAt time.Time) (string, error) {
	claims := tokenClaims{
		Role:      user.Role.String(),
		SessionID: sessionID,
		TokenType: string(tokenType),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	if user.VendorID != nil {
		claims.VendorID = *user.VendorID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *TokenManager) ParseAndValidate(rawToken string, expectedType TokenType) (Claims, error) {
	claims := tokenClaims{}
	token, err := jwt.ParseWithClaims(rawToken, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}

	if claims.TokenType != string(expectedType) {
		return Claims{}, ErrInvalidTokenType
	}

	var vendorID *string
	if claims.VendorID != "" {
		value := claims.VendorID
		vendorID = &value
	}

	role := Role(claims.Role)
	if !isKnownRole(role) {
		return Claims{}, ErrInvalidToken
	}

	if claims.ExpiresAt == nil {
		return Claims{}, ErrInvalidToken
	}

	return Claims{
		UserID:    claims.Subject,
		Role:      role,
		SessionID: claims.SessionID,
		VendorID:  vendorID,
		TokenType: expectedType,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

func isKnownRole(role Role) bool {
	for _, known := range Roles() {
		if role == known {
			return true
		}
	}
	return false
}

func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

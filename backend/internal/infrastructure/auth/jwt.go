package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
)

// Claims is the JWT payload carried in every access token.
type Claims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Role  string `json:"role"`
}

// SignAccessToken creates a signed JWT for the given user.
func SignAccessToken(user *entities.User, secret, issuer string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Email: user.Email,
		Role:  user.Role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("jwt: failed to sign access token: %w", err)
	}
	return signed, nil
}

// ParseAccessToken verifies the signature, expiry, and issuer of a JWT and returns its claims.
func ParseAccessToken(tokenStr, secret, issuer string) (*Claims, error) {
	var claims Claims
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithIssuer(issuer), jwt.WithExpirationRequired())

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("jwt: invalid token: %w", err)
	}
	return &claims, nil
}

// SubjectAsUUID parses the `sub` claim as a UUID.
func (c *Claims) SubjectAsUUID() (uuid.UUID, error) {
	id, err := uuid.Parse(c.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("jwt: sub is not a valid UUID: %w", err)
	}
	return id, nil
}

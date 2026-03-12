package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AccessClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type Identity struct {
	UserID string
	Email  string
}

type Verifier struct {
	secret []byte
	issuer string
}

func NewVerifier(secret, issuer string) (*Verifier, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("auth access token secret must be at least 32 characters")
	}
	if issuer == "" {
		return nil, fmt.Errorf("auth issuer must not be empty")
	}

	return &Verifier{
		secret: []byte(secret),
		issuer: issuer,
	}, nil
}

func (v *Verifier) VerifyAccessToken(tokenString string) (*Identity, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&AccessClaims{},
		func(token *jwt.Token) (any, error) {
			return v.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(v.issuer),
		jwt.WithLeeway(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse access token: %w", err)
	}

	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid || claims.Subject == "" {
		return nil, fmt.Errorf("invalid access token")
	}

	return &Identity{
		UserID: claims.Subject,
		Email:  claims.Email,
	}, nil
}

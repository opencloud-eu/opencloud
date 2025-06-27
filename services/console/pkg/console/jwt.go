package console

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

var ErrJWTTokenUnknownClaim = errors.New("unknown claim in JWT token")

func ParseJWTToken(tokenString, key string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	if token == nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}

func GetJWTClaim[T any](token *jwt.Token, claimKey string) (T, error) {
	var zero T
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return zero, fmt.Errorf("failed to parse token claims")
	}

	claimValue, ok := claims[claimKey].(T)
	if !ok {
		return zero, fmt.Errorf("(%w) failed to get claim %s", ErrJWTTokenUnknownClaim, claimKey)
	}

	return claimValue, nil
}

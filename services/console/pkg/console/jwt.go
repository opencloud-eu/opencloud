package console

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	jwt.RegisteredClaims
	TenantId     string `json:"tenant_id,omitempty" validate:"required"`
	InstanceId   string `json:"instance_id,omitempty" validate:"required"`
	APIVersion   string `json:"api_version,omitempty" validate:"required"`
	WebsocketURL string `json:"websocket_url,omitempty" validate:"required"`
}

func ParseUnverifiedJWTToken(tokenString string) (*jwt.Token, *Claims, error) {
	claims := &Claims{}

	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, claims)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if err := Validate.Struct(claims); err != nil {
		return nil, nil, fmt.Errorf("(%w): %w", ErrValidation, err)
	}

	return token, claims, nil
}

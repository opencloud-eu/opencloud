package config

// TokenManager is the config for using the reva token manager
type TokenManager struct {
	JWTSecret string `yaml:"jwt_secret" env:"OC_JWT_SECRET;INVITATIONS_JWT_SECRET,file" desc:"The secret to mint and validate jwt tokens." introductionVersion:"1.0.0"`
}

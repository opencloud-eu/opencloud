package config

type ConsoleRemote struct {
	JWTToken string `yaml:"jwt_token" env:"CONSOLE_REMOTE_JWT_TOKEN" desc:"The JWT token used to authenticate requests to the console service." introductionVersion:"%%NEXT%%"`
}

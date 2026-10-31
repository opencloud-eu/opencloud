package config

type Auth struct {
	Audiences           []string `yaml:"auds" env:"AUTHAPI_AUTH_AUDS" desc:"Additional audiences to inject into the userinfo response claims" introductionVersion:"1.0.0"`
	RequireSharedSecret bool     `yaml:"require_shared_secret" env:"AUTHAPI_AUTH_REQUIRE_SHARED_SECRET" desc:"Whether to require a shared secret or not" introductionVersion:"1.0.0"`
	SharedSecrets       string   `yaml:"shared_secrets" env:"AUTHAPI_AUTH_SHARED_SECRETS" desc:"Shared secret values" introductionVersion:"1.0.0"`
}

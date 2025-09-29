package config

// Service defines the available service configuration.
type Service struct {
	Name string `yaml:"name" env:"COLLABORATION_SERVICE_NAME" desc:"The name of the service which is registered. You only need to change this when more than one collaboration service is needed." introductionVersion:"3.6.0"`
}

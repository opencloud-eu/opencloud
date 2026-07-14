package logging

import (
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/config"
)

// Configure initializes a service-specific logger instance.
func Configure(name string, cfg *config.Log) log.Logger {
	return log.NewLogger(
		log.Name(name),
		log.Level(cfg.Level),
		log.Pretty(cfg.Pretty),
		log.Color(cfg.Color),
		log.File(cfg.File),
	)
}

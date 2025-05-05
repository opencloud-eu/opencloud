package jmap

import (
	"github.com/opencloud-eu/opencloud/pkg/log"
)

type JmapWellKnownClient interface {
	GetWellKnown(username string, logger *log.Logger) (WellKnownJmap, error)
}

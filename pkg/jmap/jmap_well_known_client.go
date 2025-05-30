package jmap

import (
	"github.com/opencloud-eu/opencloud/pkg/log"
)

type WellKnownClient interface {
	GetWellKnown(username string, logger *log.Logger) (WellKnownResponse, error)
}

// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"time"

	"github.com/go-viper/mapstructure/v2"
)

// AsyncConf is how a service asks for async uploads: whether they are enabled,
// and the consumer subscription to use if they are.
type AsyncConf struct {
	Enabled       bool
	ConsumerGroup string
	NumConsumers  int
	// MountID is the storage id this provider answers for, used to drop
	// postprocessing events belonging to other storages.
	MountID            string
	CommitMaxRetries   int
	CommitRetryBackoff time.Duration
}

// AsyncConfFromDriverConf reads the postprocessing settings off the driver's own
// config keys, so the coordinator and the driver cannot disagree about them.
func AsyncConfFromDriverConf(driverConf map[string]interface{}) AsyncConf {
	if driverConf == nil {
		return AsyncConf{}
	}
	var ac struct {
		AsyncFileUploads bool   `mapstructure:"asyncfileuploads"`
		MountID          string `mapstructure:"mount_id"`
		Events           struct {
			NumConsumers       int           `mapstructure:"numconsumers"`
			ConsumerGroup      string        `mapstructure:"consumer_group"`
			CommitMaxRetries   int           `mapstructure:"commit_max_retries"`
			CommitRetryBackoff time.Duration `mapstructure:"commit_retry_backoff"`
		} `mapstructure:"events"`
	}
	_ = mapstructure.Decode(driverConf, &ac)
	group := ac.Events.ConsumerGroup
	if group == "" {
		group = "dcfs"
	}
	maxRetries := ac.Events.CommitMaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	retryBackoff := ac.Events.CommitRetryBackoff
	if retryBackoff <= 0 {
		retryBackoff = 5 * time.Second
	}
	return AsyncConf{
		Enabled:            ac.AsyncFileUploads,
		ConsumerGroup:      group,
		NumConsumers:       ac.Events.NumConsumers,
		MountID:            ac.MountID,
		CommitMaxRetries:   maxRetries,
		CommitRetryBackoff: retryBackoff,
	}
}

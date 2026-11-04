// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package channels_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestChannels(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Channels Suite")
}

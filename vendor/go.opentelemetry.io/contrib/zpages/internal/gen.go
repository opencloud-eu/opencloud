// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package internal provides non-public types for the zpages package.
package internal

import "embed"

// Templates embeds all the HTML templates used used to serve the tracez
// endpoint
//
//go:embed templates/*
var Templates embed.FS

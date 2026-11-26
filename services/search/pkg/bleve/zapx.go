// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package bleve

import (
	zapv11 "github.com/blevesearch/zapx/v11"
	zapv12 "github.com/blevesearch/zapx/v12"
	zapv13 "github.com/blevesearch/zapx/v13"
	zapv14 "github.com/blevesearch/zapx/v14"
	zapv15 "github.com/blevesearch/zapx/v15"
	zapv16 "github.com/blevesearch/zapx/v16"
	zapv17 "github.com/blevesearch/zapx/v17"
)

// zapx sizes the buffer of a new segment up front from the previous segment:
// its bytes per document times (documents + 100). After a single document with
// a lot of text that is a hundred times its segment, gigabytes for the next
// small batch. A zero factor turns the estimate off and the buffer grows as
// the segment is written.
//
// An index keeps the segment version it was created with, so every zapx
// version bleve can open needs this, not only the newest one. A test fails
// when bleve registers a version that is missing here.
var segmentBufferFactors = map[uint32]*float64{
	(&zapv11.ZapPlugin{}).Version(): &zapv11.NewSegmentBufferAvgBytesPerDocFactor,
	(&zapv12.ZapPlugin{}).Version(): &zapv12.NewSegmentBufferAvgBytesPerDocFactor,
	(&zapv13.ZapPlugin{}).Version(): &zapv13.NewSegmentBufferAvgBytesPerDocFactor,
	(&zapv14.ZapPlugin{}).Version(): &zapv14.NewSegmentBufferAvgBytesPerDocFactor,
	(&zapv15.ZapPlugin{}).Version(): &zapv15.NewSegmentBufferAvgBytesPerDocFactor,
	(&zapv16.ZapPlugin{}).Version(): &zapv16.NewSegmentBufferAvgBytesPerDocFactor,
	(&zapv17.ZapPlugin{}).Version(): &zapv17.NewSegmentBufferAvgBytesPerDocFactor,
}

func init() {
	for _, factor := range segmentBufferFactors {
		*factor = 0
	}
}

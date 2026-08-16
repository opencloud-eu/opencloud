package content

import (
	"context"
	"fmt"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/clip"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

// Clip decorates another extractor with image vectorization: it delegates
// Extract and adds an ImageVector to image documents. Vectorization is
// orthogonal to text extraction, so it wraps 'basic' as well as 'tika'.
type Clip struct {
	next Extractor
	Retriever
	client   *clip.Client
	logger   log.Logger
	maxBytes uint64
}

// NewClipExtractor wraps next with a CLIP vectorization step. It probes the
// inference service once to verify that the configured model produces vectors
// of the schema dimensionality (ImageVectorDims) and refuses to start
// otherwise: mismatched vectors would be dropped silently by the index.
func NewClipExtractor(next Extractor, client *clip.Client, gatewaySelector pool.Selectable[gateway.GatewayAPIClient], logger log.Logger, cfg *config.Config) (*Clip, error) {
	probe, err := client.VectorizeText(context.Background(), "startup probe")
	if err != nil {
		return nil, fmt.Errorf("clip inference service unavailable: %w", err)
	}
	if len(probe) != ImageVectorDims {
		return nil, fmt.Errorf("clip model produces %d-dimensional vectors, the index schema requires %d; use a matching model", len(probe), ImageVectorDims)
	}
	logger.Info().Str("url", cfg.Extractor.Clip.URL).Int("dims", len(probe)).Msg("clip inference service connected")

	maxBytes := cfg.Extractor.Clip.MaxBytes
	if maxBytes == 0 {
		maxBytes = 50 * 1024 * 1024
	}

	return &Clip{
		next:      next,
		Retriever: newCS3Retriever(gatewaySelector, logger, cfg.Extractor.CS3AllowInsecure),
		client:    client,
		logger:    logger,
		maxBytes:  maxBytes,
	}, nil
}

// Extract delegates to the wrapped extractor and adds the image embedding.
// Vectorization failures are not fatal: the document is indexed without a
// vector and the rest of the search keeps working.
func (c *Clip) Extract(ctx context.Context, ri *provider.ResourceInfo) (Document, error) {
	doc, err := c.next.Extract(ctx, ri)
	if err != nil {
		return doc, err
	}

	if ri.Type != provider.ResourceType_RESOURCE_TYPE_FILE ||
		!strings.HasPrefix(ri.GetMimeType(), "image/") ||
		ri.Size == 0 || ri.Size > c.maxBytes {
		return doc, nil
	}

	data, err := c.Retrieve(ctx, ri.Id)
	if err != nil {
		c.logger.Warn().Err(err).Interface("ResourceID", ri.Id).Str("Name", ri.Name).Msg("clip: failed to retrieve image, indexing without vector")
		return doc, nil
	}
	defer data.Close()

	vector, err := c.client.VectorizeImage(ctx, data)
	if err != nil {
		c.logger.Warn().Err(err).Interface("ResourceID", ri.Id).Str("Name", ri.Name).Msg("clip: vectorization failed, indexing without vector")
		return doc, nil
	}
	if len(vector) != ImageVectorDims {
		// the index would drop a mismatched vector silently, make it visible
		c.logger.Warn().Int("dims", len(vector)).Int("want", ImageVectorDims).Interface("ResourceID", ri.Id).Str("Name", ri.Name).Msg("clip: vector dimensionality mismatch, indexing without vector")
		return doc, nil
	}

	doc.ImageVector = vector
	return doc, nil
}

package testcontainers

import (
	"context"

	"github.com/docker/docker/client"
)

// ImageInfo represents summary information of an image
type ImageInfo struct {
	ID   string
	Name string
}

type saveImageOptions struct {
	dockerSaveOpts []client.ImageSaveOption
}

type SaveImageOption func(*saveImageOptions) error

// ImageProvider allows manipulating images
type ImageProvider interface {
	ListImages(context.Context) ([]ImageInfo, error)
	SaveImages(context.Context, string, ...string) error
	SaveImagesWithOpts(context.Context, string, []string, ...SaveImageOption) error
	PullImage(context.Context, string) error
}

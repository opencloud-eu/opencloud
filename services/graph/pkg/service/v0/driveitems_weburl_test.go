package svc

import (
	"net/url"
	"testing"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

func TestCS3ResourceToDriveItemPopulatesWebUrl(t *testing.T) {
	logger := log.NewLogger()
	res := &provider.ResourceInfo{
		Id: &provider.ResourceId{
			StorageId: "storage-1",
			SpaceId:   "space-1",
			OpaqueId:  "item-1",
		},
		Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER,
	}

	t.Run("public base URL without path", func(t *testing.T) {
		base, err := url.Parse("https://example.com")
		require.NoError(t, err)

		item, err := cs3ResourceToDriveItem(&logger, base, res)
		require.NoError(t, err)
		require.NotNil(t, item.WebUrl)
		assert.Equal(t, "https://example.com/f/storage-1$space-1%21item-1", *item.WebUrl)
	})

	t.Run("public base URL with path prefix", func(t *testing.T) {
		base, err := url.Parse("https://example.com/cloud")
		require.NoError(t, err)

		item, err := cs3ResourceToDriveItem(&logger, base, res)
		require.NoError(t, err)
		require.NotNil(t, item.WebUrl)
		assert.Equal(t, "https://example.com/cloud/f/storage-1$space-1%21item-1", *item.WebUrl)
	})
}

func TestCS3ResourceToDriveItemParentReference(t *testing.T) {
	logger := log.NewLogger()
	base, err := url.Parse("https://example.com")
	require.NoError(t, err)

	t.Run("space root", func(t *testing.T) {
		res := &provider.ResourceInfo{
			Id: &provider.ResourceId{
				StorageId: "storage-1",
				SpaceId:   "space-1",
				OpaqueId:  "space-1",
			},
			ParentId: &provider.ResourceId{
				StorageId: "storage-1",
				SpaceId:   "space-1",
				OpaqueId:  "space-1",
			},
			Name: "SpaceName",
			Path: ".",
			Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER,
		}

		item, err := cs3ResourceToDriveItem(&logger, base, res)
		require.NoError(t, err)
		require.NotNil(t, item.Name)
		assert.Equal(t, "SpaceName", *item.Name)
		require.NotNil(t, item.ParentReference)
		assert.Equal(t, "/", item.ParentReference.GetName())
		assert.Equal(t, "/", item.ParentReference.GetPath())
	})

	t.Run("nested item", func(t *testing.T) {
		res := &provider.ResourceInfo{
			Id: &provider.ResourceId{
				StorageId: "storage-1",
				SpaceId:   "space-1",
				OpaqueId:  "item-1",
			},
			ParentId: &provider.ResourceId{
				StorageId: "storage-1",
				SpaceId:   "space-1",
				OpaqueId:  "folder-1",
			},
			Name: "file.txt",
			Path: "./folder/file.txt",
			Type: provider.ResourceType_RESOURCE_TYPE_FILE,
		}

		item, err := cs3ResourceToDriveItem(&logger, base, res)
		require.NoError(t, err)
		require.NotNil(t, item.ParentReference)
		assert.Equal(t, "folder", item.ParentReference.GetName())
		assert.Equal(t, "folder", item.ParentReference.GetPath())
	})
}

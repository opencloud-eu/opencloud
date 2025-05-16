package linktype

import (
	"errors"

	linkv1beta1 "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/grants"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

// NoPermissionMatchError is the message returned by a failed conversion
const NoPermissionMatchError = "no matching permission set found"

// LinkType contains cs3 permissions and a libregraph
// linktype reference
type LinkType struct {
	Permissions *provider.ResourcePermissions
	linkType    libregraph.SharingLinkType
}

// GetPermissions returns the cs3 permissions type
func (l *LinkType) GetPermissions() *provider.ResourcePermissions {
	if l != nil {
		return l.Permissions
	}
	return nil
}

// SharingLinkTypeFromCS3Permissions creates a libregraph link type
// It returns a list of libregraph actions when the conversion is not possible
func SharingLinkTypeFromCS3Permissions(permissions *linkv1beta1.PublicSharePermissions) (*libregraph.SharingLinkType, []string) {
	if permissions == nil {
		return nil, nil
	}
	linkTypes := GetAvailableLinkTypes()
	for _, linkType := range linkTypes {
		if grants.PermissionsEqual(linkType.GetPermissions(), permissions.GetPermissions()) {
			return &linkType.linkType, nil
		}
	}
	return nil, unifiedrole.CS3ResourcePermissionsToLibregraphActions(permissions.GetPermissions())
}

// CS3ResourcePermissionsFromSharingLink creates a cs3 resource permissions type
// it returns an error when the link type is not allowed or empty
func CS3ResourcePermissionsFromSharingLink(createLink libregraph.DriveItemCreateLink, info provider.ResourceType) (*provider.ResourcePermissions, error) {
	switch createLink.GetType() {
	case "":
		return nil, errors.New("link type is empty")
	case libregraph.VIEW:
		return NewViewLinkPermissionSet().GetPermissions(), nil
	case libregraph.EDIT:
		if info == provider.ResourceType_RESOURCE_TYPE_FILE {
			return NewFileEditLinkPermissionSet().GetPermissions(), nil
		}
		return NewFolderEditLinkPermissionSet().GetPermissions(), nil
	case libregraph.CREATE_ONLY:
		if info == provider.ResourceType_RESOURCE_TYPE_FILE {
			return nil, errors.New(NoPermissionMatchError)
		}
		return NewFolderDropLinkPermissionSet().GetPermissions(), nil
	case libregraph.UPLOAD:
		if info == provider.ResourceType_RESOURCE_TYPE_FILE {
			return nil, errors.New(NoPermissionMatchError)
		}
		return NewFolderUploadLinkPermissionSet().GetPermissions(), nil
	case libregraph.INTERNAL:
		return NewInternalLinkPermissionSet().GetPermissions(), nil
	default:
		return nil, errors.New(NoPermissionMatchError)
	}
}

// NewInternalLinkPermissionSet creates cs3 permissions for the internal link type
func NewInternalLinkPermissionSet() *LinkType {
	return &LinkType{
		Permissions: &provider.ResourcePermissions{},
		linkType:    libregraph.INTERNAL,
	}
}

// NewViewLinkPermissionSet creates cs3 permissions for the view link type
func NewViewLinkPermissionSet() *LinkType {
	return &LinkType{
		Permissions: &provider.ResourcePermissions{
			GetPath:              true,
			GetQuota:             true,
			InitiateFileDownload: true,
			ListContainer:        true,
			// why is this needed?
			ListRecycle: true,
			Stat:        true,
		},
		linkType: libregraph.VIEW,
	}
}

// NewFileEditLinkPermissionSet creates cs3 permissions for the file edit link type
func NewFileEditLinkPermissionSet() *LinkType {
	return &LinkType{
		Permissions: &provider.ResourcePermissions{
			GetPath:              true,
			GetQuota:             true,
			InitiateFileDownload: true,
			InitiateFileUpload:   true,
			ListContainer:        true,
			// why is this needed?
			ListRecycle: true,
			// why is this needed?
			RestoreRecycleItem: true,
			Stat:               true,
		},
		linkType: libregraph.EDIT,
	}
}

// NewFolderEditLinkPermissionSet creates cs3 permissions for the folder edit link type
func NewFolderEditLinkPermissionSet() *LinkType {
	return &LinkType{
		Permissions: &provider.ResourcePermissions{
			CreateContainer:      true,
			Delete:               true,
			GetPath:              true,
			GetQuota:             true,
			InitiateFileDownload: true,
			InitiateFileUpload:   true,
			ListContainer:        true,
			// why is this needed?
			ListRecycle: true,
			Move:        true,
			// why is this needed?
			RestoreRecycleItem: true,
			Stat:               true,
		},
		linkType: libregraph.EDIT,
	}
}

// NewFolderDropLinkPermissionSet creates cs3 permissions for the folder createOnly link type
func NewFolderDropLinkPermissionSet() *LinkType {
	return &LinkType{
		Permissions: &provider.ResourcePermissions{
			Stat:               true,
			GetPath:            true,
			CreateContainer:    true,
			InitiateFileUpload: true,
		},
		linkType: libregraph.CREATE_ONLY,
	}
}

// NewFolderUploadLinkPermissionSet creates cs3 permissions for the folder upload link type
func NewFolderUploadLinkPermissionSet() *LinkType {
	return &LinkType{
		Permissions: &provider.ResourcePermissions{
			CreateContainer:      true,
			GetPath:              true,
			GetQuota:             true,
			InitiateFileDownload: true,
			InitiateFileUpload:   true,
			ListContainer:        true,
			ListRecycle:          true,
			Stat:                 true,
		},
		linkType: libregraph.UPLOAD,
	}
}

// GetAvailableLinkTypes returns a slice of all available link types
func GetAvailableLinkTypes() []*LinkType {
	return []*LinkType{
		NewInternalLinkPermissionSet(),
		NewViewLinkPermissionSet(),
		NewFolderUploadLinkPermissionSet(),
		NewFileEditLinkPermissionSet(),
		NewFolderEditLinkPermissionSet(),
		NewFolderDropLinkPermissionSet(),
	}
}

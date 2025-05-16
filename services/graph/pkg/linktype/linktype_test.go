package linktype_test

import (
	linkv1beta1 "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/linktype"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/grants"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

var _ = Describe("LinktypeFromPermission", func() {
	var (
		internalLinkType, _        = libregraph.NewSharingLinkTypeFromValue("internal")
		createOnlyLinkType, _      = libregraph.NewSharingLinkTypeFromValue("createOnly")
		viewLinkType, _            = libregraph.NewSharingLinkTypeFromValue("view")
		uploadLinkType, _          = libregraph.NewSharingLinkTypeFromValue("upload")
		editLinkType, _            = libregraph.NewSharingLinkTypeFromValue("edit")
		folderEditPermsHaveChanged = linktype.NewFolderEditLinkPermissionSet().GetPermissions()
	)

	BeforeEach(func() {
		// simulate that permissions have changed after link creation
		folderEditPermsHaveChanged.CreateContainer = false
	})

	DescribeTable("SharingLinkTypeFromCS3Permissions",
		func(permissions *linkv1beta1.PublicSharePermissions,
			expectedSharingLinkType *libregraph.SharingLinkType,
			expectedActions []string) {

			sharingLinkType, actions := linktype.SharingLinkTypeFromCS3Permissions(permissions)
			Expect(sharingLinkType).To(Equal(expectedSharingLinkType))
			Expect(expectedActions).To(Equal(actions))
		},

		Entry("Internal",
			&linkv1beta1.PublicSharePermissions{Permissions: linktype.NewInternalLinkPermissionSet().GetPermissions()},
			internalLinkType,
			nil,
		),
		Entry("CreateOnly",
			&linkv1beta1.PublicSharePermissions{Permissions: linktype.NewFolderDropLinkPermissionSet().GetPermissions()},
			createOnlyLinkType,
			nil,
		),
		Entry("View File",
			&linkv1beta1.PublicSharePermissions{Permissions: linktype.NewViewLinkPermissionSet().GetPermissions()},
			viewLinkType,
			nil,
		),
		Entry("Upload in Folder",
			&linkv1beta1.PublicSharePermissions{Permissions: linktype.NewFolderUploadLinkPermissionSet().GetPermissions()},
			uploadLinkType,
			nil,
		),
		Entry("File Edit",
			&linkv1beta1.PublicSharePermissions{Permissions: linktype.NewFileEditLinkPermissionSet().GetPermissions()},
			editLinkType,
			nil,
		),
		Entry("Folder Edit",
			&linkv1beta1.PublicSharePermissions{Permissions: linktype.NewFolderEditLinkPermissionSet().GetPermissions()},
			editLinkType,
			nil,
		),
		Entry("Folder Edit- Permissions have changed after creation",
			&linkv1beta1.PublicSharePermissions{Permissions: folderEditPermsHaveChanged},
			nil,
			[]string{
				"libre.graph/driveItem/standard/delete",
				"libre.graph/driveItem/path/read",
				"libre.graph/driveItem/quota/read",
				"libre.graph/driveItem/content/read",
				"libre.graph/driveItem/upload/create",
				"libre.graph/driveItem/children/read",
				"libre.graph/driveItem/deleted/read",
				"libre.graph/driveItem/path/update",
				"libre.graph/driveItem/deleted/update",
				"libre.graph/driveItem/basic/read",
			},
		),
	)

	DescribeTable("CS3ResourcePermissionsFromSharingLink",
		func(createLink libregraph.DriveItemCreateLink,
			info provider.ResourceType,
			expectedPermissions *provider.ResourcePermissions,
			hasError bool) {

			permissions, err := linktype.CS3ResourcePermissionsFromSharingLink(createLink, info)
			if hasError == true {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(grants.PermissionsEqual(permissions, expectedPermissions)).To(BeTrue())
			}
		},

		Entry("Internal",
			libregraph.DriveItemCreateLink{Type: internalLinkType},
			provider.ResourceType_RESOURCE_TYPE_FILE, linktype.NewInternalLinkPermissionSet().GetPermissions(),
			false,
		),
		Entry("Create Only",
			libregraph.DriveItemCreateLink{Type: createOnlyLinkType},
			provider.ResourceType_RESOURCE_TYPE_CONTAINER, linktype.NewFolderDropLinkPermissionSet().GetPermissions(),
			false,
		),
		Entry("Create Only",
			libregraph.DriveItemCreateLink{Type: createOnlyLinkType},
			provider.ResourceType_RESOURCE_TYPE_FILE, linktype.NewFolderDropLinkPermissionSet().GetPermissions(),
			true,
		),
		Entry("View File",
			libregraph.DriveItemCreateLink{Type: viewLinkType},
			provider.ResourceType_RESOURCE_TYPE_FILE, linktype.NewViewLinkPermissionSet().GetPermissions(),
			false,
		),
		Entry("View Folder",
			libregraph.DriveItemCreateLink{Type: viewLinkType},
			provider.ResourceType_RESOURCE_TYPE_CONTAINER, linktype.NewViewLinkPermissionSet().GetPermissions(),
			false,
		),
		Entry("Upload in Folder",
			libregraph.DriveItemCreateLink{Type: uploadLinkType},
			provider.ResourceType_RESOURCE_TYPE_CONTAINER, linktype.NewFolderUploadLinkPermissionSet().GetPermissions(),
			false,
		),
		Entry("File Edit",
			libregraph.DriveItemCreateLink{Type: editLinkType},
			provider.ResourceType_RESOURCE_TYPE_FILE, linktype.NewFileEditLinkPermissionSet().GetPermissions(),
			false,
		),
		Entry("Folder Edit",
			libregraph.DriveItemCreateLink{Type: editLinkType},
			provider.ResourceType_RESOURCE_TYPE_CONTAINER, linktype.NewFolderEditLinkPermissionSet().GetPermissions(),
			false,
		),
	)
})

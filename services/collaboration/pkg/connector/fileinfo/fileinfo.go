package fileinfo

// FileInfo contains the properties of the file.
// Some properties refer to capabilities in the WOPI client, and capabilities
// that the WOPI server has.
//
// Specific implementations must allow json-encoding of their relevant
// properties because the object will be marshalled directly
type FileInfo interface {
	// SetProperties will set the properties of this FileInfo.
	// Keys should match any valid property that the FileInfo implementation
	// has. If a key doesn't match any property, it must be ignored.
	// The values must have its matching type for the target property,
	// otherwise panics might happen.
	//
	// This method should help to reduce the friction of using different
	// implementations with different properties. You can use the same map
	// for all the implementations knowing that the relevant properties for
	// each implementation will be set.
	SetProperties(props map[string]interface{})

	// GetTarget will return the target implementation (OnlyOffice, Collabora...).
	// This will help to identify the implementation we're using in an easy way.
	// Note that the returned value must be unique among all the implementations
	GetTarget() string
}

// constants that can be used to refer the fileinfo properties for the
// SetProperties method of the FileInfo interface
const (
	KeyBaseFileName = "BaseFileName"
	KeyOwnerID      = "OwnerId"
	KeySize         = "Size"
	KeyUserID       = "UserID"
	KeyVersion      = "Version"

	KeySupportedShareURLTypes     = "SupportedShareURLTypes"
	KeySupportsCobalt             = "SupportsCobalt"
	KeySupportsContainers         = "SupportsContainers"
	KeySupportsDeleteFile         = "SupportsDeleteFile"
	KeySupportsEcosystem          = "SupportsEcosystem"
	KeySupportsExtendedLockLength = "SupportsExtendedLockLength"
	KeySupportsFolders            = "SupportsFolders"
	//KeySupportsGetFileWopiSrc = "SupportsGetFileWopiSrc"  // wopivalidator is complaining and the property isn't used for now -> commented
	KeySupportsGetLock  = "SupportsGetLock"
	KeySupportsLocks    = "SupportsLocks"
	KeySupportsRename   = "SupportsRename"
	KeySupportsUpdate   = "SupportsUpdate"
	KeySupportsUserInfo = "SupportsUserInfo"

	KeyIsAnonymousUser              = "IsAnonymousUser"
	KeyIsEduUser                    = "IsEduUser"
	KeyIsAdminUser                  = "IsAdminUser"
	KeyLicenseCheckForEditIsEnabled = "LicenseCheckForEditIsEnabled"
	KeyUserFriendlyName             = "UserFriendlyName"
	KeyUserInfo                     = "UserInfo"

	KeyReadOnly                = "ReadOnly"
	KeyRestrictedWebViewOnly   = "RestrictedWebViewOnly"
	KeyUserCanAttend           = "UserCanAttend"
	KeyUserCanNotWriteRelative = "UserCanNotWriteRelative"
	KeyUserCanPresent          = "UserCanPresent"
	KeyUserCanRename           = "UserCanRename"
	KeyUserCanWrite            = "UserCanWrite"

	KeyCloseURL            = "CloseURL"
	KeyDownloadURL         = "DownloadURL"
	KeyFileEmbedCommandURL = "FileEmbedCommandURL"
	KeyFileSharingURL      = "FileSharingURL"
	KeyFileURL             = "FileURL"
	KeyFileVersionURL      = "FileVersionURL"
	KeyHostEditURL         = "HostEditURL"
	KeyHostEmbeddedViewURL = "HostEmbeddedViewURL"
	KeyHostViewURL         = "HostViewURL"
	KeySignoutURL          = "SignoutURL"

	KeyAllowAdditionalMicrosoftServices = "AllowAdditionalMicrosoftServices"
	KeyAllowErrorReportPrompt           = "AllowErrorReportPrompt"
	KeyAllowExternalMarketplace         = "AllowExternalMarketplace"
	KeyClientThrottlingProtection       = "ClientThrottlingProtection"
	KeyCloseButtonClosesWindow          = "CloseButtonClosesWindow"
	KeyCopyPasteRestrictions            = "CopyPasteRestrictions"
	KeyDisablePrint                     = "DisablePrint"
	KeyDisableTranslation               = "DisableTranslation"
	KeyFileExtension                    = "FileExtension"
	KeyFileNameMaxLength                = "FileNameMaxLength"
	KeyLastModifiedTime                 = "LastModifiedTime"
	KeyRequestedCallThrottling          = "RequestedCallThrottling"
	KeySHA256                           = "SHA256"
	KeySharingStatus                    = "SharingStatus"
	KeyTemporarilyNotWritable           = "TemporarilyNotWritable"
	//KeyUniqueContentId = "UniqueContentId"  // From microsoft docs: Not supported in CSPP -> commented

	KeyBreadcrumbBrandName  = "BreadcrumbBrandName"
	KeyBreadcrumbBrandURL   = "BreadcrumbBrandURL"
	KeyBreadcrumbDocName    = "BreadcrumbDocName"
	KeyBreadcrumbFolderName = "BreadcrumbFolderName"
	KeyBreadcrumbFolderURL  = "BreadcrumbFolderUrl"

	// Collabora (non-dupped) properties below

	KeyPostMessageOrigin = "PostMessageOrigin"
	KeyTemplateSource    = "TemplateSource"

	KeyEnableInsertRemoteImage = "EnableInsertRemoteImage"
	KeyEnableInsertRemoteFile  = "EnableInsertRemoteFile"
	KeyDisableInsertLocalImage = "DisableInsertLocalImage"
	KeyHidePrintOption         = "HidePrintOption"
	KeyHideSaveOption          = "HideSaveOption"
	KeyHideExportOption        = "HideExportOption"
	KeyDisableExport           = "DisableExport"
	KeyDisableCopy             = "DisableCopy"
	KeyDisableInactiveMessages = "DisableInactiveMessages"
	KeyDownloadAsPostMessage   = "DownloadAsPostMessage"
	KeySaveAsPostmessage       = "SaveAsPostmessage"
	KeyEnableOwnerTermination  = "EnableOwnerTermination"
	//KeyUserExtraInfo -> requires definition, currently not used
	//KeyUserPrivateInfo -> requires definition, currently not used
	KeyWatermarkText = "WatermarkText"

	KeyEnableShare  = "EnableShare"
	KeyHideUserList = "HideUserList"

	// OnlyOffice (non-dupped) properties below

	KeyClosePostMessage            = "ClosePostMessage"
	KeyEditModePostMessage         = "EditModePostMessage"
	KeyEditNotificationPostMessage = "EditNotificationPostMessage"
	KeyFileSharingPostMessage      = "FileSharingPostMessage"
	KeyFileVersionPostMessage      = "FileVersionPostMessage"

	KeyUserCanReview     = "UserCanReview"
	KeySupportsReviewing = "SupportsReviewing"
)

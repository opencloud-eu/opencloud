package fileinfo

// OnlyOffice fileInfo properties
//
// OnlyOffice WOPI check file info specification:
// https://api.onlyoffice.com/editors/wopi/restapi/checkfileinfo
type OnlyOffice struct {
	//
	// Required response properties
	//

	// copied from MS WOPI
	BaseFileName string `json:"BaseFileName,omitempty"`
	// copied from MS WOPI
	Version string `json:"Version,omitempty"`

	//
	// Breadcrumb properties
	//

	// copied from MS WOPI
	BreadcrumbBrandName string `json:"BreadcrumbBrandName,omitempty"`
	// copied from MS WOPI
	BreadcrumbBrandURL string `json:"BreadcrumbBrandUrl,omitempty"`
	// copied from MS WOPI
	BreadcrumbDocName string `json:"BreadcrumbDocName,omitempty"`
	// copied from MS WOPI
	BreadcrumbFolderName string `json:"BreadcrumbFolderName,omitempty"`
	// copied from MS WOPI
	BreadcrumbFolderURL string `json:"BreadcrumbFolderUrl,omitempty"`

	//
	// PostMessage properties
	//

	// Specifies if the WOPI client should notify the WOPI server in case the user closes the rendering or editing client currently using this file. The host expects to receive the UI_Close PostMessage when the Close UI in the online office is activated.
	ClosePostMessage bool `json:"ClosePostMessage,omitempty"`
	// Specifies if the WOPI client should notify the WOPI server in case the user tries to edit a file. The host expects to receive the UI_Edit PostMessage when the Edit UI in the online office is activated.
	EditModePostMessage bool `json:"EditModePostMessage,omitempty"`
	// Specifies if the WOPI client should notify the WOPI server in case the user tries to edit a file. The host expects to receive the Edit_Notification PostMessage.
	EditNotificationPostMessage bool `json:"EditNotificationPostMessage,omitempty"`
	// Specifies if the WOPI client should notify the WOPI server in case the user tries to share a file. The host expects to receive the UI_Sharing PostMessage when the Share UI in the online office is activated.
	FileSharingPostMessage bool `json:"FileSharingPostMessage,omitempty"`
	// Specifies if the WOPI client will notify the WOPI server in case the user tries to navigate to the previous file version. The host expects to receive the UI_FileVersions PostMessage when the Previous Versions UI in the online office is activated.
	FileVersionPostMessage bool `json:"FileVersionPostMessage,omitempty"`
	// A domain that the WOPI client must use as the targetOrigin parameter when sending messages as described in [W3C-HTML5WEBMSG].
	// copied from collabora WOPI
	PostMessageOrigin string `json:"PostMessageOrigin,omitempty"`

	//
	// File URL properties
	//

	// copied from MS WOPI
	CloseURL string `json:"CloseUrl,omitempty"`
	// copied from MS WOPI
	FileSharingURL string `json:"FileSharingUrl,omitempty"`
	// copied from MS WOPI
	FileVersionURL string `json:"FileVersionUrl,omitempty"`
	// copied from MS WOPI
	HostEditURL string `json:"HostEditUrl,omitempty"`

	//
	// Miscellaneous properties
	//

	// Specifies if the WOPI client must disable the Copy and Paste functionality within the application. By default, all Copy and Paste functionality is enabled, i.e. the setting has no effect. Possible property values:
	// BlockAll - the Copy and Paste functionality is completely disabled within the application;
	// CurrentDocumentOnly - the Copy and Paste functionality is enabled but content can only be copied and pasted within the file currently open in the application.
	// copied from MS WOPI
	CopyPasteRestrictions string `json:"CopyPasteRestrictions,omitempty"`
	// copied from MS WOPI
	DisablePrint bool `json:"DisablePrint"`
	// copied from MS WOPI
	FileExtension string `json:"FileExtension,omitempty"`
	// copied from MS WOPI
	FileNameMaxLength int `json:"FileNameMaxLength,omitempty"`
	// copied from MS WOPI
	LastModifiedTime string `json:"LastModifiedTime,omitempty"`
	// The ID of file (like the wopi/files/ID) can be a non-existing file. In that case, the file will be created from a template when the template (eg. an OTT file) is specified as TemplateSource in the CheckFileInfo response. The TemplateSource is supposed to be an URL like https://somewhere/accessible/file.ott that is accessible by the Online. For the actual saving of the content, normal PutFile mechanism will be used.
	TemplateSource string `json:"TemplateSource,omitempty"`

	//
	// User metadata properties
	//

	// copied from MS WOPI
	IsAnonymousUser bool `json:"IsAnonymousUser,omitempty"`
	// copied from MS WOPI
	UserFriendlyName string `json:"UserFriendlyName,omitempty"`
	// copied from MS WOPI
	UserID string `json:"UserId,omitempty"`

	//
	// User permissions properties
	//

	// copied from MS WOPI
	ReadOnly bool `json:"ReadOnly"`
	// copied from MS WOPI
	UserCanNotWriteRelative bool `json:"UserCanNotWriteRelative"`
	// copied from MS WOPI
	UserCanRename bool `json:"UserCanRename"`
	// Specifies if the user has permissions to review a file.
	UserCanReview bool `json:"UserCanReview,omitempty"`
	// copied from MS WOPI
	UserCanWrite bool `json:"UserCanWrite"`

	//
	// Host capabilities properties
	//

	// copied from MS WOPI
	SupportsLocks bool `json:"SupportsLocks"`
	// copied from MS WOPI
	SupportsRename bool `json:"SupportsRename"`
	// Specifies if the WOPI server supports the review permission.
	SupportsReviewing bool `json:"SupportsReviewing,omitempty"`
	// copied from MS WOPI
	SupportsUpdate bool `json:"SupportsUpdate"` // whether "Putfile" and "PutRelativeFile" work

	//
	// Other properties
	//

	// copied from collabora WOPI
	EnableInsertRemoteImage bool `json:"EnableInsertRemoteImage,omitempty"`
	// copied from collabora WOPI
	HidePrintOption bool `json:"HidePrintOption,omitempty"`
}

// SetProperties will set the file properties for the OnlyOffice implementation.
func (oinfo *OnlyOffice) SetProperties(props map[string]any) {
	for key, value := range props {
		switch key {
		case KeyBaseFileName:
			oinfo.BaseFileName = value.(string)
		case KeyVersion:
			oinfo.Version = value.(string)

		case KeyBreadcrumbBrandName:
			oinfo.BreadcrumbBrandName = value.(string)
		case KeyBreadcrumbBrandURL:
			oinfo.BreadcrumbBrandURL = value.(string)
		case KeyBreadcrumbDocName:
			oinfo.BreadcrumbDocName = value.(string)
		case KeyBreadcrumbFolderName:
			oinfo.BreadcrumbFolderName = value.(string)
		case KeyBreadcrumbFolderURL:
			oinfo.BreadcrumbFolderURL = value.(string)

		case KeyClosePostMessage:
			oinfo.ClosePostMessage = value.(bool)
		case KeyEditModePostMessage:
			oinfo.EditModePostMessage = value.(bool)
		case KeyEditNotificationPostMessage:
			oinfo.EditNotificationPostMessage = value.(bool)
		case KeyFileSharingPostMessage:
			oinfo.FileSharingPostMessage = value.(bool)
		case KeyFileVersionPostMessage:
			oinfo.FileVersionPostMessage = value.(bool)
		case KeyPostMessageOrigin:
			oinfo.PostMessageOrigin = value.(string)

		case KeyCloseURL:
			oinfo.CloseURL = value.(string)
		case KeyFileSharingURL:
			oinfo.FileSharingURL = value.(string)
		case KeyFileVersionURL:
			oinfo.FileVersionURL = value.(string)
		case KeyHostEditURL:
			oinfo.HostEditURL = value.(string)

		case KeyCopyPasteRestrictions:
			oinfo.CopyPasteRestrictions = value.(string)
		case KeyDisablePrint:
			oinfo.DisablePrint = value.(bool)
		case KeyFileExtension:
			oinfo.FileExtension = value.(string)
		case KeyFileNameMaxLength:
			oinfo.FileNameMaxLength = value.(int)
		case KeyLastModifiedTime:
			oinfo.LastModifiedTime = value.(string)
		case KeyTemplateSource:
			oinfo.TemplateSource = value.(string)
		case KeyIsAnonymousUser:
			oinfo.IsAnonymousUser = value.(bool)
		case KeyUserFriendlyName:
			oinfo.UserFriendlyName = value.(string)
		case KeyUserID:
			oinfo.UserID = value.(string)

		case KeyReadOnly:
			oinfo.ReadOnly = value.(bool)
		case KeyUserCanNotWriteRelative:
			oinfo.UserCanNotWriteRelative = value.(bool)
		case KeyUserCanRename:
			oinfo.UserCanRename = value.(bool)
		case KeyUserCanReview:
			oinfo.UserCanReview = value.(bool)
		case KeyUserCanWrite:
			oinfo.UserCanWrite = value.(bool)

		case KeySupportsLocks:
			oinfo.SupportsLocks = value.(bool)
		case KeySupportsRename:
			oinfo.SupportsRename = value.(bool)
		case KeySupportsReviewing:
			oinfo.SupportsReviewing = value.(bool)
		case KeySupportsUpdate:
			oinfo.SupportsUpdate = value.(bool)

		case KeyEnableInsertRemoteImage:
			oinfo.EnableInsertRemoteImage = value.(bool)
		case KeyHidePrintOption:
			oinfo.HidePrintOption = value.(bool)
		}
	}
}

// GetTarget will always return "OnlyOffice"
func (oinfo *OnlyOffice) GetTarget() string {
	return "OnlyOffice"
}

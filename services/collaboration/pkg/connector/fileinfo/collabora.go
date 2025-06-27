package fileinfo

// Collabora fileInfo properties
//
// Collabora WOPI check file info specification:
// https://sdk.collaboraonline.com/docs/advanced_integration.html
type Collabora struct {
	//
	// Response properties
	//

	// Copied from MS WOPI
	BaseFileName string `json:"BaseFileName,omitempty"`
	// Copied from MS WOPI
	DisablePrint bool `json:"DisablePrint"`
	// Copied from MS WOPI
	OwnerID string `json:"OwnerId,omitempty"`
	// A string for the domain the host page sends/receives PostMessages from, we only listen to messages from this domain.
	PostMessageOrigin string `json:"PostMessageOrigin,omitempty"`
	// copied from MS WOPI
	Size int64 `json:"Size"`
	// The ID of file (like the wopi/files/ID) can be a non-existing file. In that case, the file will be created from a template when the template (eg. an OTT file) is specified as TemplateSource in the CheckFileInfo response. The TemplateSource is supposed to be an URL like https://somewhere/accessible/file.ott that is accessible by the Online. For the actual saving of the content, normal PutFile mechanism will be used.
	TemplateSource string `json:"TemplateSource,omitempty"`
	// copied from MS WOPI
	UserCanWrite bool `json:"UserCanWrite"`
	// copied from MS WOPI
	UserCanNotWriteRelative bool `json:"UserCanNotWriteRelative"`
	// copied from MS WOPI
	UserID string `json:"UserId,omitempty"`
	// copied from MS WOPI
	UserFriendlyName string `json:"UserFriendlyName,omitempty"`

	//
	// Extended response properties
	//

	// If set to true, this will enable the insertion of images chosen from the WOPI storage. A UI_InsertGraphic postMessage will be send to the WOPI host to request the UI to select the file.
	EnableInsertRemoteImage bool `json:"EnableInsertRemoteImage,omitempty"`
	// If set to true, this will disable the insertion of image chosen from the local device. If EnableInsertRemoteImage is not set to true, then inserting images files is not possible.
	DisableInsertLocalImage bool `json:"DisableInsertLocalImage,omitempty"`
	// If set to true, hides the print option from the file menu bar in the UI.
	HidePrintOption bool `json:"HidePrintOption,omitempty"`
	// If set to true, hides the save button from the toolbar and file menubar in the UI.
	HideSaveOption bool `json:"HideSaveOption,omitempty"`
	// Hides Download as option in the file menubar.
	HideExportOption bool `json:"HideExportOption,omitempty"`
	// Disables export functionality in backend. If set to true, HideExportOption is assumed to be true
	DisableExport bool `json:"DisableExport,omitempty"`
	// Disables copying from the document in libreoffice online backend. Pasting into the document would still be possible. However, it is still possible to do an “internal” cut/copy/paste.
	DisableCopy bool `json:"DisableCopy,omitempty"`
	// Disables displaying of the explanation text on the overlay when the document becomes inactive or killed. With this, the JS integration must provide the user with appropriate message when it gets Session_Closed or User_Idle postMessages.
	DisableInactiveMessages bool `json:"DisableInactiveMessages,omitempty"`
	// Indicate that the integration wants to handle the downloading of pdf for printing or svg for slideshows or exported document, because it cannot rely on browser’s support for downloading.
	DownloadAsPostMessage bool `json:"DownloadAsPostMessage,omitempty"`
	// Similar to download as, doctype extensions can be provided for save-as. In this case the new file is loaded in the integration instead of downloaded.
	SaveAsPostmessage bool `json:"SaveAsPostmessage,omitempty"`
	// If set to true, it allows the document owner (the one with OwnerId =UserId) to send a closedocument message (see protocol.txt)
	EnableOwnerTermination bool `json:"EnableOwnerTermination,omitempty"`

	// JSON object that contains additional info about the user, namely the avatar image.
	//UserExtraInfo -> requires definition, currently not used
	// JSON object that contains additional info about the user, but unlike the UserExtraInfo it is not shared among the views in collaborative editing sessions.
	//UserPrivateInfo -> requires definition, currently not used

	// If set to a non-empty string, is used for rendering a watermark-like text on each tile of the document.
	WatermarkText string `json:"WatermarkText,omitempty"`

	//
	// Undocumented (from source code)
	//

	EnableShare bool `json:"EnableShare,omitempty"`
	// If set to "true", user list on the status bar will be hidden
	// If set to "mobile" | "tablet" | "desktop", will be hidden on a specified device
	// (may be joint, delimited by commas eg. "mobile,tablet")
	HideUserList      string `json:"HideUserList,omitempty"`
	SupportsLocks     bool   `json:"SupportsLocks"`
	SupportsRename    bool   `json:"SupportsRename"`
	UserCanRename     bool   `json:"UserCanRename"`
	BreadcrumbDocName string `json:"BreadcrumbDocName,omitempty"`
}

// SetProperties will set the file properties for the Collabora implementation.
func (cinfo *Collabora) SetProperties(props map[string]interface{}) {
	for key, value := range props {
		switch key {
		case KeyBaseFileName:
			cinfo.BaseFileName = value.(string)
		case KeyDisablePrint:
			cinfo.DisablePrint = value.(bool)
		case KeyOwnerID:
			cinfo.OwnerID = value.(string)
		case KeyPostMessageOrigin:
			cinfo.PostMessageOrigin = value.(string)
		case KeySize:
			cinfo.Size = value.(int64)
		case KeyTemplateSource:
			cinfo.TemplateSource = value.(string)
		case KeyUserCanWrite:
			cinfo.UserCanWrite = value.(bool)
		case KeyUserCanNotWriteRelative:
			cinfo.UserCanNotWriteRelative = value.(bool)
		case KeyUserID:
			cinfo.UserID = value.(string)
		case KeyUserFriendlyName:
			cinfo.UserFriendlyName = value.(string)

		case KeyEnableInsertRemoteImage:
			cinfo.EnableInsertRemoteImage = value.(bool)
		case KeyDisableInsertLocalImage:
			cinfo.DisableInsertLocalImage = value.(bool)
		case KeyHidePrintOption:
			cinfo.HidePrintOption = value.(bool)
		case KeyHideSaveOption:
			cinfo.HideSaveOption = value.(bool)
		case KeyHideExportOption:
			cinfo.HideExportOption = value.(bool)
		case KeyDisableExport:
			cinfo.DisableExport = value.(bool)
		case KeyDisableCopy:
			cinfo.DisableCopy = value.(bool)
		case KeyDisableInactiveMessages:
			cinfo.DisableInactiveMessages = value.(bool)
		case KeyDownloadAsPostMessage:
			cinfo.DownloadAsPostMessage = value.(bool)
		case KeySaveAsPostmessage:
			cinfo.SaveAsPostmessage = value.(bool)
		case KeyEnableOwnerTermination:
			cinfo.EnableOwnerTermination = value.(bool)
		//UserExtraInfo -> requires definition, currently not used
		//UserPrivateInfo -> requires definition, currently not used
		case KeyWatermarkText:
			cinfo.WatermarkText = value.(string)

		case KeyEnableShare:
			cinfo.EnableShare = value.(bool)
		case KeyHideUserList:
			cinfo.HideUserList = value.(string)
		case KeySupportsLocks:
			cinfo.SupportsLocks = value.(bool)
		case KeySupportsRename:
			cinfo.SupportsRename = value.(bool)
		case KeyUserCanRename:
			cinfo.UserCanRename = value.(bool)
		case KeyBreadcrumbDocName:
			cinfo.BreadcrumbDocName = value.(string)
		}
	}
}

// GetTarget will always return "Collabora"
func (cinfo *Collabora) GetTarget() string {
	return "Collabora"
}

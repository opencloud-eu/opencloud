package config

// Options are the option for the web
type Options struct {
	AccountEditLink        *AccountEditLink    `json:"accountEditLink,omitempty" yaml:"accountEditLink"`
	DisableFeedbackLink    bool                `json:"disableFeedbackLink,omitempty" yaml:"disableFeedbackLink" env:"WEB_OPTION_DISABLE_FEEDBACK_LINK" desc:"Set this option to 'true' to disable the feedback link in the top bar. Keeping it enabled by setting the value to 'false' or with the absence of the option, allows OpenCloud to get feedback from your user base through a dedicated survey website." introductionVersion:"1.0.0"`
	FeedbackLink           *FeedbackLink       `json:"feedbackLink,omitempty" yaml:"feedbackLink"`
	RunningOnEOS           bool                `json:"runningOnEos,omitempty" yaml:"runningOnEos" env:"WEB_OPTION_RUNNING_ON_EOS" desc:"Set this option to 'true' if running on an EOS storage backend (https://eos-web.web.cern.ch/eos-web/) to enable its specific features. Defaults to 'false'." introductionVersion:"1.0.0"`
	CernFeatures           bool                `json:"cernFeatures,omitempty" yaml:"cernFeatures"`
	Upload                 *Upload             `json:"upload,omitempty" yaml:"upload"`
	Editor                 *Editor             `json:"editor,omitempty" yaml:"editor"`
	ContextHelpersReadMore bool                `json:"contextHelpersReadMore,omitempty" yaml:"contextHelpersReadMore" env:"WEB_OPTION_CONTEXTHELPERS_READ_MORE" desc:"Specifies whether the 'Read more' link should be displayed or not." introductionVersion:"1.0.0"`
	LogoutURL              string              `json:"logoutUrl,omitempty" yaml:"logoutUrl" env:"WEB_OPTION_LOGOUT_URL" desc:"Adds a link to the user's profile page to point him to an external page, where he can manage his session and devices. This is helpful when an external IdP is used. This option is disabled by default." introductionVersion:"1.0.0"`
	LoginURL               string              `json:"loginUrl,omitempty" yaml:"loginUrl" env:"WEB_OPTION_LOGIN_URL" desc:"Specifies the target URL to the login page. This is helpful when an external IdP is used. This option is disabled by default. Example URL like: https://www.myidp.com/login." introductionVersion:"1.0.0"`
	TokenStorageLocal      bool                `json:"tokenStorageLocal" yaml:"tokenStorageLocal" env:"WEB_OPTION_TOKEN_STORAGE_LOCAL" desc:"Specifies whether the access token will be stored in the local storage when set to 'true' or in the session storage when set to 'false'. If stored in the local storage, login state will be persisted across multiple browser tabs, means no additional logins are required." introductionVersion:"1.0.0"`
	DisabledExtensions     []string            `json:"disabledExtensions,omitempty" yaml:"disabledExtensions" env:"WEB_OPTION_DISABLED_EXTENSIONS" desc:"A list to disable specific Web extensions identified by their ID. The ID can e.g. be taken from the 'index.ts' file of the web extension. Example: 'com.github.opencloud-eu.web.files.search,com.github.opencloud-eu.web.files.print'. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	Embed                  *Embed              `json:"embed,omitempty" yaml:"embed"`
	UserListRequiresFilter bool                `json:"userListRequiresFilter,omitempty" yaml:"userListRequiresFilter" env:"WEB_OPTION_USER_LIST_REQUIRES_FILTER" desc:"Defines whether one or more filters must be set in order to list users in the Web admin settings. Set this option to 'true' if running in an environment with a lot of users and listing all users could slow down performance. Defaults to 'false'." introductionVersion:"1.0.0"`
	ConcurrentRequests     *ConcurrentRequests `json:"concurrentRequests,omitempty" yaml:"concurrentRequests"`
	DefaultAppID           string              `json:"defaultAppId,omitempty" yaml:"defaultAppId" env:"WEB_OPTION_DEFAULT_APP_ID" desc:"Defines the entrypoint for the web ui." introductionVersion:"%%NEXT%%"`
}

// AccountEditLink are the AccountEditLink options
type AccountEditLink struct {
	Href string `json:"href,omitempty" yaml:"href" env:"WEB_OPTION_ACCOUNT_EDIT_LINK_HREF" desc:"Set a different target URL for the edit link. Make sure to prepend it with 'http(s)://'." introductionVersion:"1.0.0"`
}

// FeedbackLink are the feedback link options
type FeedbackLink struct {
	Href        string `json:"href,omitempty" yaml:"href" env:"WEB_OPTION_FEEDBACKLINK_HREF" desc:"Set a target URL for the feedback link. Make sure to prepend it with 'http(s)://'. Defaults to 'https://opencloud.eu/web-design-feedback'." introductionVersion:"1.0.0"`
	AriaLabel   string `json:"ariaLabel,omitempty" yaml:"ariaLabel" env:"WEB_OPTION_FEEDBACKLINK_ARIALABEL" desc:"Since the feedback link only has an icon, a screen reader accessible label can be set. The text defaults to 'OpenCloud feedback survey'." introductionVersion:"1.0.0"`
	Description string `json:"description,omitempty" yaml:"description" env:"WEB_OPTION_FEEDBACKLINK_DESCRIPTION" desc:"For feedbacks, provide any description you want to see as tooltip and as accessible description. Defaults to 'Provide your feedback: We'd like to improve the web design and would be happy to hear your feedback. Thank you! Your OpenCloud team'." introductionVersion:"1.0.0"`
}

// Upload are the upload options
type Upload struct {
	CompanionURL string `json:"companionUrl,omitempty" yaml:"companionUrl" env:"WEB_OPTION_UPLOAD_COMPANION_URL" desc:"Sets the URL of Companion which is a service provided by Uppy to import files from external cloud providers. See https://uppy.io/docs/companion/ for instructions on how to set up Companion. This feature is disabled as long as no URL is given." introductionVersion:"1.0.0"`
}

// Editor are the web editor options
type Editor struct {
	AutosaveEnabled  bool `json:"autosaveEnabled,omitempty" yaml:"autosaveEnabled" env:"WEB_OPTION_EDITOR_AUTOSAVE_ENABLED" desc:"Specifies if the autosave for the editor apps is enabled." introductionVersion:"1.0.0"`
	AutosaveInterval int  `json:"autosaveInterval,omitempty" yaml:"autosaveInterval" env:"WEB_OPTION_EDITOR_AUTOSAVE_INTERVAL" desc:"Specifies the time interval for the autosave of editor apps in seconds. Has no effect when WEB_OPTION_EDITOR_AUTOSAVE_ENABLED is set to 'false'." introductionVersion:"1.0.0"`
}

// Embed are the Embed options
type Embed struct {
	Enabled                      string `json:"enabled,omitempty" yaml:"enabled" env:"WEB_OPTION_EMBED_ENABLED" desc:"Defines whether Web should be running in 'embed' mode. Setting this to 'true' will enable a stripped down version of Web with reduced functionality used to integrate Web into other applications like via iFrame. Setting it to 'false' or not setting it (default) will run Web as usual with all functionality enabled. See the text description for more details." introductionVersion:"1.0.0"`
	Target                       string `json:"target,omitempty" yaml:"target" env:"WEB_OPTION_EMBED_TARGET" desc:"Defines how Web is being integrated when running in 'embed' mode. Currently, the only supported options are '' (empty) and 'location'. With '' which is the default, Web will run regular as defined via the 'embed.enabled' config option. With 'location', Web will run embedded as location picker. Resource selection will be disabled and the selected resources array always includes the current folder as the only item. See the text description for more details." introductionVersion:"1.0.0"`
	MessagesOrigin               string `json:"messagesOrigin,omitempty" yaml:"messagesOrigin" env:"WEB_OPTION_EMBED_MESSAGES_ORIGIN" desc:"Defines a URL under which Web can be integrated via iFrame in 'embed' mode. Note that setting this is mandatory when running Web in 'embed' mode. Use '*' as value to allow running the iFrame under any URL, although this is not recommended for security reasons. See the text description for more details." introductionVersion:"1.0.0"`
	DelegateAuthentication       bool   `json:"delegateAuthentication,omitempty" yaml:"delegateAuthentication" env:"WEB_OPTION_EMBED_DELEGATE_AUTHENTICATION" desc:"Defines whether Web should require authentication to be done by the parent application when running in 'embed' mode. If set to 'true' Web will not try to authenticate the user on its own but will require an access token coming from the parent application. Defaults to being unset." introductionVersion:"1.0.0"`
	DelegateAuthenticationOrigin string `json:"delegateAuthenticationOrigin,omitempty" yaml:"delegateAuthenticationOrigin" env:"WEB_OPTION_EMBED_DELEGATE_AUTHENTICATION_ORIGIN" desc:"Defines the host to validate the message event origin against when running Web in 'embed' mode with delegated authentication. Defaults to event message origin validation being omitted, which is only recommended for development setups." introductionVersion:"1.0.0"`
}

// ConcurrentRequests are the ConcurrentRequests options
type ConcurrentRequests struct {
	ResourceBatchActions int                       `json:"resourceBatchActions,omitempty" yaml:"resourceBatchActions" env:"WEB_OPTION_CONCURRENT_REQUESTS_RESOURCE_BATCH_ACTIONS" desc:"Defines the maximum number of concurrent requests per file/folder/space batch action. Defaults to 4." introductionVersion:"1.0.0"`
	SSE                  int                       `json:"sse,omitempty" yaml:"sse" env:"WEB_OPTION_CONCURRENT_REQUESTS_SSE" desc:"Defines the maximum number of concurrent requests in SSE event handlers. Defaults to 4." introductionVersion:"1.0.0"`
	Shares               *ConcurrentRequestsShares `json:"shares,omitempty" yaml:"shares"`
}

// ConcurrentRequestsShares are the Shares options inside the ConcurrentRequests options
type ConcurrentRequestsShares struct {
	Create int `json:"create,omitempty" yaml:"create" env:"WEB_OPTION_CONCURRENT_REQUESTS_SHARES_CREATE" desc:"Defines the maximum number of concurrent requests per sharing invite batch. Defaults to 4." introductionVersion:"1.0.0"`
	List   int `json:"list,omitempty" yaml:"list" env:"WEB_OPTION_CONCURRENT_REQUESTS_SHARES_LIST" desc:"Defines the maximum number of concurrent requests when loading individual share information inside listings. Defaults to 2." introductionVersion:"1.0.0"`
}

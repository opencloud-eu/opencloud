// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package ocs

import (
	"encoding/xml"
)

// ocsBool implements the xml/json Marshaler interface. The OCS API inconsistency require us to parse boolean values
// as native booleans for json requests but "truthy" 0/1 values for xml requests.
type ocsBool bool

func (c *ocsBool) MarshalJSON() ([]byte, error) {
	if *c {
		return []byte("true"), nil
	}

	return []byte("false"), nil
}

func (c ocsBool) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if c {
		return e.EncodeElement("1", start)
	}

	return e.EncodeElement("0", start)
}

// CapabilitiesData holds the capabilities data
type CapabilitiesData struct {
	Capabilities *Capabilities `json:"capabilities" xml:"capabilities"`
	Version      *Version      `json:"version" xml:"version"`
}

// Capabilities groups several capability aspects
type Capabilities struct {
	Core           *CapabilitiesCore           `json:"core" xml:"core"`
	Checksums      *CapabilitiesChecksums      `json:"checksums" xml:"checksums"`
	Files          *CapabilitiesFiles          `json:"files" xml:"files" mapstructure:"files"`
	Dav            *CapabilitiesDav            `json:"dav" xml:"dav"`
	FilesSharing   *CapabilitiesFilesSharing   `json:"files_sharing" xml:"files_sharing" mapstructure:"files_sharing"`
	Spaces         *Spaces                     `json:"spaces,omitempty" xml:"spaces,omitempty" mapstructure:"spaces"`
	Graph          *CapabilitiesGraph          `json:"graph,omitempty" xml:"graph,omitempty" mapstructure:"graph"`
	PasswordPolicy *CapabilitiesPasswordPolicy `json:"password_policy,omitempty" xml:"password_policy,omitempty" mapstructure:"password_policy"`
	Search         *CapabilitiesSearch         `json:"search,omitempty" xml:"search,omitempty" mapstructure:"search"`
	Theme          *CapabilitiesTheme          `json:"theme,omitempty" xml:"theme,omitempty" mapstructure:"theme"`
	Notifications  *CapabilitiesNotifications  `json:"notifications,omitempty" xml:"notifications,omitempty"`
	Groupware      *CapabilitiesGroupware      `json:"groupware,omitempty" xml:"groupware,omitempty" mapstructure:"groupware"`
}

// CapabilitiesSearch holds the search capabilities
type CapabilitiesSearch struct {
	Property *CapabilitiesSearchProperties `json:"property" xml:"property" mapstructure:"property"`
}

// CapabilitiesSearchProperties holds the search property capabilities
type CapabilitiesSearchProperties struct {
	Name      *CapabilitiesSearchProperty         `json:"name" xml:"name" mapstructure:"name"`
	Mtime     *CapabilitiesSearchPropertyExtended `json:"mtime" xml:"mtime" mapstructure:"mtime"`
	Size      *CapabilitiesSearchProperty         `json:"size" xml:"size" mapstructure:"size"`
	MediaType *CapabilitiesSearchPropertyExtended `json:"mediatype" xml:"mediatype" mapstructure:"mediatype"`
	Type      *CapabilitiesSearchProperty         `json:"type" xml:"type" mapstructure:"type"`
	Tag       *CapabilitiesSearchProperty         `json:"tag" xml:"tag" mapstructure:"tag"`
	Tags      *CapabilitiesSearchProperty         `json:"tags" xml:"tags" mapstructure:"tags"`
	Content   *CapabilitiesSearchProperty         `json:"content" xml:"content" mapstructure:"content"`
	Scope     *CapabilitiesSearchProperty         `json:"scope" xml:"scope" mapstructure:"scope"`
}

// CapabilitiesSearchProperty represents the default search property
type CapabilitiesSearchProperty struct {
	Enabled ocsBool `json:"enabled" xml:"enabled" mapstructure:"enabled"`
}

// CapabilitiesSearchPropertyExtended represents the extended search property
type CapabilitiesSearchPropertyExtended struct {
	CapabilitiesSearchProperty `mapstructure:",squash"`
	Keywords                   []string `json:"keywords,omitempty" xml:"keywords,omitempty" mapstructure:"keywords"`
}

// Spaces lets a service configure its advertised options related to Storage Spaces.
type Spaces struct {
	Version   string `json:"version" xml:"version" mapstructure:"version"`
	Enabled   bool   `json:"enabled" xml:"enabled" mapstructure:"enabled"`
	Projects  bool   `json:"projects" xml:"projects" mapstructure:"projects"`
	ShareJail bool   `json:"share_jail" xml:"share_jail" mapstructure:"share_jail"`
	MaxQuota  uint64 `json:"max_quota" xml:"max_quota" mapstructure:"max_quota"`
}

// CapabilitiesCore holds webdav config
type CapabilitiesCore struct {
	PollInterval      int     `json:"pollinterval" xml:"pollinterval" mapstructure:"poll_interval"`
	WebdavRoot        string  `json:"webdav-root,omitempty" xml:"webdav-root,omitempty" mapstructure:"webdav_root"`
	Status            *Status `json:"status" xml:"status"`
	CheckForUpdates   ocsBool `json:"check-for-updates" xml:"check-for-updates" mapstructure:"check_for_updates"`
	SupportURLSigning ocsBool `json:"support-url-signing" xml:"support-url-signing" mapstructure:"support_url_signing"`
	SupportSSE        ocsBool `json:"support-sse" xml:"support-sse" mapstructure:"support_sse"`
	SupportRadicale   ocsBool `json:"support-radicale" xml:"support-radicale" mapstructure:"support_radicale"`
}

// CapabilitiesGraph holds the graph capabilities
type CapabilitiesGraph struct {
	PersonalDataExport ocsBool                `json:"personal-data-export" xml:"personal-data-export" mapstructure:"personal_data_export"`
	Users              CapabilitiesGraphUsers `json:"users" xml:"users" mapstructure:"users"`
}

// CapabilitiesPasswordPolicy hold the password policy capabilities
type CapabilitiesPasswordPolicy struct {
	MinCharacters          int                 `json:"min_characters" xml:"min_characters" mapstructure:"min_characters"`
	MaxCharacters          int                 `json:"max_characters" xml:"max_characters" mapstructure:"max_characters"`
	MinLowerCaseCharacters int                 `json:"min_lowercase_characters" xml:"min_lowercase_characters" mapstructure:"min_lowercase_characters"`
	MinUpperCaseCharacters int                 `json:"min_uppercase_characters" xml:"min_uppercase_characters" mapstructure:"min_uppercase_characters"`
	MinDigits              int                 `json:"min_digits" xml:"min_digits" mapstructure:"min_digits"`
	MinSpecialCharacters   int                 `json:"min_special_characters" xml:"min_special_characters" mapstructure:"min_special_characters"`
	BannedPasswordsList    map[string]struct{} `json:"-" xml:"-" mapstructure:"banned_passwords_list"`
}

// CapabilitiesGraphUsers holds the graph user capabilities
type CapabilitiesGraphUsers struct {
	ReadOnlyAttributes         []string `json:"read_only_attributes" xml:"read_only_attributes" mapstructure:"read_only_attributes"`
	CreateDisabled             ocsBool  `json:"create_disabled" xml:"create_disabled" mapstructure:"create_disabled"`
	DeleteDisabled             ocsBool  `json:"delete_disabled" xml:"delete_disabled" mapstructure:"delete_disabled"`
	ChangePasswordSelfDisabled ocsBool  `json:"change_password_self_disabled" xml:"change_password_self_disabled" mapstructure:"change_password_self_disabled"`
	EditLoginAllowedDisabled   ocsBool  `json:"edit_login_allowed_disabled" xml:"edit_login_allowed_disabled" mapstructure:"edit_login_allowed_disabled"`
}

// Status holds basic status information
type Status struct {
	Installed      ocsBool `json:"installed" xml:"installed"`
	Maintenance    ocsBool `json:"maintenance" xml:"maintenance"`
	NeedsDBUpgrade ocsBool `json:"needsDbUpgrade" xml:"needsDbUpgrade"`
	Version        string  `json:"version" xml:"version"`
	VersionString  string  `json:"versionstring" xml:"versionstring"`
	Edition        string  `json:"edition" xml:"edition"`
	ProductName    string  `json:"productname" xml:"productname"`
	Product        string  `json:"product" xml:"product"`
	ProductVersion string  `json:"productversion" xml:"productversion"`
	Hostname       string  `json:"hostname,omitempty" xml:"hostname,omitempty"`
	Channel        string  `json:"channel" xml:"channel"`
}

// CapabilitiesChecksums holds available hashes
type CapabilitiesChecksums struct {
	SupportedTypes      []string `json:"supportedTypes" xml:"supportedTypes>element" mapstructure:"supported_types"`
	PreferredUploadType string   `json:"preferredUploadType" xml:"preferredUploadType" mapstructure:"preferred_upload_type"`
}

// CapabilitiesFilesTusSupport TODO this must be a summary of storages
type CapabilitiesFilesTusSupport struct {
	Version            string `json:"version" xml:"version"`
	Resumable          string `json:"resumable" xml:"resumable"`
	Extension          string `json:"extension" xml:"extension"`
	MaxChunkSize       int    `json:"max_chunk_size" xml:"max_chunk_size" mapstructure:"max_chunk_size"`
	HTTPMethodOverride string `json:"http_method_override" xml:"http_method_override" mapstructure:"http_method_override"`
}

// CapabilitiesArchiver holds available archivers information
type CapabilitiesArchiver struct {
	Enabled     bool     `json:"enabled" xml:"enabled" mapstructure:"enabled"`
	Version     string   `json:"version" xml:"version" mapstructure:"version"`
	Formats     []string `json:"formats" xml:"formats" mapstructure:"formats"`
	ArchiverURL string   `json:"archiver_url" xml:"archiver_url" mapstructure:"archiver_url"`
	MaxNumFiles string   `json:"max_num_files" xml:"max_num_files" mapstructure:"max_num_files"`
	MaxSize     string   `json:"max_size" xml:"max_size" mapstructure:"max_size"`
}

// CapabilitiesAppProvider holds available app provider information
type CapabilitiesAppProvider struct {
	Enabled    bool   `json:"enabled" xml:"enabled" mapstructure:"enabled"`
	Version    string `json:"version" xml:"version" mapstructure:"version"`
	AppsURL    string `json:"apps_url" xml:"apps_url" mapstructure:"apps_url"`
	OpenURL    string `json:"open_url" xml:"open_url" mapstructure:"open_url"`
	OpenWebURL string `json:"open_web_url" xml:"open_web_url" mapstructure:"open_web_url"`
	NewURL     string `json:"new_url" xml:"new_url" mapstructure:"new_url"`
}

// CapabilitiesFiles TODO this is storage specific, not global. What effect do these options have on the clients?
type CapabilitiesFiles struct {
	PrivateLinks     ocsBool                      `json:"privateLinks" xml:"privateLinks" mapstructure:"private_links"`
	BigFileChunking  ocsBool                      `json:"bigfilechunking" xml:"bigfilechunking"`
	Undelete         ocsBool                      `json:"undelete" xml:"undelete"`
	Versioning       ocsBool                      `json:"versioning" xml:"versioning"`
	Favorites        ocsBool                      `json:"favorites" xml:"favorites"`
	FullTextSearch   ocsBool                      `json:"full_text_search" xml:"full_text_search" mapstructure:"full_text_search"`
	Tags             ocsBool                      `json:"tags" xml:"tags"`
	BlacklistedFiles []string                     `json:"blacklisted_files" xml:"blacklisted_files>element" mapstructure:"blacklisted_files"`
	TusSupport       *CapabilitiesFilesTusSupport `json:"tus_support" xml:"tus_support" mapstructure:"tus_support"`
	Archivers        []*CapabilitiesArchiver      `json:"archivers" xml:"archivers" mapstructure:"archivers"`
	AppProviders     []*CapabilitiesAppProvider   `json:"app_providers" xml:"app_providers" mapstructure:"app_providers"`
}

// CapabilitiesDav holds dav endpoint config
type CapabilitiesDav struct {
	Chunking                       string   `json:"chunking" xml:"chunking"`
	Trashbin                       string   `json:"trashbin" xml:"trashbin"`
	Reports                        []string `json:"reports" xml:"reports>element" mapstructure:"reports"`
	ChunkingParallelUploadDisabled bool     `json:"chunkingParallelUploadDisabled" xml:"chunkingParallelUploadDisabled"`
}

// CapabilitiesFilesSharing TODO document
type CapabilitiesFilesSharing struct {
	APIEnabled                    ocsBool                                  `json:"api_enabled" xml:"api_enabled" mapstructure:"api_enabled"`
	GroupSharing                  ocsBool                                  `json:"group_sharing" xml:"group_sharing" mapstructure:"group_sharing"`
	SharingRoles                  ocsBool                                  `json:"sharing_roles" xml:"sharing_roles" mapstructure:"sharing_roles"`
	DenyAccess                    ocsBool                                  `json:"deny_access" xml:"deny_access" mapstructure:"deny_access"`
	AutoAcceptShare               ocsBool                                  `json:"auto_accept_share" xml:"auto_accept_share" mapstructure:"auto_accept_share"`
	ShareWithGroupMembersOnly     ocsBool                                  `json:"share_with_group_members_only" xml:"share_with_group_members_only" mapstructure:"share_with_group_members_only"`
	ShareWithMembershipGroupsOnly ocsBool                                  `json:"share_with_membership_groups_only" xml:"share_with_membership_groups_only" mapstructure:"share_with_membership_groups_only"`
	SearchMinLength               int                                      `json:"search_min_length" xml:"search_min_length" mapstructure:"search_min_length"`
	DefaultPermissions            int                                      `json:"default_permissions" xml:"default_permissions" mapstructure:"default_permissions"`
	UserEnumeration               *CapabilitiesFilesSharingUserEnumeration `json:"user_enumeration" xml:"user_enumeration" mapstructure:"user_enumeration"`
	Federation                    *CapabilitiesFilesSharingFederation      `json:"federation" xml:"federation"`
	Public                        *CapabilitiesFilesSharingPublic          `json:"public" xml:"public"`
	User                          *CapabilitiesFilesSharingUser            `json:"user" xml:"user"`
	// TODO: Remove next line once web defaults to resharing=false
	Resharing ocsBool `json:"resharing" xml:"resharing"`
}

// CapabilitiesFilesSharingPublic TODO document
type CapabilitiesFilesSharingPublic struct {
	Enabled            ocsBool                                   `json:"enabled" xml:"enabled"`
	SendMail           ocsBool                                   `json:"send_mail" xml:"send_mail" mapstructure:"send_mail"`
	SocialShare        ocsBool                                   `json:"social_share" xml:"social_share" mapstructure:"social_share"`
	Upload             ocsBool                                   `json:"upload" xml:"upload"`
	Multiple           ocsBool                                   `json:"multiple" xml:"multiple"`
	SupportsUploadOnly ocsBool                                   `json:"supports_upload_only" xml:"supports_upload_only" mapstructure:"supports_upload_only"`
	Password           *CapabilitiesFilesSharingPublicPassword   `json:"password" xml:"password"`
	ExpireDate         *CapabilitiesFilesSharingPublicExpireDate `json:"expire_date" xml:"expire_date" mapstructure:"expire_date"`
	CanEdit            ocsBool                                   `json:"can_edit" xml:"can_edit" mapstructure:"can_edit"`
	Alias              ocsBool                                   `json:"alias" xml:"alias"`
	DefaultPermissions int                                       `json:"default_permissions" xml:"default_permissions" mapstructure:"default_permissions"`
}

// CapabilitiesFilesSharingPublicPassword TODO document
type CapabilitiesFilesSharingPublicPassword struct {
	EnforcedFor *CapabilitiesFilesSharingPublicPasswordEnforcedFor `json:"enforced_for" xml:"enforced_for" mapstructure:"enforced_for"`
	Enforced    ocsBool                                            `json:"enforced" xml:"enforced"`
}

// CapabilitiesFilesSharingPublicPasswordEnforcedFor TODO document
type CapabilitiesFilesSharingPublicPasswordEnforcedFor struct {
	ReadOnly        ocsBool `json:"read_only" xml:"read_only,omitempty" mapstructure:"read_only"`
	ReadWrite       ocsBool `json:"read_write" xml:"read_write,omitempty" mapstructure:"read_write"`
	ReadWriteDelete ocsBool `json:"read_write_delete" xml:"read_write_delete,omitempty" mapstructure:"read_write_delete"`
	UploadOnly      ocsBool `json:"upload_only" xml:"upload_only,omitempty" mapstructure:"upload_only"`
}

// CapabilitiesFilesSharingPublicExpireDate TODO document
type CapabilitiesFilesSharingPublicExpireDate struct {
	Enabled ocsBool `json:"enabled" xml:"enabled"`
}

// CapabilitiesFilesSharingUser TODO document
type CapabilitiesFilesSharingUser struct {
	SendMail       ocsBool                                   `json:"send_mail" xml:"send_mail" mapstructure:"send_mail"`
	ProfilePicture ocsBool                                   `json:"profile_picture" xml:"profile_picture" mapstructure:"profile_picture"`
	Settings       []*CapabilitiesUserSettings               `json:"settings" xml:"settings" mapstructure:"settings"`
	ExpireDate     *CapabilitiesFilesSharingPublicExpireDate `json:"expire_date" xml:"expire_date" mapstructure:"expire_date"`
}

// CapabilitiesUserSettings holds available user settings service information
type CapabilitiesUserSettings struct {
	Enabled bool   `json:"enabled" xml:"enabled" mapstructure:"enabled"`
	Version string `json:"version" xml:"version" mapstructure:"version"`
}

// CapabilitiesFilesSharingUserEnumeration TODO document
type CapabilitiesFilesSharingUserEnumeration struct {
	Enabled          ocsBool `json:"enabled" xml:"enabled"`
	GroupMembersOnly ocsBool `json:"group_members_only" xml:"group_members_only" mapstructure:"group_members_only"`
}

// CapabilitiesFilesSharingFederation holds outgoing and incoming flags
type CapabilitiesFilesSharingFederation struct {
	Outgoing ocsBool `json:"outgoing" xml:"outgoing"`
	Incoming ocsBool `json:"incoming" xml:"incoming"`
}

// CapabilitiesNotifications holds a list of notification endpoints
type CapabilitiesNotifications struct {
	Endpoints    []string `json:"ocs-endpoints,omitempty" xml:"ocs-endpoints>element,omitempty" mapstructure:"endpoints"`
	Configurable bool     `json:"configurable" xml:"configurable,omitempty" mapstructure:"configurable"`
}

// CapabilitiesTheme holds theming capabilities
type CapabilitiesTheme struct {
	Logo *CapabilitiesThemeLogo `json:"logo" xml:"logo" mapstructure:"logo"`
}

// CapabilitiesThemeLogo holds theming logo capabilities
type CapabilitiesThemeLogo struct {
	// xml marshal, unmarshal does not support map[string]string, needs a custom type with MarshalXML and UnmarshalXML implementations
	PermittedFileTypes map[string]string `json:"permitted_file_types" xml:"-" mapstructure:"permitted_file_types"`
}

// CapabilitiesGroupware holds groupware capabilities
type CapabilitiesGroupware struct {
	Enabled bool `json:"enabled,omitempty" xml:"enabled,omitempty" mapstructure:"enabled"`
}

// Version holds version information
type Version struct {
	Major          int    `json:"major" xml:"major"`
	Minor          int    `json:"minor" xml:"minor"`
	Micro          int    `json:"micro" xml:"micro"` // = patch level
	String         string `json:"string" xml:"string"`
	Edition        string `json:"edition" xml:"edition"`
	Product        string `json:"product" xml:"product"`
	ProductVersion string `json:"productversion" xml:"productversion"`
	Channel        string `json:"channel" xml:"channel"`
}

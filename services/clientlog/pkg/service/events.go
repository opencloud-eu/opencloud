package service

// FileEvent is emitted when a file is uploaded/renamed/deleted/...
type FileEvent struct {
	ParentItemID string `json:"parentitemid"`
	ItemID       string `json:"itemid"`
	SpaceID      string `json:"spaceid"`
	InitiatorID  string `json:"initiatorid"`
	Etag         string `json:"etag"`

	// Only in case of sharing (refactor this into separate struct when more fields are needed)
	AffectedUserIDs []string `json:"affecteduserids"`
}

// MetadataEvent is emitted when arbitrary metadata keys were set or unset on a resource
type MetadataEvent struct {
	FileEvent
	ChangedKeys []string `json:"changedkeys"`
	RemovedKeys []string `json:"removedkeys"`
}

// BackchannelLogout is emitted when the callback revived from the identity provider
type BackchannelLogout struct {
	UserID    string `json:"userid"`
	Timestamp string `json:"timestamp"`
	SessionID string `json:"sessionid"`
}

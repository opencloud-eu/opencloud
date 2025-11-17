package inotifywaitgo

import "log/slog"

type Settings struct {
	// Directory to watch
	Dir string
	// Channel to send the file name to
	FileEvents chan FileEvent
	// Channel to send errors to
	ErrorChan chan error
	// Options for inotifywait
	Options *Options
	// Kill other inotifywait processes
	KillOthers bool
	// verbose
	Verbose bool
	// Logger
	Log *slog.Logger
}

type Options struct {
	// Watch the specified file or directory.  If this option is not specified, inotifywait will watch the current working directory.
	Events []EVENT
	// Print the name of the file that triggered the event.
	Format string
	// Watch all subdirectories of any directories passed as arguments.  Watches will be set up recursively to an unlimited depth.  Symbolic links are not traversed.  Newly created subdirectories will also be watched.
	Recursive bool
	// Set a time format string as accepted by strftime(3) for use with the `%T' conversion in the --format option.
	TimeFmt string
	// Instead of exiting after receiving a single event, execute indefinitely.  The default behaviour is to exit after the first event occurs.
	Monitor bool
}

const (
	// A watched file or a file within a watched directory was read from.
	EventAccess = "access"
	// A watched file or a file within a watched directory was written to.
	EventModify = "modify"
	// The metadata of a watched file or a file within a watched directory was modified.  This includes timestamps, file permissions, extended attributes etc.
	EventAttrib = "attrib"
	// A  watched  file or a file within a watched directory was closed, after being opened in writable mode.  This does not necessarily imply the file was written to.
	EventCloseWrite = "close_write"
	// A watched file or a file within a watched directory was closed, after being opened in read-only mode.
	EventCloseNowrite = "close_nowrite"
	//  A watched file or a file within a watched directory was closed, regardless of how it was opened.  Note that this  is  actually  implemented simply by listening for both close_write and close_nowrite, hence all close events received will be output as one of these, not CLOSE.
	EventClose = "close"
	// A watched file or a file within a watched directory was opened.
	EventOpen = "open"
	// A watched file or a file within a watched directory was moved to the watched directory.
	EventMovedTo = "moved_to"
	// A watched file or a file within a watched directory was moved from the watched directory.
	EventMovedFrom = "moved_from"
	// A watched file or a file within a watched directory was moved to or from the watched directory.  This is equivalent to listening for both moved_from and moved_to.
	EventMove = "move"
	// A watched file or directory was moved. After this event, the file or directory is no longer being watched.
	EventMoveSelf = "move_self"
	//  A file or directory was created within a watched directory.
	EventCreate = "create"
	// A watched file or a file within a watched directory was deleted.
	EventDelete = "delete"
	// A watched file or directory was deleted.  After this event the file or directory is no longer being watched.  Note that this event can  occur even if it is not explicitly being listened for.
	EventDeleteSelf = "delete_self"
	// The  filesystem  on  which  a  watched  file or directory resides was unmounted.  After this event the file or directory is no longer being 	watched.  Note that this event can occur even if it is not explicitly being listened to.
	EventUnmount = "unmount"

	// The subject of this event is a directory
	FlagIsdir = "ISDIR"
)

type EVENT int

const (
	ACCESS = iota + 1000
	MODIFY
	ATTRIB
	CLOSE_WRITE
	CLOSE_NOWRITE
	CLOSE
	OPEN
	MOVED_TO
	MOVED_FROM
	MOVE
	MOVE_SELF
	CREATE
	DELETE
	DELETE_SELF
	UNMOUNT
)

type FileEvent struct {
	Filename string
	Events   []EVENT
	IsDir    bool
}

var EVENT_MAP = map[int]string{
	ACCESS:        EventAccess,
	MODIFY:        EventModify,
	ATTRIB:        EventAttrib,
	CLOSE_WRITE:   EventCloseWrite,
	CLOSE_NOWRITE: EventCloseNowrite,
	CLOSE:         EventClose,
	OPEN:          EventOpen,
	MOVED_TO:      EventMovedTo,
	MOVED_FROM:    EventMovedFrom,
	MOVE:          EventMove,
	MOVE_SELF:     EventMoveSelf,
	CREATE:        EventCreate,
	DELETE:        EventDelete,
	DELETE_SELF:   EventDeleteSelf,
	UNMOUNT:       EventUnmount,
}

var EVENT_MAP_REVERSE = map[string]int{
	EventAccess:       ACCESS,
	EventModify:       MODIFY,
	EventAttrib:       ATTRIB,
	EventCloseWrite:   CLOSE_WRITE,
	EventCloseNowrite: CLOSE_NOWRITE,
	EventClose:        CLOSE,
	EventOpen:         OPEN,
	EventMovedTo:      MOVED_TO,
	EventMovedFrom:    MOVED_FROM,
	EventMove:         MOVE,
	EventMoveSelf:     MOVE_SELF,
	EventCreate:       CREATE,
	EventDelete:       DELETE,
	EventDeleteSelf:   DELETE_SELF,
	EventUnmount:      UNMOUNT,
}

var VALID_EVENTS = []int{
	ACCESS,
	MODIFY,
	ATTRIB,
	CLOSE_WRITE,
	CLOSE_NOWRITE,
	CLOSE,
	OPEN,
	MOVED_TO,
	MOVED_FROM,
	MOVE,
	MOVE_SELF,
	CREATE,
	DELETE,
	DELETE_SELF,
	UNMOUNT,
}

/* ERRORS */
const (
	NOT_INSTALLED  = "inotifywait is not installed"
	OPT_NIL        = "optionsInotify is nil"
	DIR_EMPTY      = "directory is empty"
	INVALID_EVENT  = "invalid event"
	INVALID_OUTPUT = "invalid output"
	DIR_NOT_EXISTS = "directory does not exists"
)

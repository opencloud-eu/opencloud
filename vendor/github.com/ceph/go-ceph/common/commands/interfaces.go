package commands

// MgrCommander in an interface for the API needed to execute JSON formatted
// commands on the ceph mgr.
type MgrCommander interface {
	MgrCommand(buf [][]byte) ([]byte, string, error)
}

// MgrBufferCommander is an interface for the API needed to execute JSON
// formatted commands with an input buffer on the Ceph mgr.
type MgrBufferCommander interface {
	MgrCommandWithInputBuffer([][]byte, []byte) ([]byte, string, error)
}

// MonCommander is an interface for the API needed to execute JSON formatted
// commands on the ceph mon(s).
type MonCommander interface {
	MonCommand(buf []byte) ([]byte, string, error)
}

// MonBufferCommander is an interface for the API needed to execute JSON
// formatted commands with an input buffer on the Ceph mon(s).
type MonBufferCommander interface {
	MonCommandWithInputBuffer([]byte, []byte) ([]byte, string, error)
}

// RadosCommander provides an interface for APIs needed to execute JSON
// formatted commands on the Ceph cluster.
type RadosCommander interface {
	MgrCommander
	MonCommander
}

// RadosBufferCommander provides an interface for APIs that need to execute
// JSON formatted commands with an input buffer on the Ceph cluster.
type RadosBufferCommander interface {
	RadosCommander
	MgrBufferCommander
	MonBufferCommander
}

//go:build !(octopus || pacific || quincy || reef || squid) && ceph_preview

package admin

// SubVolumeSnapshotPath returns the path for a snapshot from the source subvolume.
//
// Similar To:
//
//	ceph fs subvolume snapshot getpath <volume> --group-name=<group> <source> <name>
func (fsa *FSAdmin) SubVolumeSnapshotPath(volume, group, source, name string) (string, error) {
	m := map[string]string{
		"prefix":    "fs subvolume snapshot getpath",
		"vol_name":  volume,
		"sub_name":  source,
		"snap_name": name,
		"format":    "json",
	}
	if group != NoGroup {
		m["group_name"] = group
	}
	return parsePathResponse(fsa.marshalMgrCommand(m))
}

package comfunc

import "path/filepath"

// JoinPaths2 elements, like the filepath.Join()
func JoinPaths2(basePath string, elems []string) string {
	paths := make([]string, len(elems)+1)
	paths[0] = basePath
	copy(paths[1:], elems)
	return filepath.Join(paths...)
}

// JoinPaths3 elements, like the filepath.Join()
func JoinPaths3(basePath, secPath string, elems []string) string {
	paths := make([]string, len(elems)+2)
	paths[0] = basePath
	paths[1] = secPath
	copy(paths[2:], elems)
	return filepath.Join(paths...)
}

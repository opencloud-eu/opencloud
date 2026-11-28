package revisions

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/lookup"
	"github.com/test-go/testify/require"
)

var (
	_basePath = "/spaces/8f/638374-6ea8-4f0d-80c4-66d9b49830a5/nodes/"
)

// func TestInit(t *testing.T) {
// initialize(10, 2)
// defer os.RemoveAll("test_temp")
// }

func TestGlob30(t *testing.T)  { test(t, 10, 2, glob) }
func TestGlob80(t *testing.T)  { test(t, 20, 3, glob) }
func TestGlob250(t *testing.T) { test(t, 50, 4, glob) }
func TestGlob600(t *testing.T) { test(t, 100, 5, glob) }

func TestWalk30(t *testing.T)  { test(t, 10, 2, walk) }
func TestWalk80(t *testing.T)  { test(t, 20, 3, walk) }
func TestWalk250(t *testing.T) { test(t, 50, 4, walk) }
func TestWalk600(t *testing.T) { test(t, 100, 5, walk) }

func TestList30(t *testing.T)  { test(t, 10, 2, list2) }
func TestList80(t *testing.T)  { test(t, 20, 3, list10) }
func TestList250(t *testing.T) { test(t, 50, 4, list20) }
func TestList600(t *testing.T) { test(t, 100, 5, list2) }

func TestGlobWorkers30(t *testing.T)  { test(t, 10, 2, globWorkersD1) }
func TestGlobWorkers80(t *testing.T)  { test(t, 20, 3, globWorkersD2) }
func TestGlobWorkers250(t *testing.T) { test(t, 50, 4, globWorkersD4) }
func TestGlobWorkers600(t *testing.T) { test(t, 100, 5, globWorkersD2) }

func BenchmarkGlob30(b *testing.B)        { benchmark(b, 10, 2, glob) }
func BenchmarkWalk30(b *testing.B)        { benchmark(b, 10, 2, walk) }
func BenchmarkList30(b *testing.B)        { benchmark(b, 10, 2, list2) }
func BenchmarkGlobWorkers30(b *testing.B) { benchmark(b, 10, 2, globWorkersD2) }

func BenchmarkGlob80(b *testing.B)        { benchmark(b, 20, 3, glob) }
func BenchmarkWalk80(b *testing.B)        { benchmark(b, 20, 3, walk) }
func BenchmarkList80(b *testing.B)        { benchmark(b, 20, 3, list2) }
func BenchmarkGlobWorkers80(b *testing.B) { benchmark(b, 20, 3, globWorkersD2) }

func BenchmarkGlob250(b *testing.B)        { benchmark(b, 50, 4, glob) }
func BenchmarkWalk250(b *testing.B)        { benchmark(b, 50, 4, walk) }
func BenchmarkList250(b *testing.B)        { benchmark(b, 50, 4, list2) }
func BenchmarkGlobWorkers250(b *testing.B) { benchmark(b, 50, 4, globWorkersD2) }

func BenchmarkGlobAT600(b *testing.B)          { benchmark(b, 100, 5, glob) }
func BenchmarkWalkAT600(b *testing.B)          { benchmark(b, 100, 5, walk) }
func BenchmarkList2AT600(b *testing.B)         { benchmark(b, 100, 5, list2) }
func BenchmarkList10AT600(b *testing.B)        { benchmark(b, 100, 5, list10) }
func BenchmarkList20AT600(b *testing.B)        { benchmark(b, 100, 5, list20) }
func BenchmarkGlobWorkersD1AT600(b *testing.B) { benchmark(b, 100, 5, globWorkersD1) }
func BenchmarkGlobWorkersD2AT600(b *testing.B) { benchmark(b, 100, 5, globWorkersD2) }
func BenchmarkGlobWorkersD4AT600(b *testing.B) { benchmark(b, 100, 5, globWorkersD4) }

func BenchmarkGlobAT22000(b *testing.B)          { benchmark(b, 2000, 10, glob) }
func BenchmarkWalkAT22000(b *testing.B)          { benchmark(b, 2000, 10, walk) }
func BenchmarkList2AT22000(b *testing.B)         { benchmark(b, 2000, 10, list2) }
func BenchmarkList10AT22000(b *testing.B)        { benchmark(b, 2000, 10, list10) }
func BenchmarkList20AT22000(b *testing.B)        { benchmark(b, 2000, 10, list20) }
func BenchmarkGlobWorkersD1AT22000(b *testing.B) { benchmark(b, 2000, 10, globWorkersD1) }
func BenchmarkGlobWorkersD2AT22000(b *testing.B) { benchmark(b, 2000, 10, globWorkersD2) }
func BenchmarkGlobWorkersD4AT22000(b *testing.B) { benchmark(b, 2000, 10, globWorkersD4) }

func BenchmarkGlob110000(b *testing.B)        { benchmark(b, 10000, 10, glob) }
func BenchmarkWalk110000(b *testing.B)        { benchmark(b, 10000, 10, walk) }
func BenchmarkList110000(b *testing.B)        { benchmark(b, 10000, 10, list2) }
func BenchmarkGlobWorkers110000(b *testing.B) { benchmark(b, 10000, 10, globWorkersD2) }

func benchmark(b *testing.B, numNodes int, numRevisions int, f func(string) <-chan string) {
	base := initialize(numNodes, numRevisions)
	defer os.RemoveAll(base)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := f(base)
		PurgeRevisions(ch, nil, false, false)
	}
	b.StopTimer()
}

func test(t *testing.T, numNodes int, numRevisions int, f func(string) <-chan string) {
	base := initialize(numNodes, numRevisions)
	defer os.RemoveAll(base)

	ch := f(base)
	_, _, revisions := PurgeRevisions(ch, nil, false, false)
	require.Equal(t, numNodes*numRevisions, revisions, "Deleted Revisions")
}

func glob(base string) <-chan string {
	return Glob(base + _basePath + "*/*/*/*/*")
}

func walk(base string) <-chan string {
	return Walk(base + _basePath)
}

func list2(base string) <-chan string {
	return List(base+_basePath, 2)
}

func list10(base string) <-chan string {
	return List(base+_basePath, 10)
}

func list20(base string) <-chan string {
	return List(base+_basePath, 20)
}

func globWorkersD1(base string) <-chan string {
	return GlobWorkers(base+_basePath, "*", "/*/*/*/*")
}

func globWorkersD2(base string) <-chan string {
	return GlobWorkers(base+_basePath, "*/*", "/*/*/*")
}

func globWorkersD4(base string) <-chan string {
	return GlobWorkers(base+_basePath, "*/*/*/*", "/*")
}

func initialize(numNodes int, numRevisions int) string {
	base := "test_temp_" + uuid.New().String()
	if err := os.Mkdir(base, os.ModePerm); err != nil {
		fmt.Println("Error creating test_temp directory", err)
		os.RemoveAll(base)
		os.Exit(1)
	}

	// create base path
	if err := os.MkdirAll(base+_basePath, fs.ModePerm); err != nil {
		fmt.Println("Error creating base path", err)
		os.RemoveAll(base)
		os.Exit(1)
	}

	for i := 0; i < numNodes; i++ {
		path := lookup.Pathify(uuid.New().String(), 4, 2)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(base+_basePath+dir, fs.ModePerm); err != nil {
			fmt.Println("Error creating test_temp directory", err)
			os.RemoveAll(base)
			os.Exit(1)
		}

		if _, err := os.Create(base + _basePath + path); err != nil {
			fmt.Println("Error creating file", err)
			os.RemoveAll(base)
			os.Exit(1)
		}
		for i := 0; i < numRevisions; i++ {
			os.Create(base + _basePath + path + ".REV.2024-05-22T07:32:53.89969" + strconv.Itoa(i) + "Z")
		}
	}
	return base
}

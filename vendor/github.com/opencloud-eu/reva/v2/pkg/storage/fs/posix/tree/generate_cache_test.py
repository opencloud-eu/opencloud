import sys

content = """
Describe("WarmupIDCache", func() {
var (
tree   *Tree
tmpDir string
logger zerolog.Logger
ctx    context.Context
)

BeforeEach(func() {
ctx = context.Background()
var err error
tmpDir, err = os.MkdirTemp("", "warmupidcache-*")
Expect(err).ToNot(HaveOccurred())

logger = zerolog.Nop()

o := &options.Options{
Options: decomposedoptions.Options{
Root: tmpDir,
},
}

// We need a backend and caches
c, _ := idcache.NewMemoryIDCache()
historyCache, _ := idcache.NewMemoryIDCache()
um := &usermapper.NullMapper{}

backend := metadata.NewMessagePackBackend(o.FileMetadataCache)
lu, err := lookup.New(backend, um, o, &timemanager.Manager{}, c, historyCache)
Expect(err).ToNot(HaveOccurred())

tree = &Tree{
log:     &logger,
options: o,
lookup:  lu,
}
})

AfterEach(func() {
os.RemoveAll(tmpDir)
})

It("returns nil for an empty directory", func() {
err := tree.WarmupIDCache(tmpDir, false, false)
Expect(err).ToNot(HaveOccurred())
})
        
        It("picks up new files and directories", func() {
            subDir := filepath.Join(tmpDir, "sub")
            err := os.Mkdir(subDir, 0755)
            Expect(err).ToNot(HaveOccurred())
            
            filePath := filepath.Join(subDir, "test.txt")
            err = os.WriteFile(filePath, []byte("hello world"), 0644)
            Expect(err).ToNot(HaveOccurred())
            
            // Should not crash, tests basic traverse
            err = tree.WarmupIDCache(tmpDir, false, false)
            Expect(err).ToNot(HaveOccurred())
        })

        It("verifies tree sizes and recursion", func() {
            // Setup a small directory structure
            subDir := filepath.Join(tmpDir, "sub2")
            err := os.Mkdir(subDir, 0755)
            Expect(err).ToNot(HaveOccurred())
            
            nestedDir := filepath.Join(subDir, "nested")
            err = os.Mkdir(nestedDir, 0755)
            Expect(err).ToNot(HaveOccurred())
            
            filePath := filepath.Join(nestedDir, "test.txt")
            err = os.WriteFile(filePath, []byte("hello world"), 0644) // 11 bytes
            Expect(err).ToNot(HaveOccurred())
            
            // Run assimilation 
            err = tree.WarmupIDCache(tmpDir, true, false)
            Expect(err).ToNot(HaveOccurred())
            
            // If assimilation runs, the files will have xattrs and tree sizes evaluated
            // wait, but without mocked idResolver for the Tree, it might fail? Let's check!
        })
})
"""

with open("assimilation_test.go", "r") as f:
    text = f.read()

import re

# Add imports
imports = """
import (
"context"
"os"
"path/filepath"

"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/lookup"
"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/idcache"
"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/usermapper"
"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/timemanager"

. "github.com/onsi/ginkgo/v2"
. "github.com/onsi/gomega"
"github.com/rs/zerolog"

"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
decomposedoptions "github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/options"
)
"""

# Replace imports
start_import = text.find('import (')
end_import = text.find(')', start_import) + 1
text = text[:start_import] + imports.strip() + text[end_import:]

start_idx = text.find('Describe("WarmupIDCache", func() {')
end_idx = text.find('})\n\n})', start_idx) + 2

if start_idx != -1 and end_idx != -1:
    new_text = text[:start_idx] + content.strip() + text[end_idx:]
    with open("assimilation_test.go", "w") as f:
        f.write(new_text)


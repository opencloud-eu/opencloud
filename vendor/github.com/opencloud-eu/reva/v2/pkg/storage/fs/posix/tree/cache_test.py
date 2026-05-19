import re
with open("/home/andre/src/opencloud/reva/pkg/storage/fs/posix/tree/assimilation_test.go", "r") as f:
    text = f.read()

new_content = """
Describe("WarmupIDCache", func() {
var (
tree   *Tree
tmpDir string
logger zerolog.Logger
)

BeforeEach(func() {
var err error
tmpDir, err = os.MkdirTemp("", "warmupidcache-*")
Expect(err).ToNot(HaveOccurred())

logger = zerolog.Nop()

tree = &Tree{
log:    &logger,
options: &options.Options{
Options: decomposedoptions.Options{
Root: tmpDir,
},
},
}
})

AfterEach(func() {
os.RemoveAll(tmpDir)
})

It("returns nil for an empty directory", func() {
err := tree.WarmupIDCache(tmpDir, false, false)
Expect(err).ToNot(HaveOccurred())
})
})
"""

text = re.sub(r'Describe\("WarmupIDCache", func\(\).*$', new_content, text, flags=re.DOTALL)
text += "\n})\n"
with open("/home/andre/src/opencloud/reva/pkg/storage/fs/posix/tree/assimilation_test.go", "w") as f:
    f.write(text)


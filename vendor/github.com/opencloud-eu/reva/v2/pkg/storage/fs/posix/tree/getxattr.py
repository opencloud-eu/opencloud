with open("assimilation_test.go", "r") as f:
    text = f.read()

import re

# We will inject the xattr check.
insert = """// verify that tree sizes are updated
// Since we used assimilate=true, the treesize xattr on sub2 and nested should be 11.

b, err := env.Lookup.MetadataBackend().Get(env.Ctx, subDir, "user.oc.treesize")
Expect(err).ToNot(HaveOccurred())
Expect(string(b)).To(Equal("11"))

b, err = env.Lookup.MetadataBackend().Get(env.Ctx, nestedDir, "user.oc.treesize")
Expect(err).ToNot(HaveOccurred())
Expect(string(b)).To(Equal("11"))"""

# inject right after env.Tree.WarmupIDCache call
text = text.replace("err = env.Tree.WarmupIDCache(tmpDir+\"/users/admin\", true, false)\n\t\tExpect(err).ToNot(HaveOccurred())", "err = env.Tree.WarmupIDCache(tmpDir+\"/users/admin\", true, false)\n\t\tExpect(err).ToNot(HaveOccurred())\n\n" + insert)

with open("assimilation_test.go", "w") as f:
    f.write(text)


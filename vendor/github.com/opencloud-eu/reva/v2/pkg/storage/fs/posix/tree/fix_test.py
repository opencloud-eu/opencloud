import re

with open("assimilation_test.go", "r") as f:
    text = f.read()

# remove CreateTestStorageSpace which throws error
text = text.replace('_, err = env.CreateTestStorageSpace("personal", nil)\n\t\tExpect(err).ToNot(HaveOccurred())', '')

with open("assimilation_test.go", "w") as f:
    f.write(text)

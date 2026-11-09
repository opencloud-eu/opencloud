recursive = true
output_file = "tw.txt"
extensions = [".go"]
exclude_dirs {
  items = [
    "_examples", "_readme", "_lab", "_tmp", "pkg", "lab", "cmd", "test.txt", "tmp",
    "_readme", "pkg", "renderer"
  ]
}
exclude_files {
 items  = ["README.md","README_LEGACY.md","MIGRATION.md","test.hcl","csv.go"]
}
use_gitignore = true
detailed = true
go_mode = "all"
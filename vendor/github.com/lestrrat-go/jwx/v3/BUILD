load("@gazelle//:def.bzl", "gazelle")
load("@rules_go//go:def.bzl", "go_library", "go_test")

# gazelle:prefix github.com/lestrrat-go/jwx/v3
# gazelle:go_naming_convention import_alias

# Scratch directories that are not part of the module. Without these,
# gazelle walks bazel's own output tree under .gauntlet and rewrites every
# BUILD file to point at copies of the repo it finds in there.
# gazelle:exclude .gauntlet
# gazelle:exclude .tmp

gazelle(name = "gazelle")

go_library(
    name = "jwx",
    srcs = [
        "format.go",
        "formatkind_string_gen.go",
        "jwx.go",
        "options.go",
    ],
    importpath = "github.com/lestrrat-go/jwx/v3",
    visibility = ["//visibility:public"],
    deps = [
        "//internal/json",
        "//internal/tokens",
        "@com_github_lestrrat_go_option_v2//:option",
    ],
)

go_test(
    name = "jwx_test",
    srcs = ["jwx_test.go"],
    deps = [
        ":jwx",
        "//internal/jose",
        "//internal/json",
        "//internal/jwxtest",
        "//jwa",
        "//jwe",
        "//jwk",
        "//jwk/ecdsa",
        "//jws",
        "@com_github_stretchr_testify//require",
    ],
)

alias(
    name = "go_default_library",
    actual = ":jwx",
    visibility = ["//visibility:public"],
)

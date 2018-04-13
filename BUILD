load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "element.go",
        "soap.go",
    ],
    importpath = "aqwari.net/exp/soap",
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = ["example_test.go"],
    embed = [":go_default_library"],
)

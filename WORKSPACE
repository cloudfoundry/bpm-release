workspace(name = "bpm")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
http_archive(
    name = "io_bazel_rules_go",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.13.0/rules_go-0.13.0.tar.gz"],
    sha256 = "ba79c532ac400cefd1859cbc8a9829346aa69e3b99482cd5a54432092cbc3933",
)
load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()

http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.13.0/bazel-gazelle-0.13.0.tar.gz"],
    sha256 = "bc653d3e058964a5a26dcad02b6c72d7d63e6bb88d94704990b908a1445b8758",
)
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
gazelle_dependencies()

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

git_repository(
    name = "com_github_xoebus_rules_bosh",
    remote = "https://github.com/xoebus/rules_bosh.git",
    commit = "764a758d2a05bbeb39730961ec863adc2dd22cf7",
)

http_file(
    name = "runc",
    url = "https://github.com/opencontainers/runc/releases/download/v1.0.0-rc5/runc.amd64",
    sha256 = "eaa9c9518cc4b041eea83d8ef83aad0a347af913c65337abe5b94b636183a251",
    executable = True,
)

load("@bazel_gazelle//:def.bzl", "gazelle", "gazelle_binary", "DEFAULT_LANGUAGES")

# gazelle:prefix github.com/bazelbuild/rules_closure/gazelle
gazelle(
    name = "gazelle",
    gazelle = ":gazelle_js",
    extra_args = [
        "gazelle",
    ],
)

# In your project you should define your own Gazelle binary. Follow the
# instructions at
# https://github.com/bazelbuild/bazel-gazelle/blob/master/extend.rst to add the
# languages you depend on. You can `bazel run @io_bazel_rules_closure//gazelle`
# target in this file if you just need golang, proto and JS support.

gazelle_binary(
    name = "gazelle_js",
    languages = DEFAULT_LANGUAGES + [
        "@io_bazel_rules_closure//gazelle/closure_js:go_default_library",
    ],
    visibility = ["//visibility:private"],
)

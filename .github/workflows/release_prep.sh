#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Builds the release archive and prints release notes to stdout.
set -o errexit -o nounset -o pipefail

TAG="$1"
VERSION="${TAG#v}"
ARCHIVE="bazel_hdlf_tools-${TAG}.zip"

# The archive holds the tracked files of the tagged commit and nothing else,
# so nothing the release run creates in the checkout can leak into it.
git archive --format=zip --output="${ARCHIVE}" "${TAG}"

cat <<EOF
## Using Bzlmod

\`\`\`starlark
bazel_dep(name = "hdlf_tools", version = "${VERSION}", dev_dependency = True)
\`\`\`
EOF

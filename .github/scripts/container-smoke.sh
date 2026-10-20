#!/usr/bin/env bash
set -euo pipefail

# CI builds this image with the default dev/unknown metadata and loads it locally.
image=${1:-pdfgen:test}
docker run --rm "$image" --version | grep 'pdfgen dev (commit unknown)'
test "$(docker image inspect "$image" --format '{{.Config.User}}')" = '65532:65532'

smoke_dir=$(mktemp -d)
trap 'rm -rf "$smoke_dir"' EXIT
# The temporary mount must be writable by both UID 65532 and the runner user.
chmod 777 "$smoke_dir"
printf '<html><body><p>Bundled fonts: Hello PDF</p></body></html>' > "$smoke_dir/input.html"

# Exercise the default container user and the UID override shown in the README.
docker run --rm -v "$smoke_dir:/work" "$image" input.html default-user.pdf
docker run --rm --user "$(id -u):$(id -g)" -v "$smoke_dir:/work" "$image" input.html output.pdf

# Keep PDF bytes on stdout so pipelines can redirect them on the host.
docker run --rm -i "$image" - - < "$smoke_dir/input.html" > "$smoke_dir/stdout.pdf"

# This is a basic runtime check; rendering correctness is covered by the Go tests.
for file in default-user output stdout; do
  test "$(head -c 5 "$smoke_dir/$file.pdf")" = '%PDF-'
  test "$(wc -c < "$smoke_dir/$file.pdf")" -gt 1000
done

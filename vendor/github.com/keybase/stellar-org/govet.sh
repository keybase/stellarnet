#! /bin/bash
set -e

printf "Running go vet...\n"
go vet -all -composites=false -unreachable=false -tests=false ./...

printf "Running go vet shadow...\n"
command -v shadow >/dev/null 2>&1 || (
  dir=$(mktemp -d)
  pushd $dir
  go mod init shadow
  # The go vet -vettool option was added in 1.12, and broken in the initial
  # 1.13 release for any vettool that doesn't support the unsafeptr flag. The
  # shadow tool doesn't support the unsafeptr flag.  Until either support for
  # unsafeptr is added to shadow or passing through that flag is removed from
  # go vet we can use the analyzer by wrapping it in our own main that ignores
  # unsafeptr. This temporary work around was suggested here:
  # https://github.com/golang/go/issues/34053
  gofmt > main.go <<-EOF
    package main
    import "flag"
    import "golang.org/x/tools/go/analysis/passes/shadow"
    import "golang.org/x/tools/go/analysis/singlechecker"
    func init() { flag.String("unsafeptr", "", "") }
    func main() { singlechecker.Main(shadow.Analyzer) }
EOF
  go install
  popd
)

go vet -vettool=$(which shadow) ./...

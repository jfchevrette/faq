#!/bin/bash
set -e 

dep ensure
go vet $(go list ./...)
diff <(goimports -d $(find . -type f -name '*.go' -not -path "./vendor/*")) <(printf "")
(for d in $(go list ./...); do diff <(golint $d) <(printf "") || exit 1;  done)
go test -v ./...

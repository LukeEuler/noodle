all: build

commit := `git describe --always`
buildtime := `date +%FT%T%z`
MODULE = "github.com/LukeEuler/noodle"
ARGUMENTS_PATH = github.com/LukeEuler/dolly/common

LD_FLAGS := -ldflags "-X ${ARGUMENTS_PATH}.commit=${commit} -X ${ARGUMENTS_PATH}.buildTime=${buildtime}"

GOIMPORTS := $(shell command -v goimports 2> /dev/null)
CILINT := $(shell command -v golangci-lint 2> /dev/null)

style:
ifndef GOIMPORTS
	$(error "goimports is not available please install goimports")
endif
	! find . -path ./vendor -prune -o -name '*.go' -print | xargs goimports -d -local ${MODULE} | grep '^'

format:
ifndef GOIMPORTS
	$(error "goimports is not available please install goimports")
endif
	find . -path ./vendor -prune -o -name '*.go' -print | xargs goimports -l -local ${MODULE} | xargs goimports -l -local ${MODULE} -w

cilint:
ifndef CILINT
	$(error "golangci-lint is not available please install golangci-lint")
endif
	golangci-lint run --timeout 5m0s

test: style cilint
	go test -cover ./...

build: test
	go build ${LD_FLAGS} -o build/noodle app/main.go

linux: test
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LD_FLAGS} -o build/noodle app/main.go

.PHONY: linux style format test build

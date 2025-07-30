APP=health
BINARY_NAME=health

PACKAGE=github.com/health
VERSION=$(shell git describe --tags --always --abbrev=0 --match='v[0-9]*.[0-9]*.[0-9]*' 2> /dev/null | sed 's/^.//')
COMMIT=$(shell git rev-parse --short HEAD)
DATE=$(shell date '+%Y-%m-%d %H:%M:%S')

LDFLAGS=-X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'


HOOKS= \
           https://raw.githubusercontent.com/thisdougb/git-time-hooks/main/commit-msg \
           https://raw.githubusercontent.com/thisdougb/git-time-hooks/main/prepare-commit-msg

githooks:
	@cd .git/hooks && \
		for i in $(HOOKS); do \
                echo "installing git hook $$i"; \
                curl -sO $$i; \
                chmod +x "`echo $$i | rev | cut -f1 -d'/' | rev`"; \
        done
	git config branch.master.mergeOptions "--squash"

test:
	go test -count=1 -tags dev ./...

build:
	CGO_ENABLED=1 go build -o bin/$(APP) -ldflags="$(LDFLAGS)" main.go

buildlinux: test
	CC="zig cc -target x86_64-linux" CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o bin/$(APP) -ldflags="$(LDFLAGS)" main.go


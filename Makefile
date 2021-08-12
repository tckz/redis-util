.PHONY: dist test clean all sonar

export GO111MODULE=on

ifeq ($(GO_CMD),)
GO_CMD:=go
endif

VERSION := $(shell git describe --always)
GO_BUILD := CGO_ENABLED=0 $(GO_CMD) build -ldflags "-X main.version=$(VERSION)"

DIST_HGETALL=dist/hgetall
DIST_HSET=dist/hset
DIST_GET=dist/get
DIST_SET=dist/set
DIST_ZADD=dist/zadd
DIST_DEL=dist/del
DIST_PTTL=dist/pttl
DIST_SCAN=dist/scan
DIST_PEXPIREAT=dist/pexpireat

TARGETS=\
	$(DIST_HGETALL) \
	$(DIST_GET) \
	$(DIST_SET) \
	$(DIST_ZADD) \
	$(DIST_DEL) \
	$(DIST_PTTL) \
	$(DIST_SCAN) \
	$(DIST_PEXPIREAT) \
	$(DIST_HSET)

SRCS_OTHER := $(shell find . \
	-type d -name cmd -prune -o \
	-type d -name vendor -prune -o \
	-type f -name "*.go" -print) go.mod

all: $(TARGETS)
	@echo "$@ done." 1>&2

sonar: test-detail
	./gradlew sonar
	@echo "$@ done." 1>&2

clean:
	/bin/rm -f $(TARGETS)
	@echo "$@ done." 1>&2

test:
	go test -v
	@echo "$@ done." 1>&2

test-detail:
	$(GO_CMD) test -coverprofile=reports/coverage.out -json > reports/test.json
	@echo "$@ done." 1>&2

$(DIST_HGETALL): cmd/hgetall/* $(SRCS_OTHER)
	$(GO_BUILD) -o $@ ./cmd/hgetall/

$(DIST_HSET): cmd/hset/* $(SRCS_OTHER)
	$(GO_BUILD) -o $@ ./cmd/hset/

$(DIST_GET): cmd/get/* $(SRCS_OTHER)
	$(GO_BUILD) -o $@ ./cmd/get/

$(DIST_SET): cmd/set/* $(SRCS_OTHER)
	$(GO_BUILD) -o $@ ./cmd/set/

$(DIST_ZADD): cmd/zadd/* $(SRCS_OTHER)
	$(GO_BUILD) -o $@ ./cmd/zadd/

$(DIST_DEL): cmd/del/* $(SRCS_OTHER)
	$(GO_BUILD) -o $@ ./cmd/del/

$(DIST_PTTL): cmd/pttl/* $(SRCS_OTHER)
	$(GO_BUILD) -o $@ ./cmd/pttl/

$(DIST_PEXPIREAT): cmd/pexpireat/* $(SRCS_OTHER)
	$(GO_BUILD) -o $@ ./cmd/pexpireat/

$(DIST_SCAN): cmd/scan/* $(SRCS_OTHER)
	$(GO_BUILD) -o $@ ./cmd/scan/


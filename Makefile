.PHONY: dist test clean all sonar
.SUFFIXES: .proto .pb.go .go

DIST_HGETALL=dist/hgetall
DIST_HMSET=dist/hmset
DIST_GET=dist/get
DIST_SET=dist/set
DIST_DEL=dist/del
DIST_PTTL=dist/pttl
DIST_PEXPIREAT=dist/pexpireat

TARGETS=\
	$(DIST_HGETALL) \
	$(DIST_GET) \
	$(DIST_SET) \
	$(DIST_DEL) \
	$(DIST_PTTL) \
	$(DIST_PEXPIREAT) \
	$(DIST_HMSET)

SRCS_OTHER=$(shell find . -type d -name vendor -prune -o -type d -name cmd -prune -o -type f -name "*.go" -print) go.mod

all: $(TARGETS)
	@echo "$@ done."

sonar: test
	./gradlew sonar
	@echo "$@ done."

clean:
	/bin/rm -f $(TARGETS)
	@echo "$@ done."

# if block ends with... : I want to narrownize scope within if block.
lint:
	golint ./... | (egrep -v "(if block ends with a return statement)" || :)

test:
	GO111MODULE=on go test -coverprofile=reports/coverage.out -json > reports/test.json
	@echo "$@ done."

test-plain:
	GO111MODULE=on go test -v
	@echo "$@ done."

$(DIST_HGETALL): cmd/hgetall/* $(SRCS_OTHER)
	GO111MODULE=on GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ -ldflags "-X main.version=`git describe --tags --always`" ./cmd/hgetall/
	@echo "$@ done."

$(DIST_HMSET): cmd/hmset/* $(SRCS_OTHER)
	GO111MODULE=on GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ -ldflags "-X main.version=`git describe --tags --always`" ./cmd/hmset/
	@echo "$@ done."

$(DIST_GET): cmd/get/* $(SRCS_OTHER)
	GO111MODULE=on GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ -ldflags "-X main.version=`git describe --tags --always`" ./cmd/get/
	@echo "$@ done."

$(DIST_SET): cmd/set/* $(SRCS_OTHER)
	GO111MODULE=on GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ -ldflags "-X main.version=`git describe --tags --always`" ./cmd/set/
	@echo "$@ done."

$(DIST_DEL): cmd/del/* $(SRCS_OTHER)
	GO111MODULE=on GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ -ldflags "-X main.version=`git describe --tags --always`" ./cmd/del/
	@echo "$@ done."

$(DIST_PTTL): cmd/pttl/* $(SRCS_OTHER)
	GO111MODULE=on GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ -ldflags "-X main.version=`git describe --tags --always`" ./cmd/pttl/
	@echo "$@ done."

$(DIST_PEXPIREAT): cmd/pexpireat/* $(SRCS_OTHER)
	GO111MODULE=on GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ -ldflags "-X main.version=`git describe --tags --always`" ./cmd/pexpireat/
	@echo "$@ done."



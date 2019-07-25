.PHONY: dist test clean all
.SUFFIXES: .proto .pb.go .go

DIST_HGETALL=dist/hgetall

TARGETS=\
	$(DIST_HGETALL)

SRCS_OTHER=$(shell find . -type d -name vendor -prune -o -type d -name cmd -prune -o -type f -name "*.go" -print)

all: $(TARGETS)
	@echo "$@ done."

clean:
	/bin/rm -f $(TARGETS)
	@echo "$@ done."

test:
	go test -coverprofile=reports/coverage.out -json > reports/test.json
	@echo "$@ done."

$(DIST_HGETALL): cmd/hgetall/* go.mod $(SRCS_OTHER)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ -ldflags "-X main.version=`git describe --tags --always`" ./cmd/hgetall/
	@echo "$@ done."



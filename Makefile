.PHONY: dist test clean

TARGETS=\
	dist/gapi \

SRCS_OTHER = \
	$(wildcard */*.go) \
	$(wildcard *.go) \
	go.mod

all: $(TARGETS)
	@echo "$@ done."

clean:
	/bin/rm -f $(TARGETS)
	@echo "$@ done."

dist/gapi: cmd/gapi/* $(SRCS_OTHER)
	go build -o $@ -ldflags "-X main.version=`git describe  --tags --always`" ./cmd/gapi/
	@echo "$@ done."

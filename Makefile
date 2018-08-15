.PHONY: dist test clean

TARGETS=\
	dist/gapi \

SRCS_OTHER = \
	$(wildcard */*.go) \
	$(wildcard *.go)

all: $(TARGETS)
	@echo "$@ done."

clean:
	/bin/rm -f $(TARGETS)
	@echo "$@ done."

dist/gapi: cmd/gapi/main.go $(SRCS_OTHER)
	if [ ! -d dist ];then mkdir dist; fi
	go build -o $@ -ldflags "-X main.version=`git show -s --format=%H`" $<
	@echo "$@ done."

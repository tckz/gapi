.PHONY: dist test clean

TARGETS=\
	dist/gapi \

SRCS_OTHER = \
	$(wildcard */*.go) \
	$(wildcard *.go)

SRCS_GAPI = \
	$(wildcard cmd/*/*.go)

all: $(TARGETS)
	@echo "$@ done."

clean:
	/bin/rm -f $(TARGETS)
	@echo "$@ done."

dist/gapi: $(SRCS_GAPI) $(SRCS_OTHER)
	if [ ! -d dist ];then mkdir dist; fi
	go build -o $@ -ldflags "-X main.version=`git show -s --format=%H`" $<
	@echo "$@ done."

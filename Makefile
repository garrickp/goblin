PROJECTS = sleep basename dc
BINS = $(patsubst %, bin/%, $(PROJECTS))

GO = /usr/local/go/bin/go
GOFMT = /usr/local/go/bin/gofmt
RM = /bin/rm

GOPATH = $(shell pwd)
GOBIN = $(shell pwd)/bin

all: ${BINS}

bin/%: src/%/*.go
	env GOPATH=${GOPATH} GOBIN=${GOBIN} ${GO} install $^

test:
	env GOPATH=${GOPATH} ${GO} test ${PROJECTS}

format:
	${GOFMT} -w ${SOURCES}

clean:
	${RM} ${BINS}

.PHONY: all format clean test

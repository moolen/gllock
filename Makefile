BINARY=gllock

VERSION=`git describe --tags`

LDFLAGS=-ldflags "-w -s -X main.version=${VERSION}"

# Builds the project
build:
	packr build ${LDFLAGS} -o ${BINARY}

# Cleans our project: deletes binaries
clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

.PHONY: clean install

VERSION=`git describe --tags --always --dirty`
VERSION_VARIABLE_NAME=verbot/constants.VERSION
VERSION_VARIABLE_BUILD_FLAG=-ldflags "-X ${VERSION_VARIABLE_NAME}=${VERSION}"
BUILD_COMMAND:=go build
BUILD_WITH_VERSION_COMMAND=${BUILD_COMMAND} ${VERSION_VARIABLE_BUILD_FLAG}
BIN_DIRECTORY:=bin

all: verbot

verbot: ## Build verbot
	@ ${BUILD_WITH_VERSION_COMMAND} -o ${BIN_DIRECTORY}/$@ cmd/$@/main.go

deps: ## Download all dependencies
	@ ${GO} get
	@ which CompileDaemon 2>&1 > /dev/null || ( cd $(mktemp -d);  go get github.com/githubnemo/CompileDaemon)


dev: deps ## Build, run and rebuild after any change
	@ CompileDaemon -build "make verbot" -command "bin/verbot -c example/verbot.toml" -exclude-dir=".git"



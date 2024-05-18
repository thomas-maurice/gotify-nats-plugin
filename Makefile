BUILDDIR=./build
GOTIFY_VERSION=2.4.0
PLUGIN_NAME=nats-plugin
PLUGIN_ENTRY=plugin.go
GO_VERSION=`cat $(BUILDDIR)/gotify-server-go-version`
DOCKER_BUILD_IMAGE=gotify/build
DOCKER_WORKDIR=/proj
DOCKER_RUN=docker run --rm -e GO11MODULE=on -v "$$PWD/.:${DOCKER_WORKDIR}" -w ${DOCKER_WORKDIR}
DOCKER_GO_BUILD=go build -mod=readonly -a -installsuffix cgo -ldflags "$$LD_FLAGS" -buildmode=plugin

download-tools:
	go install github.com/gotify/plugin-api/cmd/gomod-cap

create-build-dir:
	mkdir -p ${BUILDDIR} || true

update-go-mod: create-build-dir
	wget -LO ${BUILDDIR}/gotify-server.mod https://raw.githubusercontent.com/gotify/server/v${GOTIFY_VERSION}/go.mod
	gomod-cap -from ${BUILDDIR}/gotify-server.mod -to go.mod
	rm ${BUILDDIR}/gotify-server.mod || true
	go mod tidy

get-gotify-server-go-version: create-build-dir
	rm ${BUILDDIR}/gotify-server-go-version || true
	wget -LO ${BUILDDIR}/gotify-server-go-version https://raw.githubusercontent.com/gotify/server/v${GOTIFY_VERSION}/GO_VERSION

build-linux-amd64: get-gotify-server-go-version update-go-mod
	${DOCKER_RUN} ${DOCKER_BUILD_IMAGE}:$(GO_VERSION)-linux-amd64 ${DOCKER_GO_BUILD} -o ${BUILDDIR}/${PLUGIN_NAME}-for-${GOTIFY_VERSION}-linux-amd64${FILE_SUFFIX}.so ${DOCKER_WORKDIR}

build: build-linux-amd64

run:
	if ! [ -d data/plugins ]; then mkdir -p data/plugins; fi
	cp ${BUILDDIR}/${PLUGIN_NAME}-for-${GOTIFY_VERSION}-linux-amd64${FILE_SUFFIX}.so data/plugins/gotify-nats.so
	docker run --user 1000:1000 --net host -e GOTIFY_SERVER_PORT=8080 -v ./data:/app/data -v ./gotify-config.yml:/etc/gotify/config.yml -it gotify/server:${GOTIFY_VERSION}

.PHONY: build

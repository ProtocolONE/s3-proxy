ifndef VERBOSE
.SILENT:
endif

override ROOT_DIR = $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

TAG ?= unknown
CACHE_TAG ?= unknown_cache
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED ?= 0
DIND_UID ?= 0
DING_GUID ?= 0

define build_resources
 	find "$(ROOT_DIR)/resources" -maxdepth 1 -mindepth 1 -exec cp -R -f {} $(ROOT_DIR)/artifacts/${1} \;
endef

define go_docker
	. ${ROOT_DIR}/scripts/common.sh ${ROOT_DIR}/scripts ;\
	docker run --rm \
		-v /${ROOT_DIR}:/${ROOT_DIR} \
		-v /$${GO_PATH}/pkg/mod:/$${GO_PATH}/pkg/mod \
		-w /${ROOT_DIR} \
		-e GOPATH=/$${GO_PATH}:/go \
		$${GO_IMAGE}:$${GO_IMAGE_TAG} \
		sh -c 'GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=${CGO_ENABLED} TAG=${TAG} $(subst ",,${1}); if [ "${DIND_UID}" != "0" ]; then chown -R ${DIND_UID}:${DIND_GUID} ${ROOT_DIR}; fi'
endef


up: ## initialize required tools
	. ${ROOT_DIR}/scripts/common.sh ${ROOT_DIR}/scripts ;\
	(docker network inspect $${DOCKER_NETWORK} &>/dev/null && echo "Docker network \"$${DOCKER_NETWORK}\" already created") || \
	(echo "Create docker network" && docker network create $${DOCKER_NETWORK})
	if [ "${DIND}" != "1" ]; then \
		export GO111MODULE=on ;\
		go get github.com/google/wire/cmd/wire@v0.3.0 ;\
    fi;
.PHONY: up

down: dev-docker-compose-down ## reset to zero setting
	. ${ROOT_DIR}/scripts/common.sh ${ROOT_DIR}/scripts ;\
	(docker network inspect $${DOCKER_NETWORK} &>/dev/null && \
	(echo "Delete docker network" && docker network rm $${DOCKER_NETWORK}) || echo "Docker network \"$${DOCKER_NETWORK}\" already deleted")
.PHONY: down

build-resources: ## prepare artifacts for application binary
	$(call build_resources)
.PHONY: build-resources

build: init ## build application
	if [ "${DIND}" = "1" ]; then \
		$(call go_docker,"make build") ;\
    else \
		. ${ROOT_DIR}/scripts/common.sh ${ROOT_DIR}/scripts ;\
		echo "Build with parameters GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=${CGO_ENABLED}" ;\
		$(call build_resources) ;\
        GO111MODULE=on GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=${CGO_ENABLED} \
        go build -mod vendor -ldflags "-X $${GO_PKG}/cmd/version.appVersion=$(TAG)-$$(date -u +%Y%m%d%H%M)" -o "$(ROOT_DIR)/artifacts/bin" main.go ;\
    fi;
.PHONY: build

build-jenkins: build ## build with jenkins
.PHONY: build-jenkins

clean: ## remove generated files, tidy vendor dependencies
	if [ "${DIND}" = "1" ]; then \
		$(call go_docker,"make clean") ;\
    else \
        export GO111MODULE=on ;\
        go mod tidy ;\
    	rm -rf profile.out artifacts/* generated/* vendor ;\
    fi;
.PHONY: clean

dev-build-up: build docker-image-cache dev-docker-compose-up ## create new build and recreate containers in docker-compose
.PHONY: dev-build-up

dev-docker-compose-down: ## stop and remove containers, networks, images, and volumes
	. ${ROOT_DIR}/scripts/common.sh ${ROOT_DIR}/scripts ;\
	TAG=${TAG} DOCKER_NETWORK=$${DOCKER_NETWORK} docker-compose -p $${PROJECT_NAME} -f docker/docker-compose.yml -f docker/docker-compose-local.yml down -v
.PHONY: dev-docker-compose-down

dev-docker-compose-up: ## create and start containers
	. ${ROOT_DIR}/scripts/common.sh ${ROOT_DIR}/scripts ;\
	TAG=${TAG} DOCKER_NETWORK=$${DOCKER_NETWORK} docker-compose -p $${PROJECT_NAME} -f docker/docker-compose.yml -f docker/docker-compose-local.yml up -d
.PHONY: dev-docker-compose-up

dev-test: test lint ## test application in dev env with race and lint
.PHONY: dev-test

dind: ## useful for windows
	if [ "${DIND}" = "1" ]; then \
		echo "Already in DIND" ;\
    else \
	    . ${ROOT_DIR}/scripts/common.sh ${ROOT_DIR}/scripts ;\
	    docker run -it --rm --name dind --privileged \
            -v //var/run/docker.sock://var/run/docker.sock \
            -v /${ROOT_DIR}:/${ROOT_DIR} \
            -v /$${GO_PATH}/pkg/mod:/$${GO_PATH}/pkg/mod \
            -w /${ROOT_DIR} \
            p1hub/dind:latest ;\
    fi;
.PHONY: dind

docker-clean: ## delete previous docker image build
	. ${ROOT_DIR}/scripts/common.sh ${ROOT_DIR}/scripts ;\
	docker rmi $${DOCKER_IMAGE}:${CACHE_TAG} || true ;\
	docker rmi $${DOCKER_IMAGE}:${TAG} || true
.PHONY: docker-clean

docker-image-cache: ## build docker image and tagged as cache
	. ${ROOT_DIR}/scripts/common.sh ${ROOT_DIR}/scripts ;\
	docker build --cache-from $${DOCKER_IMAGE}:${CACHE_TAG} -f "${ROOT_DIR}/docker/app/Dockerfile" -t $${DOCKER_IMAGE}:${TAG} -t $${DOCKER_IMAGE}:${CACHE_TAG} ${ROOT_DIR}
.PHONY: docker-image-cache

docker-image: ## build docker image
	. ${ROOT_DIR}/scripts/common.sh ${ROOT_DIR}/scripts ;\
	docker build --cache-from $${DOCKER_IMAGE}:${CACHE_TAG} -f "${ROOT_DIR}/docker/app/Dockerfile" -t $${DOCKER_IMAGE}:${TAG} ${ROOT_DIR}
.PHONY: docker-image

docker-image-jenkins: docker-image ## build docker image with jenkins
.PHONY: docker-image-jenkins

docker-push: ## push docker image to registry
	. ${ROOT_DIR}/scripts/common.sh ${ROOT_DIR}/scripts ;\
	docker push $${DOCKER_IMAGE}:${TAG}
.PHONY: docker-push

generate: go-generate ## execute all generators
.PHONY: generate

github-build: docker-image docker-push docker-clean ## build application in CI
.PHONY: github-build

github-test: vendor test-with-coverage ## test application in CI
.PHONY: github-test

go-depends: ## view final versions that will be used in a build for all direct and indirect dependencies
	if [ "${DIND}" = "1" ]; then \
		$(call go_docker,"make go-depends") ;\
    else \
        cd $(ROOT_DIR) ;\
        GO111MODULE=on go list -m all ;\
    fi;
.PHONY: go-depends

go-generate: ## go generate
	if [ "${DIND}" = "1" ]; then \
		$(call go_docker,"make go-generate") ;\
    else \
        cd $(ROOT_DIR) ;\
        GO111MODULE=on go generate $$(go list ./pkg/...) || exit 1 ;\
        $(MAKE) vendor  ;\
    fi;
.PHONY: go-generate

go-update-all: ## view available minor and patch upgrades for all direct and indirect
	if [ "${DIND}" = "1" ]; then \
		$(call go_docker,"make go-update-all") ;\
    else \
        cd $(ROOT_DIR) ;\
    	GO111MODULE=on go list -u -m all ;\
    fi;
.PHONY: go-update-all

lint: ## execute linter
	if [ "${DIND}" = "1" ]; then \
		$(call go_docker,"make lint") ;\
    else \
        golangci-lint run --no-config --issues-exit-code=0 --deadline=30m \
          --disable-all --enable=deadcode  --enable=gocyclo --enable=golint --enable=varcheck \
          --enable=structcheck --enable=maligned --enable=errcheck --enable=dupl --enable=ineffassign \
          --enable=interfacer --enable=unconvert --enable=goconst --enable=gosec --enable=megacheck ./... ;\
    fi;
.PHONY: lint

test-with-coverage: ## test application with race and total coverage
	if [ "${DIND}" = "1" ]; then \
		$(call go_docker,"make test-with-coverage") ;\
    else \
        GO111MODULE=on CGO_ENABLED=1 \
        go test -mod vendor -v -race -covermode atomic -coverprofile profile.out ./pkg/... || exit 1 ;\
        go tool cover -func=profile.out && rm -rf profile.out ;\
    fi;
.PHONY: test-with-coverage

test: ## test application with race
	if [ "${DIND}" = "1" ]; then \
		$(call go_docker,"make test") ;\
    else \
        GO111MODULE=on CGO_ENABLED=1 \
        go test -mod vendor  -race -v ./pkg/... ;\
    fi;
.PHONY: test

vendor: ## update vendor dependencies
	if [ "${DIND}" = "1" ]; then \
		$(call go_docker,"make vendor") ;\
    else \
        rm -rf $(ROOT_DIR)/vendor ;\
    	GO111MODULE=on \
    	go mod vendor ;\
    fi;
.PHONY: vendor

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help

init:
	mkdir -p generated artifacts ;\
    rm -rf artifacts/*
.PHONY: init

.DEFAULT_GOAL := help
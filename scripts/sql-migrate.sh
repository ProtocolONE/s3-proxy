#!/usr/bin/env sh

if [ -n "$1" ] && [ ${0:0:4} = "/bin" ]; then
  ROOT_DIR=$1
else
  ROOT_DIR="$( cd "$( dirname "$0" )" && pwd )/.."
fi

. $ROOT_DIR/scripts/common.sh

docker run --rm \
		-v /${ROOT_DIR}:/${ROOT_DIR} \
		-v /${GO_PATH}/pkg/mod:/${GO_PATH}/pkg/mod \
		-w /${ROOT_DIR} \
		-e GOPATH=/${GO_PATH}:/go \
		--network="${DOCKER_NETWORK}" \
		${GO_IMAGE}:${GO_IMAGE_TAG} \
		sh -c "sql-migrate $*"
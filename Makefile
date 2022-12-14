.DEFAULT_GOAL:=help

#============================================================================

# Load environment variables for local development
include .env
export

K6BIN := $(if $(shell command -v k6 2> /dev/null),k6,$(shell mktemp -d)/k6)

#============================================================================

.PHONY: dev
dev:							## Run dev container
	@docker inspect --type container ${SERVICE_NAME} >/dev/null 2>&1 && echo "A container named ${SERVICE_NAME} is already running." || \
	echo "Run dev container ${SERVICE_NAME}. To stop it, run \"make stop\"." && \
	docker run -d --rm \
	-v $(PWD):/${SERVICE_NAME} \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-p ${SERVICE_PORT}:${SERVICE_PORT} \
	--network instill-network \
	--name ${SERVICE_NAME} \
	instill/${SERVICE_NAME}:dev >/dev/null 2>&1

.PHONY: logs
logs:							## Tail container logs with -n 10
	@docker logs ${SERVICE_NAME} --follow --tail=10

.PHONY: stop
stop:							## Stop container
	@docker stop -t 1 ${SERVICE_NAME}

.PHONY: rm
rm:							## Remove container
	@docker rm -f ${SERVICE_NAME}

.PHONY: top
top:							## Display all running service processes
	@docker top ${SERVICE_NAME}

.PHONY: build
build:							## Build dev docker image
	docker build --build-arg SERVICE_NAME=${SERVICE_NAME} -f Dockerfile.dev  -t instill/${SERVICE_NAME}:dev .

.PHONY: go-gen
go-gen:       					## Generate codes
	go generate ./...

.PHONY: unit-test
unit-test:       				## Run unit test
	@go test -v -race -coverpkg=./... -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out
	@rm coverage.out

.PHONY: integration-test
integration-test:				## Run integration test
	@if [ ${K6BIN} != "k6" ]; then\
		echo "Install k6 binary at ${K6BIN}";\
		go version;\
		go install go.k6.io/xk6/cmd/xk6@latest;\
		xk6 build --with github.com/szkiba/xk6-jose@latest --output ${K6BIN};\
	fi
	@TEST_FOLDER_ABS_PATH=${PWD} ${K6BIN} run -e HOST=$(HOST) integration-test/rest.js --no-usage-report
	@if [ ${K6BIN} != "k6" ]; then rm -rf $(dirname ${K6BIN}); fi

.PHONY: help
help:       	 				## Show this help
	@echo "\nMakefile for locel development"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m (default: help)\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL:=help

#============================================================================

# Load environment variables for local development
include .env
export

GOTEST_FLAGS := CFG_DATABASE_HOST=${TEST_DBHOST} CFG_DATABASE_NAME=${TEST_DBNAME}
ifeq (${DBTEST}, true)
	GOTEST_TAGS := -tags=dbtest
endif

#============================================================================

.PHONY: dev
dev:							## Run dev container
	@docker compose ls -q | grep -q "instill-core" && true || \
		(echo "Error: Run \"make latest PROFILE=pipeline\" in vdp repository (https://github.com/instill-ai/instill-core) in your local machine first." && exit 1)
	@docker inspect --type container ${SERVICE_NAME} >/dev/null 2>&1 && echo "A container named ${SERVICE_NAME} is already running." || \
		echo "Run dev container ${SERVICE_NAME}. To stop it, run \"make stop\"."
	@docker run -d --rm \
		-v $(PWD):/${SERVICE_NAME} \
		-p ${PUBLIC_SERVICE_PORT}:${PUBLIC_SERVICE_PORT} \
		-p ${PRIVATE_SERVICE_PORT}:${PRIVATE_SERVICE_PORT} \
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
rm:								## Remove container
	@docker rm -f ${SERVICE_NAME}

.PHONY: top
top:							## Display all running service processes
	@docker top ${SERVICE_NAME}

.PHONY: build
build:							## Build dev docker image
	@docker build \
		--build-arg SERVICE_NAME=${SERVICE_NAME} \
		--build-arg GOLANG_VERSION=${GOLANG_VERSION} \
		--build-arg K6_VERSION=${K6_VERSION} \
		--build-arg XK6_VERSION=${XK6_VERSION} \
		-f Dockerfile.dev  -t instill/${SERVICE_NAME}:dev .

.PHONY: run-dev-services
run-dev-services:							## Run test container with image built by Dockerfile
	@docker compose ls -q | grep -q "instill-core" && true || \
		(echo "Error: Run \"make latest PROFILE=pipeline\" in vdp repository (https://github.com/instill-ai/instill-core) in your local machine first." && exit 1)
	@docker inspect --type container ${SERVICE_NAME} >/dev/null 2>&1 && echo "A container named ${SERVICE_NAME} is already running." || \
		echo "Run dev container ${SERVICE_NAME}. To stop it, run \"make stop\"."
	@docker run --network=instill-network \
		--name ${SERVICE_NAME} \
		-d ${SERVICE_NAME}:latest ./${SERVICE_NAME}
	@docker run --network=instill-network \
		--name pipeline-backend-worker \
		-d ${SERVICE_NAME}:latest ./${SERVICE_NAME}-worker

.PHONY: rm-test-container
rm-test-container:
	@docker rm -f ${SERVICE_NAME} ${SERVICE_NAME}-worker

.PHONY: build-dev-image
build-dev-image:							## Build test docker image with Dockerfile
	@docker buildx build \
		--build-arg GOLANG_VERSION=${GOLANG_VERSION} \
		--build-arg SERVICE_NAME=${SERVICE_NAME} \
		-t pipeline-backend:latest .

.PHONY: go-gen
go-gen:       					## Generate codes
	go generate ./...

.PHONY: dbtest-pre
dbtest-pre:
	@${GOTEST_FLAGS} go run ./cmd/migration

.PHONY: coverage
coverage:
	@if [ "${DBTEST}" = "true" ]; then  make dbtest-pre; fi
	@${GOTEST_FLAGS} go test -v -race ${GOTEST_TAGS} -coverpkg=./... -coverprofile=coverage.out -covermode=atomic ./...
	@if [ "${HTML}" = "true" ]; then  \
		go tool cover -func=coverage.out && \
		go tool cover -html=coverage.out && \
		rm coverage.out; \
	fi

.PHONY: test
test:
	@if [ "${OCR}" = "true" ]; then \
		make test-ocr; \
	else \
		go test -v ./... -json | tparse --notests --all; \
	fi

.PHONY: test-ocr
test-ocr:
# Certain component tests require additional dependencies.
# Install tesseract via `brew install tesseract`
# Setup `export LIBRARY_PATH="/opt/homebrew/lib"` `export CPATH="/opt/homebrew/include"`
ifeq ($(shell uname), Darwin)
	@TESSDATA_PREFIX=$(shell dirname $(shell brew list tesseract | grep share/tessdata/eng.traineddata)) go test -v ./... -tags ocr -json | tparse --notests --all
else
	@echo "This target can only be executed on Darwin (macOS)."
endif

.PHONY: integration-test
integration-test:				## Run integration test
	@ # DB_HOST points to localhost by default. Override this variable if
	@ # pipeline-backend's database isn't accessible at that host.
	@TEST_FOLDER_ABS_PATH=${PWD} k6 run \
		-e API_GATEWAY_PROTOCOL=${API_GATEWAY_PROTOCOL} \
		-e API_GATEWAY_URL=${API_GATEWAY_URL} \
		-e DB_HOST=${DB_HOST} \
		integration-test/pipeline/grpc.js --no-usage-report --quiet
	@TEST_FOLDER_ABS_PATH=${PWD} k6 run \
		-e API_GATEWAY_PROTOCOL=${API_GATEWAY_PROTOCOL} \
		-e API_GATEWAY_URL=${API_GATEWAY_URL} \
		-e DB_HOST=${DB_HOST} \
		integration-test/pipeline/rest.js --no-usage-report --quiet

.PHONY: gen-mock
gen-mock:
	@go install github.com/gojuno/minimock/v3/cmd/minimock@v3.4.0
	@go generate -run minimock ./...

.PHONY: help
help:       	 				## Show this help
	@echo "\nMakefile for local development"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m (default: help)\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

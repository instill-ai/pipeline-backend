.DEFAULT_GOAL:=help

#============================================================================

# Load environment variables for local development
include .env
export

GOTEST_FLAGS := CFG_DATABASE_HOST=${TEST_DBHOST} CFG_DATABASE_NAME=${TEST_DBNAME}

#============================================================================

.PHONY: dev
dev: ## Run dev container
	@docker compose ls -q | grep -q "instill-core" && true || \
		(echo "Error: Run \"make latest PROFILE=exclude-pipeline\" in vdp repository (https://github.com/instill-ai/instill-core) in your local machine first." && exit 1)
	@docker inspect --type container ${SERVICE_NAME} >/dev/null 2>&1 && echo "A container named ${SERVICE_NAME} is already running." || \
		echo "Run dev container ${SERVICE_NAME}. To stop it, run \"make stop\"."
	@docker run -d --rm \
		-v $(PWD):/${SERVICE_NAME} \
		-p ${PUBLIC_SERVICE_PORT}:${PUBLIC_SERVICE_PORT} \
		-p ${PRIVATE_SERVICE_PORT}:${PRIVATE_SERVICE_PORT} \
		--env-file .env.component \
		--network instill-network \
		--name ${SERVICE_NAME} \
		instill/${SERVICE_NAME}:dev >/dev/null 2>&1
.PHONY: latest
latest: ## Run latest container
	@docker compose ls -q | grep -q "instill-core" && true || \
		(echo "Error: Run \"make latest PROFILE=exclude-pipeline\" in vdp repository (https://github.com/instill-ai/instill-core) in your local machine first." && exit 1)
	@docker inspect --type container ${SERVICE_NAME} >/dev/null 2>&1 && echo "A container named ${SERVICE_NAME} is already running." || \
		echo "Run latest container ${SERVICE_NAME} and ${SERVICE_NAME}-worker. To stop it, run \"make stop\"."
	@docker run --network=instill-network \
		--name ${SERVICE_NAME} \
		-d instill/${SERVICE_NAME}:latest ./${SERVICE_NAME}
	@docker run --network=instill-network \
		--name ${SERVICE_NAME}-worker \
		-d instill/${SERVICE_NAME}:latest ./${SERVICE_NAME}-worker

.PHONY: rm
rm: ## Remove all running containers
	@docker rm -f ${SERVICE_NAME} ${SERVICE_NAME}-worker >/dev/null 2>&1

.PHONY: build-dev
build-dev: ## Build dev docker image
	@docker build \
		--build-arg SERVICE_NAME=${SERVICE_NAME} \
		--build-arg GOLANG_VERSION=${GOLANG_VERSION} \
		--build-arg K6_VERSION=${K6_VERSION} \
		--build-arg XK6_VERSION=${XK6_VERSION} \
		--build-arg XK6_SQL_VERSION=${XK6_SQL_VERSION} \
		--build-arg XK6_SQL_POSTGRES_VERSION=${XK6_SQL_POSTGRES_VERSION} \
		-f Dockerfile.dev -t instill/${SERVICE_NAME}:dev .

.PHONY: build-latest
build-latest: ## Build latest docker image
	@docker build \
		--build-arg GOLANG_VERSION=${GOLANG_VERSION} \
		--build-arg SERVICE_NAME=${SERVICE_NAME} \
		-t instill/pipeline-backend:latest .

.PHONY: go-gen
go-gen: ## Generate codes
	go generate ./...

.PHONY: dbtest-pre
dbtest-pre:
	@${GOTEST_FLAGS} go run ./cmd/migration

.PHONY: coverage
coverage: ## Generate coverage report
	@if [ "${DBTEST}" = "true" ]; then  make dbtest-pre; fi
	@docker run --rm \
		-v $(PWD):/${SERVICE_NAME} \
		-e GOTEST_FLAGS="${GOTEST_FLAGS}" \
		--user $(id -u):$(id -g) \
		--entrypoint= \
		instill/${SERVICE_NAME}:dev \
			go test -v -race ${GOTEST_TAGS} -coverpkg=./... -coverprofile=coverage.out -covermode=atomic -timeout 30m ./...
	@if [ "${HTML}" = "true" ]; then  \
		docker run --rm \
			-v $(PWD):/${SERVICE_NAME} \
			--user $(id -u):$(id -g) \
			--entrypoint= \
			instill/${SERVICE_NAME}:dev \
				go tool cover -func=coverage.out && \
				go tool cover -html=coverage.out && \
				rm coverage.out; \
	fi

# Tests should run in container without local tparse installation.
# If you encounter container test issues, install tparse locally:
# go install github.com/mfridman/tparse/cmd/tparse@latest
.PHONY: test
test: ## Run unit test
	@TAGS=""; \
	if [ "$${OCR}" = "true" ]; then \
		TAGS="$$TAGS,ocr"; \
		[ "$$(uname)" = "Darwin" ] && export TESSDATA_PREFIX=$$(dirname $$(brew list tesseract | grep share/tessdata/eng.traineddata)); \
	fi; \
	if [ "$${ONNX}" = "true" ]; then \
		if [ "$$(uname)" = "Darwin" ]; then \
			echo "ONNX Runtime test is not supported on Darwin (macOS)."; \
		else \
			TAGS="$$TAGS,onnx"; \
		fi; \
	fi; \
	TAGS=$${TAGS#,}; \
	if [ -n "$$TAGS" ]; then \
		echo "Running tests with tags: $$TAGS"; \
		go test -v -tags="$$TAGS" ./... -json | tparse --notests --all; \
	else \
		echo "Running standard tests"; \
		go test -v ./... -json | tparse --notests --all; \
	fi

.PHONY: integration-test
integration-test: ## Run integration test
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
gen-mock: ## Generate mock files
	@go install github.com/gojuno/minimock/v3/cmd/minimock@v3.4.0
	@go generate -run minimock ./...

.PHONY: gen-component-doc
gen-component-doc: ## Generate component docs
	@rm -f $$(find ./pkg/component -name README.mdx | paste -d ' ' -s -)
	@cd ./pkg/component/tools/compogen && go install .
	@go generate -run compogen ./pkg/component/...

.PHONY: help
help: ## Show this help
	@echo "\nMakefile for local development"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m (default: help)\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

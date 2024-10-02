COMPONENT_BASE := ../../

#============================================================================
# Generate the documentation of one of more components with the local version
# of `compogen`.
# - `t`: component type
# - `c`: component name
# Example: make local-gen-doc t=application c=slack
local-gen-doc:
	@go install .
	@cd ${COMPONENT_BASE} && go generate -run compogen $(shell echo ./$t/$c/... | tr -s /)

test:
	go test -mod=mod -race -cover -json ./... | tparse --all --notests

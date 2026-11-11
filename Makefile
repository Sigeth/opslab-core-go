# Tools are pinned in tools/go.mod so they do not leak into this library's
# dependencies. They are built outside any go.work (-modfile is not allowed in
# workspace mode) but run inside it.
CONTROLLER_GEN := bin/controller-gen
GENERATED := api crds schema

.PHONY: all generate test lint verify-generated

all: generate lint test

$(CONTROLLER_GEN): tools/go.mod
	GOWORK=off go build -modfile=tools/go.mod -o $@ sigs.k8s.io/controller-tools/cmd/controller-gen

## generate: deepcopy functions, CRD manifests and the challenge package JSON Schema
generate: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object paths=./api/...
	$(CONTROLLER_GEN) crd paths=./api/... output:crd:artifacts:config=crds
	go run ./cmd/schemagen -o schema

test:
	go test -race ./...

lint:
	go vet ./...
	@unformatted=$$(gofmt -l .); if [ -n "$$unformatted" ]; then echo "$$unformatted"; exit 1; fi

## verify-generated: fail if generated files are out of date (used by CI)
verify-generated: generate
	@changes=$$(git status --porcelain -- $(GENERATED)); if [ -n "$$changes" ]; then echo "generated files are out of date, run 'make generate':"; echo "$$changes"; exit 1; fi

build_tags := "sqlite_json,sqlite_fts5"

build:
	go build -tags $(build_tags)

test:
	go vet ./...
	go test -tags $(build_tags) ./...

build-ui:
	cd ui/ && npm run build

.PHONY: build test build-ui

SHA1_LOCK_FILE := $(abspath $(CURDIR)/../../protogen/buf.sha1.lock)

# bingo creates symlinks from the -l option in GOBIN, from where
# we can easily use it with buf. To have the symlinks inside this
# repo and on a known location, we set GOBIN to .bingo in the root
# of the repository (therefore we need to cd ../..)
.PHONY: protoc-deps
protoc-deps: $(BINGO)
	@cd ../.. && GOPATH="" GOBIN=".bingo" $(BINGO) get -l google.golang.org/protobuf/cmd/protoc-gen-go
	@cd ../.. && GOPATH="" GOBIN=".bingo" $(BINGO) get -l github.com/go-micro/generator/cmd/protoc-gen-micro
	@cd ../.. && GOPATH="" GOBIN=".bingo" $(BINGO) get -l github.com/owncloud/protoc-gen-microweb
	@cd ../.. && GOPATH="" GOBIN=".bingo" $(BINGO) get -l github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2
	@cd ../.. && GOPATH="" GOBIN=".bingo" $(BINGO) get -l github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc
	@cd ../.. && GOPATH="" GOBIN=".bingo" $(BINGO) get -l github.com/favadi/protoc-go-inject-tag

.PHONY: buf-generate
buf-generate: $(SHA1_LOCK_FILE)
	@find $(abspath $(CURDIR)/../../protogen/proto/) -type f -print0 | sort -z | xargs -0 sha1sum > buf.sha1.lock.tmp
	@cmp $(SHA1_LOCK_FILE) buf.sha1.lock.tmp --quiet || $(MAKE) -B $(SHA1_LOCK_FILE)
	@rm -f buf.sha1.lock.tmp

$(SHA1_LOCK_FILE): $(BUF) protoc-deps
	@echo "generating protobuf content"
	cd ../../protogen/proto && $(BUF) generate
	find $(abspath $(CURDIR)/../../protogen/proto/) -type f -print0 | sort -z | xargs -0 sha1sum > $(SHA1_LOCK_FILE)

# SylastraClaws Makefile
# 日常开发用 go build/go test 即可（不带 matrix tag，不影响）。
# 全量编译时用 make build-all，带上 Matrix 通道（纯 Go 加密，无需 CGo）。

GOFLAGS_ALL = -tags="matrix goolm"
BINARY      = sylastraclaws

.PHONY: build build-all test-all clean

build:
	go build ./...

build-all:
	go build $(GOFLAGS_ALL) -o $(BINARY) .

test-all:
	go test $(GOFLAGS_ALL) ./...

vet-all:
	go vet $(GOFLAGS_ALL) ./...

clean:
	rm -f $(BINARY)

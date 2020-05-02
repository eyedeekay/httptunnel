
GO_COMPILER_OPTS = -a -tags netgo -ldflags '-w -extldflags "-static"'

bhttpall: bfmt bwin blin blinarm bmac

bfmt:
	find . -name '*.go' -exec gofmt -w {} \;


all: bhttpall

bwin: bwin32 bwin64

bwin64:
	#GOOS=windows GOARCH=amd64 go build \
	#	$(GO_COMPILER_OPTS) \
	#	-buildmode=exe \
	#	-o ./httpproxy.exe \
	#	./windows/main.go
	#@echo "built"

bwin32:
	#GOOS=windows GOARCH=386 go build \
	#	$(GO_COMPILER_OPTS) \
	#	-buildmode=exe \
	#	-o ./httpproxy.exe \
	#	./windows/main.go
	#@echo "built"

blin: blin64 blin32

blin64:
	GOOS=linux GOARCH=amd64 go build \
		$(GO_COMPILER_OPTS) \
		-o ./browserproxy-64 \
		./multiproxy/browserproxy/main.go
	@echo "built"

blin32:
	GOOS=linux GOARCH=386 go build \
		$(GO_COMPILER_OPTS) \
		-buildmode=exe \
		-o ./browserproxy-32 \
		./multiproxy/browserproxy/main.go
	@echo "built"

blinarm: blinarm32 blinarm64

blinarm64:
	GOOS=linux GOARCH=arm64 go build \
		$(GO_COMPILER_OPTS) \
		-buildmode=exe \
		-o ./browserproxy-arm64 \
		./multiproxy/browserproxy/main.go
	@echo "built"

blinarm32:
	GOOS=linux GOARCH=arm go build \
		$(GO_COMPILER_OPTS) \
		-buildmode=exe \
		-o ./browserproxy-arm32 \
		./multiproxy/browserproxy/main.go
	@echo "built"


bmac: bmac32 bmac64

bmac64:
	GOOS=darwin GOARCH=amd64 go build \
		$(GO_COMPILER_OPTS) \
		-o ./browserproxy-64.app \
		./multiproxy/browserproxy/main.go
	@echo "built"

bmac32:
	GOOS=darwin GOARCH=amd64 go build \
		$(GO_COMPILER_OPTS) \
		-o ./browserproxy-32.app \
		./multiproxy/browserproxy/main.go
	@echo "built"

bvet:
	go vet ./*.go
	go vet ./multiproxy/browserproxy/*.go
	go vet ./windows/browserproxy*.go

bclean:
	rm -f browserproxy*

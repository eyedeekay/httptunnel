
GO_COMPILER_OPTS = -a -tags netgo -ldflags '-w -extldflags "-static"'

all: fmt win lin linarm mac

fmt:
	gofmt -w *.go */*.go

dep:
	go get -u github.com/eyedeekay/httptunnel/httpproxy

win: win32 win64

win64:
	GOOS=windows GOARCH=amd64 go build \
		$(GO_COMPILER_OPTS) \
		-buildmode=exe \
		-o ./httpproxy.exe \
		./windows/main.go
	@echo "built"

win32:
	GOOS=windows GOARCH=386 go build \
		$(GO_COMPILER_OPTS) \
		-buildmode=exe \
		-o ./httpproxy.exe \
		./windows/main.go
	@echo "built"

lin: lin64 lin32

lin64:
	GOOS=linux GOARCH=amd64 go build \
		$(GO_COMPILER_OPTS) \
		-o ./httpproxy-64 \
		./httpproxy/main.go
	@echo "built"

lin32:
	GOOS=linux GOARCH=386 go build \
		$(GO_COMPILER_OPTS) \
		-buildmode=exe \
		-o ./httpproxy-32 \
		./httpproxy/main.go
	@echo "built"

linarm: linarm32 linarm64

linarm64:
	GOOS=linux GOARCH=arm64 go build \
		$(GO_COMPILER_OPTS) \
		-buildmode=exe \
		-o ./httpproxy-arm64 \
		./httpproxy/main.go
	@echo "built"

linarm32:
	GOOS=linux GOARCH=arm go build \
		$(GO_COMPILER_OPTS) \
		-buildmode=exe \
		-o ./httpproxy-arm32 \
		./httpproxy/main.go
	@echo "built"


mac: mac32 mac64

mac64:
	GOOS=darwin GOARCH=amd64 go build \
		$(GO_COMPILER_OPTS) \
		-o ./httpproxy-64.app \
		./httpproxy/main.go
	@echo "built"

mac32:
	GOOS=darwin GOARCH=amd64 go build \
		$(GO_COMPILER_OPTS) \
		-o ./httpproxy-32.app \
		./httpproxy/main.go
	@echo "built"

vet:
	go vet ./*.go
	go vet ./httpproxy/*.go
	go vet ./windows/*.go

clean:
	rm -f httpproxy-* *.exe *.log

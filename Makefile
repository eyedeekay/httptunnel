
GO_COMPILER_OPTS = -a -tags netgo -ldflags '-w -extldflags "-static"'

all: fmt win lin linarm mac

fmt:
	gofmt -w *.go */*.go

dep:
	go get -u github.com/eyedeekay/httptunnel/httpproxy
	go get -u github.com/gopherjs/gopherjs

win: win32 win64

win64:
	GOOS=windows GOARCH=amd64 go build \
		$(GO_COMPILER_OPTS) \
		-buildmode=exe \
		-o ./httproxy.exe \
		./windows/main.go

win32:
	GOOS=windows GOARCH=386 go build \
		$(GO_COMPILER_OPTS) \
		-buildmode=exe \
		-o ./httpproxy.exe \
		./windows/main.go

lin: lin64 lin32

lin64:
	GOOS=linux GOARCH=amd64 go build \
		$(GO_COMPILER_OPTS) \
		-o ./httpproxy-64 \
		./httpproxy/main.go

lin32:
	GOOS=linux GOARCH=386 go build \
		$(GO_COMPILER_OPTS) \
		-buildmode=exe \
		-o ./httpproxy-32 \
		./httpproxy/main.go

linarm: linarm32 linarm64

linarm64:
	GOOS=linux GOARCH=arm64 go build \
		$(GO_COMPILER_OPTS) \
		-buildmode=exe \
		-o ./httpproxy-arm64 \
		./httpproxy/main.go

linarm32:
	GOOS=linux GOARCH=arm go build \
		$(GO_COMPILER_OPTS) \
		-buildmode=exe \
		-o ./httpproxy-arm32 \
		./httpproxy/main.go


mac: mac32 mac64

mac64:
	GOOS=darwin GOARCH=amd64 go build \
		$(GO_COMPILER_OPTS) \
		-o ./httpproxy-64.app \
		./httpproxy/main.go

mac32:
	GOOS=darwin GOARCH=amd64 go build \
		$(GO_COMPILER_OPTS) \
		-o ./httpproxy-32.app \
		./httpproxy/main.go

js:
	gopherjs build ./httpproxy

vet:
	go vet ./*.go
	go vet ./httpproxy/*.go
	go vet ./windows/*.go

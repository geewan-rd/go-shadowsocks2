NAME=shadowsocks2
BINDIR=output
GOBUILD=CGO_ENABLED=0 go build -ldflags '-w -s'
# The -w and -s flags reduce binary sizes by excluding unnecessary symbols and debug info

export GOSUMDB=off
export GOPROXY=https://goproxy.io,direct

BUILD_VERSION   := $(shell git describe --tags)
GIT_COMMIT_SHA1 := $(shell git rev-parse --short HEAD)
BUILD_TIME      := $(shell date "+%F %T")

all: linux macos win64

prebuild:
	go get golang.org/x/mobile/cmd/gomobile

linux:
	GOARCH=amd64 GOOS=linux $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

macos:
	GOARCH=amd64 GOOS=darwin $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

win64:
	GOARCH=amd64 GOOS=windows $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.exe

releases: linux macos win64
	chmod +x $(BINDIR)/$(NAME)-*
	gzip $(BINDIR)/$(NAME)-linux
	gzip $(BINDIR)/$(NAME)-macos
	zip -m -j $(BINDIR)/$(NAME)-win64.zip $(BINDIR)/$(NAME)-win64.exe

clean:
	rm $(BINDIR)/*

build-android:
	rm -rf output/android
	mkdir -p output/android
	gomobile bind -target android -o output/android/shadowsocks.aar github.com/shadowsocks/go-shadowsocks2/clientlib
	cd output && zip -r shadowsocks_android_${BUILD_VERSION}_${GIT_COMMIT_SHA1}.zip android
	# gomobile bind -target android -o output/android/tun2socks.aar github.com/geewan-rd/transocks-electron/accel/gotun2socks

build-ios:
	rm -rf output/ios
	mkdir -p output/ios
	gomobile bind -target ios,macos -o output/ios/shadowsocks.xcframework github.com/shadowsocks/go-shadowsocks2/clientlib
	cd output && zip -r shadowsocks_ios_${BUILD_VERSION}_${GIT_COMMIT_SHA1}.zip ios

build-socks5test:
	go build -o output/socks5_test github.com/shadowsocks/go-shadowsocks2/socks5test

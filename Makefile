NAME=TrojanGO
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
	# gomobile bind -target android -o output/android/trojan_go.framework github.com/p4gefau1t/trojan-go/clientlib

	gomobile bind -target android/arm64,android/arm -o output/android/trojan_go.aar github.com/p4gefau1t/trojan-go/clientlib

	cd output && zip -r trojan-go_android_${BUILD_VERSION}_${GIT_COMMIT_SHA1}.zip android
	
	# gomobile bind -target android -o output/android/tun2socks.aar github.com/geewan-rd/transocks-electron/accel/gotun2socks

build-ios:
	rm -rf output/ios
	mkdir -p output/ios
	export CGO_ENABLED=1
	export GOARCH=amd64
	# go build -buildmode=c-archive -o shadowsocks.a
	gomobile bind -target ios -o output/ios/trojan_go.framework github.com/p4gefau1t/trojan-go/clientlib
	cd output && zip -r shadowsocks_ios_${BUILD_VERSION}_${GIT_COMMIT_SHA1}.zip ios


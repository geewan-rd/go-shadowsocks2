module github.com/shadowsocks/go-shadowsocks2

go 1.12

require (
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da
	github.com/fregie/mpx v0.2.0
	github.com/gorilla/websocket v1.4.2
	github.com/juju/ratelimit v1.0.1
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	golang.org/x/crypto v0.0.0-20200128174031-69ecbb4d6d5d
	golang.org/x/sys v0.0.0-20191026070338-33540a1f6037 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
)

replace (
	github.com/fregie/mpx => ../mpx
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2 => github.com/golang/crypto v0.0.0-20190308221718-c2843e01d9a2
	golang.org/x/crypto v0.0.0-20190426145343-a29dc8fdc734 => github.com/golang/crypto v0.0.0-20190426145343-a29dc8fdc734
	golang.org/x/net v0.0.0-20190404232315-eb5bcb51f2a3 => github.com/golang/net v0.0.0-20190404232315-eb5bcb51f2a3
	golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a => github.com/golang/sys v0.0.0-20190215142949-d0b11bdaac8a
	golang.org/x/sys v0.0.0-20190412213103-97732733099d => github.com/golang/sys v0.0.0-20190412213103-97732733099d
	golang.org/x/text v0.3.0 => github.com/golang/text v0.3.0
)

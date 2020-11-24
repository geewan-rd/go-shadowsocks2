package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"sync"
)

var (
	times    = flag.Int("t", 100, "number of times")
	parallel = flag.Int("p", 1, "number of parallel")
)

func main() {
	flag.Parse()
	proxy := func(_ *http.Request) (*url.URL, error) {
		return url.Parse("socks5://127.0.0.1:1080")
	}
	transport := &http.Transport{Proxy: proxy}
	client := &http.Client{Transport: transport}
	for i := 0; i < *times; i++ {
		log.Printf("%d x %d", i, *parallel)
		wg := sync.WaitGroup{}
		doReq := func() {
			_, err := client.Get("https://www.google.com")
			if err != nil {
				log.Printf("failed: %s", err)
				return
			}
			wg.Done()
		}
		for j := 0; j < *parallel; j++ {
			wg.Add(1)
			go doReq()
		}
		wg.Wait()
	}
}

package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var (
	parallel = flag.Int("p", 10, "parallel of connections")
	socks5   = flag.String("socks5", "127.0.0.1:1080", "Socks5 proxy")
	getURL   = flag.String("url", "https://abs.fobwifi.com/api/1/ip", "url to test")
)

func main() {
	flag.Parse()
	proxy := func(_ *http.Request) (*url.URL, error) {
		return url.Parse("socks5://" + *socks5)
	}
	transport := &http.Transport{Proxy: proxy}
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	wg := sync.WaitGroup{}
	for i := 0; i < *parallel; i++ {
		wg.Add(1)
		go func(no int) {
			res, err := client.Get(*getURL)
			if err != nil {
				log.Print(err)
				return
			}
			defer res.Body.Close()
			ioutil.ReadAll(res.Body)
			log.Printf("[%d] finish!", no)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

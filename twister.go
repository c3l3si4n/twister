package main

import (
	"atomicgo.dev/robin"
	"bufio"
	"fmt"
	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/auth"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"sync"
	"time"
)

var proxyFileContents []string

// map with proxy servers
var hostnameToLoadBalancerMap map[string]*robin.Loadbalancer[string]
var hostnameToLoadBalancerMapLock sync.Mutex

func main() {
	if len(os.Args) < 4 {
		log.Fatal("Usage: twister <proxylist file> <proxy username> <proxy password>")
	}
	rand.Seed(time.Now().UnixNano())
	readLoadBalancer(os.Args[1])
	hostnameToLoadBalancerMap = make(map[string]*robin.Loadbalancer[string])

	proxy := goproxy.NewProxyHttpServer()
	proxy.Tr.Proxy = func(req *http.Request) (*url.URL, error) {
		url, err := url.Parse(getProxyForHostname(req))
		if err != nil {
			log.Fatal(err)
		}
		return url, err
	}
	auth.ProxyBasic(proxy, "twister", func(user, passwd string) bool {
		return user == os.Args[2] && passwd == os.Args[3]
	})
	proxy.ConnectDial = proxy.CustomHTTPDialer(getProxyForHostname, SetPassword)

	fmt.Println("Started Twister on http://0.0.0.0:58081")

	err := http.ListenAndServe("0.0.0.0:58081", proxy)

	if err != nil {
		log.Fatal(err)
	}

}

func SetPassword(r *http.Request) {
	r.SetBasicAuth(os.Args[2], os.Args[3])
	r.Header["Proxy-Authorization"] = r.Header["Authorization"]
	r.Header["Authorization"] = nil
}

func shuffleSlice(slice []string) []string {
	for i := range slice {
		j := rand.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
	return slice
}

func readLoadBalancer(filename string) *robin.Loadbalancer[string] {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxies = append(proxies, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	lb := robin.NewLoadbalancer(shuffleSlice(proxies))

	proxyFileContents = proxies

	return lb
}

func reloadLoadBalancer(proxies []string) *robin.Loadbalancer[string] {

	lb := robin.NewLoadbalancer(proxies)
	return lb
}

func getProxyForHostname(req *http.Request) string {
	hostname := req.Host
	hostnameToLoadBalancerMapLock.Lock()
	loadBalancer, found := hostnameToLoadBalancerMap[req.Host]
	hostnameToLoadBalancerMapLock.Unlock()
	if !found {
		var newProxyServer *robin.Loadbalancer[string]
		newProxyServer = reloadLoadBalancer(proxyFileContents)
		hostnameToLoadBalancerMapLock.Lock()
		hostnameToLoadBalancerMap[hostname] = newProxyServer
		hostnameToLoadBalancerMapLock.Unlock()
		return newProxyServer.Next()
	} else {
		newProxy := loadBalancer.Next()
		return newProxy
	}
}

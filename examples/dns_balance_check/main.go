package main

import (
	"context"
	"fmt"
	"github.com/ethanvc/evo/evolog"
	"net"
	"net/http"
	"os"
)

func main() {
	host := os.Getenv("HOST")
	fmt.Printf("test host %s\n", host)
	if len(host) == 0 {
		host = "www.baidu.com"
	}
	go func() {
		err := http.ListenAndServe("127.0.0.1:9100", evolog.DefaultReporter().HttpHandler())
		if err != nil {
			panic(err)
		}
	}()
	c := evolog.WithLogContext(nil, &evolog.LogContextConfig{Method: "query_dns"})
	for {
		QueryDns(c, host)
	}
}

func QueryDns(c context.Context, host string) (err error) {
	var ip string
	defer func() {
		evolog.ReportEvent(c, ip)
	}()
	ips, err := net.LookupIP(host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get IPs: %v\n", err)
		return
	}
	if len(ips) == 0 {
		fmt.Fprintf(os.Stderr, "no ip found\n")
		return
	}
	ip = ips[0].String()
	return
}

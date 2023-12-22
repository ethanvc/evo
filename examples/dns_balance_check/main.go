package main

import (
	"context"
	"fmt"
	"github.com/ethanvc/evo/plog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net"
	"net/http"
	"os"
)

func main() {
	host := os.Getenv("HOST")
	fmt.Printf("test host %s, PreferGo is %t\n", host, net.DefaultResolver.PreferGo)
	if len(host) == 0 {
		host = "www.baidu.com"
	}
	go func() {
		err := http.ListenAndServe("127.0.0.1:9100", promhttp.Handler())
		if err != nil {
			panic(err)
		}
	}()
	c := plog.WithLogContext(nil, &plog.LogContextConfig{Method: "query_dns"})
	for {
		QueryDns(c, host)
	}
}

func QueryDns(c context.Context, host string) (err error) {
	var ip string
	defer func() {
		plog.ReportEvent(c, ip)
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

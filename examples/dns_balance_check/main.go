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
	go func() {
		err := http.ListenAndServe("127.0.0.1:9100", evolog.DefaultReporter().HttpHandler())
		if err != nil {
			panic(err)
		}
	}()
	c := evolog.WithLogContext(nil, &evolog.LogContextConfig{Method: "query_dns"})
	for {
		QueryDns(c)
	}
}

func QueryDns(c context.Context) (err error) {
	var ip string
	defer func() {
		evolog.ReportEvent(c, ip)
	}()
	ips, err := net.LookupIP("google.com")
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

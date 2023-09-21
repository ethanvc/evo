package main

import (
	"fmt"
	"github.com/ethanvc/evo/evolog"
	"net"
	"os"
	"time"
)

func main() {
	c := evolog.WithLogContext(nil, &evolog.LogContextConfig{Method: "query_dns"})
	for {
		ips, err := net.LookupIP("google.com")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not get IPs: %v\n", err)
			time.Sleep(time.Second)
		}
		if len(ips) == 0 {
			fmt.Fprintf(os.Stderr, "no ip found\n")
			time.Sleep(time.Second)
		}
		evolog.ReportServerRequest(c, "ResolveIP:"+ips[0].String())
	}
}

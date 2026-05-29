package httpproto

import "github.com/ethanvc/evo/httpcli"

func ConfigClient(cli *httpcli.Client) {
	cli.Serializer = &Serializer{}
}

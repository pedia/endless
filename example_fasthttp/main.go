package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/pedia/endless"
	"github.com/valyala/fasthttp"
)

func serve_http(addr string, ln net.Listener) (*fasthttp.Server, error) {
	server := &fasthttp.Server{Handler: func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString(fmt.Sprintf("%d\n", os.Getpid()))
	}}

	go server.Serve(ln)

	return server, nil
}

func main() {
	addr := ":3030"
	var s *fasthttp.Server

	endless.Start(
		func(p *endless.Parent) error {
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}

			p.AddListener(ln, addr)
			s, err = serve_http(addr, ln)
			return err
		}, func(c *endless.Child) error {
			nf, ok := c.NamedFiles[addr]
			fmt.Printf("got %#v\n", nf)
			if !ok {
				return fmt.Errorf("inherit %s not found", addr)
			}

			c.AddListener(nf.Listener, addr)
			server, err := serve_http(addr, nf.Listener)
			s = server
			return err
		},
		func(ctx context.Context) error {
			return s.Shutdown()
		},
	)
	fmt.Printf("Quit %d\n", os.Getpid())
}

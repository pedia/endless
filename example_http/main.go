package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/pedia/endless"
)

func serve_http(addr string, ln net.Listener) *http.Server {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%d\n", os.Getpid())
	})

	server := &http.Server{Addr: addr}
	go server.Serve(ln)

	return server
}

func main() {
	addr := ":3030"
	var s *http.Server

	endless.Start(
		func(p *endless.Parent) error {
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}

			p.AddListener(ln, addr)
			s = serve_http(addr, ln)
			return err
		}, func(c *endless.Child) error {
			nf, ok := c.NamedFiles[addr]
			fmt.Printf("got %#v\n", nf)
			if !ok {
				return fmt.Errorf("inherit %s not found", addr)
			}

			c.AddListener(nf.LN, addr)
			s = serve_http(addr, nf.LN)
			return nil
		},
		func(ctx context.Context) error {
			return s.Shutdown(ctx)
		},
	)
	fmt.Printf("Quit %d\n", os.Getpid())
}

Zero downtime restarts for Go Servers.

# Features
- Pass `init_parent`, `init_child`, `quit` callback for Server.
- `AddFile`, `AddListener` for multiple {file|socket} inherited.
- Handle signal HUP, INT, TERM.


# Test
Short with `make test`
```bash
# build http server
make

# startup parent
./e1 &

# fork new child
kill -HUP `curl -s http://127.0.0.1:3030`
```


# Demo
## Demo for http.Server
```go
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
```

## Demo for fasthttp.Server
```go
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

			c.AddListener(nf.LN, addr)
			server, err := serve_http(addr, nf.LN)
			s = server
			return err
		},
		func(ctx context.Context) error {
			return s.Shutdown()
		},
	)
	fmt.Printf("Quit %d\n", os.Getpid())
}
```
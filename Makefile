all: e1 e2

e1: example_http/main.go endless.go
	go build -o e1 example_http/main.go

e2: example_fasthttp/main.go endless.go
	go build -o e2 example_fasthttp/main.go

clean:
	rm -f e1 e2

test: e1
	./e1 &
	kill -HUP `curl -s http://127.0.0.1:3030`
	kill -HUP `curl -s http://127.0.0.1:3030`
	kill -HUP `curl -s http://127.0.0.1:3030`

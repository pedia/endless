package endless

import (
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func create_file(fn string) *os.File {
	f, _ := os.Create("/tmp/foo")
	return f
}

func create_listener(addr string) net.Listener {
	ln, _ := net.Listen("tcp", addr)
	return ln
}

func TestPair(t *testing.T) {
	is := assert.New(t)

	a := []int32{}
	a = append(a, []int32{}...)
	is.Equal(0, len(a))

	//
	p := new(Parent)

	p.AddFile(create_file("/tmp/foo"))
	p.AddListener(create_listener(":3030"), ":3030")
	is.Equal(2, len(p.Files))

	os.Remove("/tmp/foo")

	p.Quit()
}

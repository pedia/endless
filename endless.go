package endless

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type named_file struct {
	Name       string `json:"name"`
	file       *os.File
	FD         int          `json:"fd"`
	IsListener bool         `json:"l"`
	Addr       string       `json:"addr"`
	LN         net.Listener `json:"-"`
}

type Parent struct {
	Files []named_file
}

func (p *Parent) AddFile(f *os.File) {
	p.add(named_file{f.Name(), f, int(f.Fd()), false, "", nil})
}

func listener_to_file(ln net.Listener) (*os.File, error) {
	switch t := ln.(type) {
	case *net.TCPListener:
		return t.File()
	case *net.UnixListener:
		return t.File()
	}
	return nil, fmt.Errorf("unsupported listener: %T", ln)
}

func (p *Parent) AddListener(l net.Listener, addr string) {
	p.add(named_file{LN: l, Addr: addr, IsListener: true})
}

func (p *Parent) add(nfs ...named_file) {
	if p.Files == nil {
		p.Files = make([]named_file, 0, len(nfs))
	}
	p.Files = append(p.Files, nfs...)
}

func (p *Parent) ForkChild() (*os.Process, error) {
	// Get current process name and directory.
	exec_fp, err := os.Executable()
	if err != nil {
		return nil, err
	}

	// Current folder mabybe not same as folder of exec_fp
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	lnf, err := listener_to_file(p.Files[0].LN)
	if err != nil {
		return nil, err
	}
	defer lnf.Close()

	files := []*os.File{
		os.Stdin,
		os.Stdout,
		os.Stderr,
		lnf,
	}

	p.Files[0].FD = int(lnf.Fd())
	p.Files[0].Name = lnf.Name()

	// Get current environment and add `endless` to it.
	bs, err := json.Marshal(p.Files)
	if err != nil {
		return nil, err
	}
	environment := append(os.Environ(), "ENDLESS="+string(bs))

	args := []string{}
	if len(os.Args) > 1 {
		args = os.Args[1:]
	}

	// Spawn child process.
	process, err := os.StartProcess(
		exec_fp,
		args,
		&os.ProcAttr{
			Dir:   dir,
			Env:   environment,
			Files: files,
			Sys:   &syscall.SysProcAttr{},
		},
	)
	if err != nil {
		return nil, err
	}

	return process, nil
}

func (p *Parent) WaitForSignal(quit func(ctx context.Context) error) error {
	signalCh := make(chan os.Signal, 1024)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)
	for {
		s := <-signalCh
		fmt.Printf("receive signal.%v\n", s)

		switch s {
		case syscall.SIGHUP:
			proc, err := p.ForkChild()
			if err != nil {
				fmt.Printf("unable fork child: %s\n", err)
				continue
			}

			fmt.Printf("forked child: %d\n", proc.Pid)
			proc.Release() // must wait

		case syscall.SIGINT, syscall.SIGQUIT:
			// Create a context that will expire in 5 seconds and use this as a
			// timeout to Shutdown.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			err := quit(ctx)

			defer cancel()
			return err
		}
	}
}

func Start(
	init_parent func(p *Parent) error,
	init_child func(c *Child) error,
	quit func(ctx context.Context) error,
) {
	env := os.Getenv("ENDLESS")
	if env == "" {
		// it's parent, wait for SIGHUP
		p := new(Parent)
		if init_parent(p) != nil {
			os.Exit(1)
			return
		}

		err := p.WaitForSignal(quit)
		if err != nil {
			fmt.Printf("parent wait failed: %s\n", err)
		}
		return
	}

	c := NewClient(env)
	if c == nil {
		os.Exit(2)
		return
	}

	err := init_child(c)
	if err != nil {
		fmt.Printf("init child failed: %s\n", err)
		os.Exit(3)
		return
	}

	c.Ready()

	err = c.WaitForSignal(quit)
	if err != nil {
		fmt.Printf("quit failed: %s\n", err)
	}
}

type Child struct {
	*Parent
	NamedFiles map[string]named_file
}

func NewClient(env string) *Child {
	nfs := []named_file{}
	err := json.Unmarshal([]byte(env), &nfs)
	if err != nil {
		fmt.Printf("parse endless('%s') failed: %s\n", env, err)
		return nil
	}

	c := Child{&Parent{}, map[string]named_file{}}

	first_fd := 3

	for i, nf := range nfs {
		if nf.IsListener {
			file := os.NewFile(uintptr(first_fd+i), nf.Name)
			defer file.Close()
			nf.LN, err = net.FileListener(file)
			if err != nil {
				fmt.Printf("create listener inner failed: %s\n", err)
			}
			c.NamedFiles[nf.Addr] = nf
		} else {
			nf.file = os.NewFile(uintptr(first_fd+i), nf.Name)
			c.NamedFiles[nf.Name] = nf
		}
	}
	return &c
}

func (c *Child) Ready() {
	proc, err := os.FindProcess(os.Getppid())
	if err != nil {
		fmt.Printf("find parent failed %s\n", err)
		return
	}

	err = proc.Signal(os.Interrupt)
	if err != nil {
		fmt.Printf("signal int to parent failed %s\n", err)
		return
	}

	fmt.Print("signal to parent done\n")
}

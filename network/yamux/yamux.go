package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/yamux"
)

type logConn struct {
	net.Conn
}

func (l *logConn) Read(b []byte) (n int, err error) {
	n, err = l.Conn.Read(b)

	fmt.Printf("Read: %d; %v\n%s\n", n, err, hex.Dump(b[:n]))

	return n, err
}

func (l *logConn) Write(b []byte) (n int, err error) {
	n, err = l.Conn.Write(b)

	fmt.Printf("Write: %d; %v\n%s\n", n, err, hex.Dump(b[:n]))

	return n, err
}

func main() {
	srv, cli := net.Pipe()

	srv, cli = &logConn{srv}, &logConn{cli}

	config := yamux.DefaultConfig()
	server, err := yamux.Server(srv, config)
	if err != nil {
		panic(err)
	}

	client, err := yamux.Client(cli, config)
	if err != nil {
		panic(err)
	}

	// go handleClient(client, []byte{2})
	// go handleClient(client, []byte{1})
	go handleClient(server, []byte{42})

	// go handleServer(server, "s")
	// go handleServer(server, "s")
	go handleServer(client, "c")

	time.Sleep(3 * time.Second)
}

func handleServer(ses *yamux.Session, s string) {
	fmt.Println("a", s)
	stream, err := ses.AcceptStream()
	if err != nil {
		panic(err)
	}

	for {
		b := make([]byte, 256)
		n, err := stream.Read(b)
		if err != nil {
			panic(err)
		}

		fmt.Println("r", s, stream.StreamID(), b[:n])
	}
}

func handleClient(ses *yamux.Session, data []byte) {
	open, err := ses.OpenStream()
	if err != nil {
		panic(err)
	}

	for {
		fmt.Println("w", open.StreamID(), data)

		_, err := open.Write(data)
		if err != nil {
			panic(err)
		}
		time.Sleep(1 * time.Second)
	}
}

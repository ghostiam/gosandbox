package main

import (
	"encoding/hex"
	"fmt"
	"net"

	"github.com/cenkalti/rpc2"
	"github.com/cenkalti/rpc2/jsonrpc"
)

type logConn struct {
	Name string
	net.Conn
}

func (l *logConn) Read(b []byte) (n int, err error) {
	n, err = l.Conn.Read(b)

	fmt.Printf("[%s] Read: %d; %v\n%s\n", l.Name, n, err, hex.Dump(b[:n]))

	return n, err
}

func (l *logConn) Write(b []byte) (n int, err error) {
	n, err = l.Conn.Write(b)

	fmt.Printf("[%s] Write: %d; %v\n%s\n", l.Name, n, err, hex.Dump(b[:n]))

	return n, err
}

type HelloArgs struct {
	Name string
}

type HelloReply struct {
	Message string
}

func main() {
	srv, cli := net.Pipe()
	srv, cli = &logConn{"Server", srv}, &logConn{"Client", cli}

	// sRpc := rpc2.NewClient(srv)
	// cRpc := rpc2.NewClient(cli)
	sRpc := rpc2.NewClientWithCodec(jsonrpc.NewJSONCodec(srv))
	cRpc := rpc2.NewClientWithCodec(jsonrpc.NewJSONCodec(cli))

	sRpc.Handle("hello", func(_ *rpc2.Client, args HelloArgs, reply *HelloReply) error {
		*reply = HelloReply{
			Message: "Hello, " + args.Name,
		}
		return nil
	})

	cRpc.Handle("hello", func(_ *rpc2.Client, args HelloArgs, reply *HelloReply) error {
		*reply = HelloReply{
			Message: "Hello, " + args.Name,
		}
		return nil
	})

	go sRpc.Run()
	go cRpc.Run()

	var sReply HelloReply
	err := sRpc.Call("hello", HelloArgs{Name: "Server"}, &sReply)
	if err != nil {
		panic(err)
	}

	fmt.Println(sReply.Message)

	err = sRpc.Call("hello", HelloArgs{Name: "Server 2"}, &sReply)
	if err != nil {
		panic(err)
	}

	fmt.Println(sReply.Message)

	var cReply HelloReply
	err = cRpc.Call("hello", HelloArgs{Name: "Client"}, &cReply)
	if err != nil {
		panic(err)
	}

	fmt.Println(cReply.Message)

	err = cRpc.Call("hello", HelloArgs{Name: "Client 2"}, &cReply)
	if err != nil {
		panic(err)
	}

	fmt.Println(cReply.Message)
}

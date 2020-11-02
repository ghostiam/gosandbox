package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"

	stream "github.com/ghostiam/gosandbox/grpc/stream/proto"
)

func main() {
	addr := flag.String("addr", ":8901", "grpc chatServer address")
	flag.Parse()

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	stream.RegisterChatServiceServer(s, &chatServer{
		client: make(map[net.Addr]chan *stream.ChatMessage),
	})

	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type chatServer struct {
	clientMutex sync.Mutex
	client      map[net.Addr]chan *stream.ChatMessage
	//
	stream.UnimplementedChatServiceServer
}

func (s *chatServer) clientAdd(addr net.Addr) <-chan *stream.ChatMessage {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	msgCh := make(chan *stream.ChatMessage)
	s.client[addr] = msgCh
	return msgCh
}

func (s *chatServer) clientRemove(addr net.Addr) {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	close(s.client[addr])
	delete(s.client, addr)
}

func (s *chatServer) sendBroadcast(msg *stream.ChatMessage) {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	for _, c := range s.client {
		c <- msg
	}
}

func (s *chatServer) Chat(server stream.ChatService_ChatServer) error {
	ctx := server.Context()
	peerInfo, ok := peer.FromContext(ctx)
	if !ok {
		return errors.New("failed get peer info")
	}

	peerAddr := peerInfo.Addr
	fmt.Printf("connected: %s\n", peerAddr)

	clientCh := s.clientAdd(peerAddr)
	defer func() {
		fmt.Printf("disconnected: %s\n", peerAddr)
		s.clientRemove(peerAddr)
	}()

	errCh := make(chan error, 3)

	go func() {
		for {
			message, err := server.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					time.Sleep(100 * time.Millisecond)
					continue
				}

				errCh <- fmt.Errorf("server.Recv: %w", err)
				return
			}

			fmt.Printf("[%s]: %s\n", message.Username, message.Message)
			s.sendBroadcast(message)
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Printf("done: %v\n", ctx.Err())
				errCh <- ctx.Err()
				return

			case msg, ok := <-clientCh:
				if !ok {
					fmt.Printf("clientCh closed\n")
					errCh <- nil
					return
				}
				err := server.Send(msg)
				if err != nil {
					errCh <- fmt.Errorf("send from server to client: %w", err)
					return
				}
			}
		}
	}()

	return <-errCh
}

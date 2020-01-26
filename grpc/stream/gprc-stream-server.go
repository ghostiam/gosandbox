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

func (s *chatServer) Chat(chat stream.ChatService_ChatServer) error {
	peerInfo, ok := peer.FromContext(chat.Context())
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

	errCh := make(chan error, 2)

	go func() {
		for {
			message, err := chat.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					time.Sleep(100 * time.Millisecond)
					continue
				}

				errCh <- fmt.Errorf("chat.Recv: %w", err)
				return
			}

			fmt.Printf("[%s]: %s\n", message.Username, message.Message)
			s.sendBroadcast(message)
		}
	}()

	go func() {
		for {
			msg, ok := <-clientCh
			if !ok {
				errCh <- errors.New("clientCh closed")
				return
			}

			err := chat.Send(msg)
			if err != nil {
				errCh <- fmt.Errorf("chat.Send: %w", err)
				return
			}
		}
	}()

	return <-errCh
}

package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"

	stream "github.com/ghostiam/gosandbox/grpc/stream/proto"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8901", "grpc chatServer address")
	flag.Parse()

	conn, err := grpc.Dial(*addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := stream.NewChatServiceClient(conn)

	chat, err := client.Chat(context.Background())
	if err != nil {
		log.Fatalf("failed to connect to chat: %v", err)
	}
	defer chat.CloseSend()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("failed readline from console: %v", err)
	}
	username = strings.TrimSpace(username)

	err = chatHandler(chat, username)
	if err != nil {
		log.Fatalf("failed chatHandler: %v", err)
	}
}

func chatHandler(chat stream.ChatService_ChatClient, username string) error {
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
		}
	}()

	go func() {
		for {
			reader := bufio.NewReader(os.Stdin)
			message, err := reader.ReadString('\n')
			if err != nil {
				errCh <- fmt.Errorf("readline from console: %w", err)
				return
			}
			message = strings.TrimSpace(message)
			if message == "" {
				continue
			}

			err = chat.Send(&stream.ChatMessage{
				Username: username,
				Message:  message,
			})
			if err != nil {
				errCh <- fmt.Errorf("chat.Send: %w", err)
				return
			}
		}
	}()

	return <-errCh
}

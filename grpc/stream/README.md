# GRPC streaming chat

## Running

Start server:
```bash
go run gprc-stream-server.go
```

Start a few client(in other terminals):
```bash
go run grpc-stream-client.go
```

Set username and write message in terminal.

## Regenerate proto (if you changed the proto file)

```bash
make proto
```




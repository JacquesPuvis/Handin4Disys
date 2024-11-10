package main

import (
	"context"
	"fmt"
	pb "handin4/grpc"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
)

type server struct {
    pb.UnimplementedTokenRingServer
    hasToken   bool
    nextNode   string
}

/*
checks if HasToken in the TokenRequest message is true and "notifies" the internal server if this is the case. 
otherwise only returns TokenReply
*/
func (s *server) RequestToken(ctx context.Context, req *pb.TokenRequest) (*pb.TokenReply, error) {
    if req.HasToken {
        s.hasToken = true
        fmt.Printf("Node %s: Entering Critical Section\n", os.Getenv("NODE_ID"))
        time.Sleep(2 * time.Second)
        fmt.Printf("Node %s: Exiting Critical Section\n", os.Getenv("NODE_ID"))
        s.passToken()
    }
    return &pb.TokenReply{Message: "Token Received"}, nil
}

/*
forwards the token to the next node in the ring. It connects to the next node's gRPC server,
creates a client, and calls the RequestToken method to pass the token.
*/
func (s *server) passToken() {
    fmt.Printf("Node %s: Passing token to next node %s\n", os.Getenv("NODE_ID"), s.nextNode)

    conn, err := grpc.Dial(s.nextNode, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("Could not connect to next node: %v", err)
    }
    defer conn.Close()

    client := pb.NewTokenRingClient(conn)

    _, err = client.RequestToken(context.Background(), &pb.TokenRequest{HasToken: true})
    if err != nil {
        log.Fatalf("Could not pass token to next node: %v", err)
    }

    fmt.Printf("Node %s: Successfully passed token to Node %s\n", os.Getenv("NODE_ID"), s.nextNode)

    s.hasToken = false
}

/*
initializes the server node for the token ring. It loads environment variables to set the node’s address,
the next node's address, and whether this node starts with the token. then it listens on the specified
address and registers the gRPC TokenRingServer. If the node starts with the token, it calls RequestToken to 
enter the critical section and pass the token.
*/
func main() {
´    currentNode := os.Getenv("CURRENT_NODE")
    nextNode := os.Getenv("NEXT_NODE")
    hasToken := os.Getenv("HAS_TOKEN") == "true"

    lis, err := net.Listen("tcp", currentNode)
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    s := &server{
        hasToken: hasToken,
        nextNode: nextNode,
    }
    grpcServer := grpc.NewServer()
    pb.RegisterTokenRingServer(grpcServer, s)

    go func() {
        if err := grpcServer.Serve(lis); err != nil {
            log.Fatalf("Failed to serve: %v", err)
        }
    }()

´    if s.hasToken {
        fmt.Printf("Node %s: Starting with token\n", currentNode)
        s.RequestToken(context.Background(), &pb.TokenRequest{HasToken: true})
    }

    select {}
}

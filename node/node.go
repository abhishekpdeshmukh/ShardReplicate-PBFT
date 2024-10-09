package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	pb "ShardReplicate-PBFT/proto"

	"google.golang.org/grpc"
)

// Node structure
type Node struct {
	pb.UnimplementedPBFTServiceServer
	nodeID          string                 // Unique identifier for the node
	isPrimary       bool                   // Whether this node is the primary
	replicas        []string               // List of replica addresses in the shard
	requestLog      map[string]*pb.Request // Log of requests for replay protection
	accountBalances map[string]int         // Account balances for state management
	mu              sync.Mutex             // Mutex for synchronizing access to the log and balances
}

func GenerateDigest(request *pb.Request) string {
	hasher := sha256.New()
	hasher.Write([]byte(request.Operation + request.ClientId + fmt.Sprintf("%d", request.Timestamp)))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (n *Node) ClientRequest(ctx context.Context, request *pb.Request) (*pb.Response, error) {
	if !n.isPrimary {
		return nil, fmt.Errorf("this node is not the primary")
	}

	if request.ShardFrom != request.ShardTo {
		return nil, fmt.Errorf("cross-shard transactions are not supported in this setup")
	}

	digest := GenerateDigest(request)
	n.mu.Lock()
	n.requestLog[digest] = request
	n.mu.Unlock()

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.accountBalances[request.AccountFrom] < int(request.Amount) {
		return nil, fmt.Errorf("insufficient balance in %s", request.AccountFrom)
	}

	n.accountBalances[request.AccountFrom] -= int(request.Amount)
	n.accountBalances[request.AccountTo] += int(request.Amount)

	fmt.Printf("Transferred %d from %s to %s. Updated balances: %s=%d, %s=%d\n",
		request.Amount, request.AccountFrom, request.AccountTo,
		request.AccountFrom, n.accountBalances[request.AccountFrom],
		request.AccountTo, n.accountBalances[request.AccountTo])

	prePrepare := &pb.PrePrepare{
		View:           "1",
		SequenceNumber: time.Now().Unix(),
		Digest:         digest,
		Request:        request,
		ReplicaId:      n.nodeID,
		Signature:      []byte("signature_placeholder"),
	}

	ack, err := n.PrePrepareBroadcast(ctx, prePrepare)
	if err != nil {
		return nil, err
	}
	fmt.Println(ack)
	response := &pb.Response{
		ClientId:  request.ClientId,
		Timestamp: request.Timestamp,
		Result:    "Transaction completed and replicated",
	}
	return response, nil
}

func (n *Node) PrePrepareBroadcast(ctx context.Context, prePrepare *pb.PrePrepare) (*pb.Ack, error) {
	fmt.Printf("Primary node %s is broadcasting PrePrepare with digest: %s to replicas\n", n.nodeID, prePrepare.Digest)

	for _, replicaAddr := range n.replicas {
		fmt.Printf("Sending PrePrepare message to replica at %s\n", replicaAddr)
	}

	ack, err := n.PrepareBroadcast(ctx, &pb.Prepare{
		View:           prePrepare.View,
		SequenceNumber: prePrepare.SequenceNumber,
		ReplicaId:      n.nodeID,
		Digest:         prePrepare.Digest,
		Signature:      []byte("signature_placeholder"),
	})
	if err != nil {
		return nil, err
	}

	return ack, nil
}

func (n *Node) PrepareBroadcast(ctx context.Context, prepare *pb.Prepare) (*pb.Ack, error) {
	fmt.Printf("Primary node %s is broadcasting Prepare with digest: %s to replicas\n", n.nodeID, prepare.Digest)

	for _, replicaAddr := range n.replicas {
		fmt.Printf("Sending Prepare message to replica at %s\n", replicaAddr)
	}

	time.Sleep(1 * time.Second)

	ack, err := n.CommitBroadcast(ctx, &pb.Commit{
		View:           prepare.View,
		SequenceNumber: prepare.SequenceNumber,
		ReplicaId:      n.nodeID,
		Digest:         prepare.Digest,
		Signature:      []byte("signature_placeholder"),
	})
	if err != nil {
		return nil, err
	}

	fmt.Println(ack)
	return &pb.Ack{Success: true}, nil
}

func (n *Node) CommitBroadcast(ctx context.Context, commit *pb.Commit) (*pb.Ack, error) {
	fmt.Printf("Primary node %s is broadcasting Commit with digest: %s to replicas\n", n.nodeID, commit.Digest)

	for _, replicaAddr := range n.replicas {
		fmt.Printf("Sending Commit message to replica at %s\n", replicaAddr)
	}

	time.Sleep(500 * time.Millisecond)
	fmt.Printf("Primary node %s completed transaction commit.\n", n.nodeID)

	return &pb.Ack{Success: true}, nil
}

func (n *Node) PrePrepare(ctx context.Context, req *pb.PrePrepare) (*pb.Ack, error) {
	fmt.Printf("Replica node %s received PrePrepare message with digest: %s\n", n.nodeID, req.Digest)

	n.mu.Lock()
	n.requestLog[req.Digest] = req.Request
	n.mu.Unlock()

	return &pb.Ack{Success: true}, nil
}

func (n *Node) Prepare(ctx context.Context, req *pb.Prepare) (*pb.Ack, error) {
	fmt.Printf("Replica node %s received Prepare message with digest: %s\n", n.nodeID, req.Digest)

	n.mu.Lock()
	n.requestLog[req.Digest] = &pb.Request{}
	n.mu.Unlock()

	return &pb.Ack{Success: true}, nil
}

func (n *Node) Commit(ctx context.Context, req *pb.Commit) (*pb.Ack, error) {
	fmt.Printf("Replica node %s received Commit message with digest: %s\n", n.nodeID, req.Digest)

	n.mu.Lock()
	n.requestLog[req.Digest] = &pb.Request{}
	n.mu.Unlock()

	return &pb.Ack{Success: true}, nil
}

func startNode(nodeID string, isPrimary bool, address string, replicas []string) {
	node := &Node{
		nodeID:     nodeID,
		isPrimary:  isPrimary,
		replicas:   replicas,
		requestLog: make(map[string]*pb.Request),
		accountBalances: map[string]int{
			"AccountA": 1000,
			"AccountB": 500,
			"AccountC": 300,
		},
	}

	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", address, err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterPBFTServiceServer(grpcServer, node)

	fmt.Printf("Node %s (primary=%v) started, listening on %s\n", nodeID, isPrimary, address)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func main() {

	nodeID := flag.String("nodeID", "", "Unique node identifier")
	isPrimary := flag.Bool("primary", false, "Is this node the primary?")
	address := flag.String("address", ":50051", "Node address")
	replicas := flag.String("replicas", "", "Comma-separated list of replica addresses")

	flag.Parse()

	if *nodeID == "" || *address == "" {
		fmt.Println("Usage: go run node.go --nodeID=node1 --primary=true --address=:50051 --replicas=:50052,:50053")
		os.Exit(1)
	}

	replicaList := []string{}
	if *replicas != "" {
		replicaList = split(*replicas, ",")
	}

	startNode(*nodeID, *isPrimary, *address, replicaList)
}

func split(s, sep string) []string {
	var result []string
	result = append(result, strings.Split(s, sep)...)
	return result
}

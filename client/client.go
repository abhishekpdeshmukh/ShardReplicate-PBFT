package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	pb "ShardReplicate-PBFT/proto"

	"google.golang.org/grpc"
)

func readCSVTransactions(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("could not read file: %v", err)
	}

	return records, nil
}

func main() {
	f := 1
	requiredResponses := f + 1

	primaryAddress := "localhost:50051" // This should be dynamic in case of view changes, but for now, assume it's known.

	conn, err := grpc.Dial(primaryAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewPBFTServiceClient(conn)

	clientID := "client1"

	transactions, err := readCSVTransactions("../input.csv")
	if err != nil {
		log.Fatalf("Failed to read CSV: %v", err)
	}

	for _, transaction := range transactions {
		if len(transaction) < 6 {
			log.Printf("Skipping invalid transaction: %v", transaction)
			continue
		}

		operation := transaction[0]
		accountFrom := transaction[1]
		shardFrom := transaction[2]
		accountTo := transaction[3]
		shardTo := transaction[4]
		amount := transaction[5]

		amountS, err := strconv.ParseInt(amount, 10, 32)
		if err != nil {
			fmt.Println(err)
		}
		request := &pb.Request{
			Operation:   operation,
			AccountFrom: accountFrom,
			ShardFrom:   shardFrom,
			AccountTo:   accountTo,
			ShardTo:     shardTo,
			Amount:      amountS,
			ClientId:    clientID,
			Timestamp:   time.Now().UnixNano(),
		}

		// Send the request to the primary (or presumed primary)
		response, err := client.ClientRequest(context.Background(), request)
		if err != nil {
			log.Fatalf("ClientRequest failed: %v", err)
		}

		fmt.Printf("Received response from primary: %s\n", response.Result)

		responsesReceived := 1
		for responsesReceived < requiredResponses {

			time.Sleep(500 * time.Millisecond)
			responsesReceived++
			fmt.Printf("Received response %d/%d\n", responsesReceived, requiredResponses)
		}

		fmt.Printf("Completed operation: %s\n", request.Operation)
		time.Sleep(2 * time.Second)
	}
}

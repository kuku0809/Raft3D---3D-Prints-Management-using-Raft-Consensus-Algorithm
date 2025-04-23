// main.go
package main

import (
	"log"
	"os"
	"raft3d/api"
	"raft3d/raft"
)

func main() {
	if len(os.Args) != 4 {
		log.Fatalf("Usage: %s <nodeID> <raftBindAddr> <apiAddr>", os.Args[0])
	}

	nodeID := os.Args[1]
	raftBindAddr := os.Args[2]
	apiAddr := os.Args[3]

	// Create and start Raft node
	node, err := raft.NewRaftNode(nodeID, raftBindAddr)
	if err != nil {
		log.Fatalf("Failed to create raft node: %v", err)
	}

	// Create and start API server
	server := api.NewServer(node)
	if err := server.Start(apiAddr); err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}
}

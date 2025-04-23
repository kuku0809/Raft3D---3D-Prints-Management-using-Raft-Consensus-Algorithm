// node.go
package raft

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

type RaftNode struct {
	Raft *raft.Raft
	FSM  *FSM
}

func NewRaftNode(nodeID, bindAddr string) (*RaftNode, error) {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)
	config.SnapshotInterval = 30 * time.Second
	config.SnapshotThreshold = 2
	config.LogOutput = os.Stdout // Enable raft logging
	// Setup FSM
	fsm := NewFSM()

	// Create data directory if it doesn't exist
	dataDir := filepath.Join("raft-data", nodeID)
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("could not create data directory: %v", err)
	}

	// Create stores
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft-log.db"))
	if err != nil {
		return nil, fmt.Errorf("could not create log store: %v", err)
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft-stable.db"))
	if err != nil {
		return nil, fmt.Errorf("could not create stable store: %v", err)
	}

	snapshotStore, err := raft.NewFileSnapshotStore(dataDir, 3, os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("could not create snapshot store: %v", err)
	}

	// Create transport
	transport, err := raft.NewTCPTransport(bindAddr, nil, 3, 10*time.Second, os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("could not create transport: %v", err)
	}

	// Create Raft instance
	r, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("could not create raft instance: %v", err)
	}

	// Bootstrap cluster only if this is the first node AND there's no existing state
		if nodeID == "node1" {
		hasState, err := raft.HasExistingState(logStore, stableStore, snapshotStore)
		if err != nil {
		    return nil, fmt.Errorf("failed to check for existing state: %v", err)
		}

		if !hasState {
		    cfg := raft.Configuration{
		        Servers: []raft.Server{
		            {ID: raft.ServerID("node1"), Address: raft.ServerAddress("127.0.0.1:12000")},
		            {ID: raft.ServerID("node2"), Address: raft.ServerAddress("127.0.0.1:12001")},
		            {ID: raft.ServerID("node3"), Address: raft.ServerAddress("127.0.0.1:12002")},
		        },
		    }
		    f := r.BootstrapCluster(cfg)
		    if err := f.Error(); err != nil && err != raft.ErrCantBootstrap {
		        return nil, fmt.Errorf("bootstrap failed: %v", err)
		    }
		}
	}

	return &RaftNode{
		Raft: r,
		FSM:  fsm,
	}, nil
}

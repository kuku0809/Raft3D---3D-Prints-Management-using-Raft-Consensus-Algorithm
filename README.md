# Raft3D---3D-Prints-Management-using-Raft-Consensus-Algorithm

Raft3D is a backend API for a distributed 3D printer management system, where data persistence and consistency are implemented using the Raft Consensus Algorithm rather than a traditional centralized database.

The system manages distributed resources like 3D printers, filaments, and print jobs across a cluster of nodes.

Each node runs a Raft instance, ensuring that all operations (e.g., job creation, status updates) are replicated and agreed upon by a majority of nodes, enabling high availability, fault tolerance, and strong consistency, even in case of node failures.

The backend exposes a set of HTTP API endpoints to interact with the system. Internally, it uses Raft mechanisms such as leader election, log replication, and a state machine to maintain consistent state across the network.

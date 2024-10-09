# ShardReplicate-PBFT
PBFT in Sharded, Replicated Permissioned Blockchain

This repository contains an implementation of Practical Byzantine Fault Tolerance (PBFT) within a sharded, replicated, and permissioned blockchain environment. The goal is to simulate PBFT as a precursor to upgrading the system to use a more advanced Byzantine Fault Tolerant consensus mechanism, such as FaB Paxos, in the future. This project is part of the groundwork for a larger effort to implement SHARPER (Sharding Permissioned Blockchains Over Network Clusters), a cutting-edge algorithm designed for scalable blockchain systems, specifically focusing on managing cross-shard transactions and deterministic safety guarantees.

While SHARPER introduces a more comprehensive consensus mechanism with cross-shard transaction processing in parallel, this repository focuses on deploying PBFT in a SHARPER-like environment to provide strong Byzantine fault tolerance across clusters.

Key aspects:

Sharded and Replicated Data: Data is partitioned into shards, with each shard replicated across clusters to ensure fault tolerance.
PBFT Integration: PBFT ensures consensus in the presence of Byzantine nodes within each shard.
Preparation for FaB Paxos: This PBFT implementation serves as a foundational practice for future upgrades to FaB Paxos in this blockchain setting.
For more details on the SHARPER model and its decentralized flattened consensus approach, refer to the lniked documentation.
https://www3.cs.stonybrook.edu/~amiri/teaching/ds/24f/index.html

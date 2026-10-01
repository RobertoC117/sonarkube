# sonarkube

sonarkube is a network-connectivity toolkit written in Go that checks whether the network connections that *should* work in a system actually do.

🚧 **Work in progress — early development.** This is a personal learning project, built one stage at a time. See [Roadmap](#roadmap) for where it's headed.

## Why

This project exists to practice Go, networking, and Kubernetes together, by building one real system end to end instead of studying each topic in isolation.

## What works today

sonarkube is currently a CLI you run by hand:

- `sonarkube subnet <CIDR>` — calculate a subnet's network/broadcast address and usable host range (`--info`), or split it into smaller subnets (`--split`).
- `sonarkube probe tcp <host:port>` — check TCP connectivity to a host and port.
- `sonarkube probe dns <host>` — resolve a hostname and report resolution latency.
- `sonarkube probe icmp <host>` — ping a host with ICMP echo requests (requires admin privileges for the raw socket).

All commands support `--output table` (default) or `--output json`.

## Roadmap

One project, five stages, each building on the last:

| Stage | What sonarkube becomes |
|---|---|
| 1. CLI toolkit (current) | A terminal tool for subnet math and host probing (TCP/DNS/ICMP), run manually. |
| 2. Standalone service | A service that probes many targets concurrently and stores results in a database. |
| 3. Kubernetes-native | That same service, running inside a Kubernetes cluster. |
| 4. Production-grade | The system operated like production: security hardening, monitoring, and failure-recovery practice. |
| 5. Custom controller | sonarkube teaches Kubernetes a new concept — a network-watcher CRD/operator — so anyone can request a check by writing a config file. |

## Getting started

Requirements: Go 1.27+ (Docker optional).

```bash
git clone https://github.com/RobertoC117/sonarkube.git
cd sonarkube

# run directly
go run . subnet 10.0.0.0/24 --info
go run . subnet 10.0.0.0/24 --split 4
go run . probe tcp example.com:443
go run . probe dns example.com
sudo go run . probe icmp example.com

# or via make
make test
make build-image
```

## Tech stack

**Today:** Go, [Cobra](https://github.com/spf13/cobra), Docker.
**Planned:** a database (stage 2), Kubernetes (stage 3+).

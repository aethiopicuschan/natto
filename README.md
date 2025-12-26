# natto

[![License: MIT](https://img.shields.io/badge/License-MIT-brightgreen?style=flat-square)](/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/aethiopicuschan/natto.svg)](https://pkg.go.dev/github.com/aethiopicuschan/natto)
[![Go Report Card](https://goreportcard.com/badge/github.com/aethiopicuschan/natto)](https://goreportcard.com/report/github.com/aethiopicuschan/natto)
[![CI](https://github.com/aethiopicuschan/natto/actions/workflows/ci.yaml/badge.svg)](https://github.com/aethiopicuschan/natto/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/aethiopicuschan/natto/graph/badge.svg?token=6A4Y75PXH5)](https://codecov.io/gh/aethiopicuschan/natto)

`natto` is a lightweight Go library for NAT traversal using UDP hole punching.
It helps applications establish direct peer-to-peer connections across NATs with minimal dependencies and a simple API.

## Installation

```sh
go get -u github.com/aethiopicuschan/natto
```

## Packages

- `natto/nat`: Core NAT traversal functionalities, including UDP hole punching.

### TODO

- `natto/stun`: STUN client implementation for discovering public IP and port mappings.

## Example

See the [examples directory](./examples) for usage examples.

- [NAT Traversal with UDP Hole Punching](./examples/nat/udp)
- [NAT Traversal with UDP to TCP Upgrade](./examples/nat/tcp_upgrade)

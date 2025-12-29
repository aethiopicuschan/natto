# STUN Binding Example (Public Address Discovery)

This example demonstrates how to use the `stun` package to perform a
**STUN Binding Request** and discover the **public (NAT-mapped) UDP address**
of the local machine.

The example is designed as a **minimal but practical building block**
for NAT traversal and peer-to-peer (P2P) networking.

It focuses on:

- Sending a STUN Binding Request over UDP
- Receiving and parsing the STUN response
- Extracting the public IP address and port
- Reusing the same UDP socket for future P2P communication

---

## Overview

STUN (Session Traversal Utilities for NAT) allows a client behind NAT to
discover how it appears from the public internet.

This example performs the following steps:

1. Create a UDP socket bound to a local ephemeral port
2. Send a STUN Binding Request to a STUN server
3. Receive a Binding Success Response
4. Extract the `XOR-MAPPED-ADDRESS`
5. Print the public IP and port

This information is typically used as input for:

- UDP hole punching
- ICE / ICE-lite candidate exchange
- P2P rendezvous protocols
- NAT behavior detection (extended use)

---

## Why This Matters for P2P

For NAT traversal, **the UDP socket used for STUN must be reused** for
actual peer-to-peer traffic.

This example intentionally:

- Uses `net.DialUDP`
- Keeps the same `*net.UDPConn` alive
- Avoids creating a new socket after STUN

This mirrors real-world designs used in systems such as:

- WebRTC (ICE)
- Game networking engines
- P2P overlays
- Custom NAT traversal protocols

---

## Requirements

- Go 1.20+
- IPv4 networking
- Outbound UDP connectivity
- A reachable STUN server

The default STUN server is Google’s public STUN service:

```
stun.l.google.com:19302
```

---

## Running the Example

### Basic usage

```bash
go run .
```

Example output:

```
STUN server: stun.l.google.com:19302
Local UDP address: 0.0.0.0:53142
Public mapped address:
  IP  : 203.0.113.45
  Port: 62018
```

---

### Specify a custom STUN server

```bash
go run ./examples/basic --stun stun1.l.google.com:19302
```

```bash
go run ./examples/basic --stun stun.cloudflare.com:3478
```

---

### Custom timeout

```bash
go run ./examples/basic --timeout 3s
```

---

## Flags

| Flag        | Description                              |
| ----------- | ---------------------------------------- |
| `--stun`    | STUN server address (`host:port`)        |
| `--timeout` | Overall timeout for the STUN transaction |

---

## Output Explained

### Local UDP address

```
Local UDP address: 0.0.0.0:53142
```

This is the **local socket binding**.
The port is chosen by the OS and is the one actually mapped by the NAT.

---

### Public mapped address

```
Public mapped address:
  IP  : 203.0.113.45
  Port: 62018
```

This is the address as seen by the STUN server and represents:

- The NAT’s external IP
- The NAT-mapped UDP port

This `(IP, port)` pair is what you typically share with peers.

---

## Common Issues and Troubleshooting

### STUN request timed out

Possible causes:

- UDP blocked by firewall
- Incorrect STUN server address
- Very restrictive NAT or captive network
- Short timeout value

Try:

```bash
go run ./examples/basic --timeout 5s
```

---

### Public IP equals local IP

This usually means:

- You are on a machine with a public IP
- Or inside the same network as the STUN server (rare)

This is not an error.

---

## Design Notes

- Only **STUN Binding (RFC 5389)** is implemented
- `XOR-MAPPED-ADDRESS` is preferred over legacy `MAPPED-ADDRESS`
- No authentication (`MESSAGE-INTEGRITY`) is used
- No NAT behavior discovery (RFC 5780) is performed

This keeps the example simple and suitable as a foundation.

---

## What This Example Demonstrates

- Practical STUN usage in Go
- Correct UDP socket lifecycle for NAT traversal
- Public address discovery behind NAT
- A clean base for P2P / hole punching implementations

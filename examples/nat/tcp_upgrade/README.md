# NAT Traversal Example (UDP → TCP Upgrade)

This example demonstrates how to use the `nat` package to establish
a peer-to-peer connection using **UDP NAT traversal (hole punching)**,
and then **upgrade the connection to TCP** for reliable data transfer.

The example runs as **two separate processes** and is designed to work in:

- Local development environments
- Real global IP setups (VPS / cloud)
- NATed client → public server scenarios

UDP is used as a **control and traversal plane**,
and TCP is used as the **data plane** after connectivity is established.

---

## Directory Structure

```
examples/tcp_upgrade/
├── acceptor/
│ └── main.go
├── dialer/
│ └── main.go
└── README.md
```

---

## Overview

| Dialer (NAT / Private) | Phase                 | Acceptor (Global IP / VPS) |
| ---------------------- | --------------------- | -------------------------- |
| Dialer → Acceptor      | UDP Hole Punching     |                            |
|                        | UDP Hole Punching     | Acceptor → Dialer          |
|                        | TCP Port Announcement |                            |
| Dialer → Acceptor      | TCP Connect           |                            |
|                        | TCP Data              | TCP Data                   |

### Roles

- **Acceptor**
  - Runs on a machine with a public (global) IP address (e.g. VPS)
  - Accepts UDP hole punching
  - Opens a TCP listener after UDP is established
- **Dialer**
  - Runs behind NAT (home network, Wi-Fi, LTE, etc.)
  - Initiates UDP traversal
  - Connects to TCP once the port is announced

---

## Why UDP → TCP?

UDP NAT traversal is excellent for:

- Discovering reachable addresses
- Establishing connectivity through NAT
- Lightweight control messaging

However, application data often benefits from:

- Reliable delivery
- Ordered streams
- Backpressure handling

This example demonstrates a **hybrid design**:

- **UDP**: traversal + control plane
- **TCP**: reliable data plane

This mirrors real-world systems such as:

- WebRTC (ICE + DTLS/TCP fallback)
- QUIC / HTTP/3 bootstrapping
- Many P2P and overlay networks

---

## Requirements

- Go 1.25+
- IPv4 networking (`udp4`, `tcp4`)
- UDP and TCP traffic allowed on the Acceptor side

---

## Running the Example

### 1. Start the Acceptor

#### Local testing (same machine)

```bash
cd acceptor
go run . --ip=127.0.0.1
```

Example output:

```
=== Acceptor (UDP -> TCP upgrade) ===
My peer ID: peer-B
UDP listen addr to share: 127.0.0.1:57293

Waiting for UDP punch...
```

After a Dialer connects:

```
UDP session established!
Peer ID   : peer-A
Peer Addr : 127.0.0.1:55376

TCP listening on: 0.0.0.0:63883
Sent to dialer via UDP: TCP_PORT:63883
Waiting for TCP connect...
```

---

#### VPS / Global IP

```bash
go run . --ip=203.0.113.10
```

Example output:

```
UDP listen addr to share: 203.0.113.10:41782
```

> The port is chosen automatically.
> Only the IP portion is controlled by the flag.

### 2. Start the Dialer

```bash
cd dialer
go run .
```

When prompted:

```
Enter acceptor UDP address (host:port):
```

Paste the address printed by the Acceptor, for example:

```
127.0.0.1:55376
```

Example output:

```
=== Dialer (UDP -> TCP upgrade) ===
My peer ID: peer-A
Enter acceptor UDP address (host:port): 127.0.0.1:49689
Local UDP addr: 0.0.0.0:64063
Dialing UDP...
UDP connected!
Peer ID   : peer-B
Peer Addr : 127.0.0.1:49689
Behavior  : endpoint-independent-like

Waiting for TCP port announcement via UDP...
Got TCP port: 63893
Dialing TCP to: 127.0.0.1:63893
TCP connected!
```

---

### Interactive Messaging (TCP)

After TCP is connected:

- Type a line and press Enter on either process
- The message is sent over TCP
- The peer prints the received message

Example:

```
hello over tcp
```

Remote output:

```
tcp recv: "hello over tcp"
```

---

### Reliability Notes

#### TCP Port Announcement over UDP

The TCP port is announced via UDP using a control message:

```
TCP_PORT:<port>
```

Because UDP is unreliable:

- The Acceptor **re-sends the TCP_PORT message multiple times**
- The Dialer accepts the first valid announcement

This pattern is intentional and mirrors real-world UDP control protocols.

---

## Flags

### Acceptor

| Flag   | Description                           |
| ------ | ------------------------------------- |
| `--ip` | IP address to advertise to the Dialer |

Example:

```bash
go run . --ip=127.0.0.1
go run . --ip=203.0.113.10
```

The actual socket always binds to `0.0.0.0`.

---

## Common Issues and Troubleshooting

### UDP traversal timed out

Check:

- Correct IP and port were entered
- UDP is allowed through firewall
- Both sides are using IPv4
- Acceptor was started first
- The network is not using a symmetric NAT

---

### TCP connect failed

Check:

- TCP port announced was received
- TCP is allowed through firewall
- The Acceptor is reachable from the Dialer

---

### Firewall (VPS example)

```bash
sudo ufw allow 1:65535/udp
sudo ufw allow 1:65535/tcp
# or
sudo iptables -A INPUT -p udp -j ACCEPT
sudo iptables -A INPUT -p tcp -j ACCEPT
```

---

### Design Notes

- UDP is used only for traversal and control
- TCP is established only after reachability is confirmed
- UDP keepalive maintains NAT mappings during TCP setup
- The Session abstraction cleanly separates control and data planes

---

### What This Example Demonstrates

- Real UDP hole punching
- Cross-process NAT traversal
- Dynamic peer address updates
- Reliable TCP upgrade after UDP discovery
- Control-plane vs data-plane separation
- Practical, production-style P2P connection flow

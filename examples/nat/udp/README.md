# NAT Traversal Example (Two Processes, Advertised IP Switchable)

This example demonstrates how to use the `nat` package to establish
a peer-to-peer UDP connection using **NAT traversal (UDP hole punching)**.

The example runs as **two separate processes** and supports switching
the **advertised IP address via command-line flags**, making it usable
for both **local testing** and **real global IP environments (VPS, cloud)**.

---

## Directory Structure

```
examples/udp/
├── acceptor/
│ └── main.go
├── dialer/
│ └── main.go
└── README.md
```

---

## Overview

| Dialer (NAT / Private) | Communication     | Acceptor (Global IP / VPS) |
| ---------------------- | ----------------- | -------------------------- |
| Dialer → Acceptor      | UDP Hole Punching |                            |
|                        | UDP Hole Punching | Acceptor → Dialer          |

- **Acceptor**
  - Runs on a machine with a public (global) IP address (e.g. VPS)
  - Waits for incoming NAT traversal attempts
- **Dialer**
  - Runs behind NAT (home network, Wi-Fi, LTE, etc.)
  - Initiates the traversal

After successful punching, both peers communicate directly via UDP.

---

## Key Feature: Advertised IP via Flag

The Acceptor **binds to `0.0.0.0`** internally (correct for networking),
but the address shown to the Dialer can be customized using a flag.

This avoids confusion such as:

- `0.0.0.0:port` (not connectable)
- `127.0.0.1:port` (local only)
- `203.x.x.x:port` (public)

---

## Requirements

- Go 1.25+
- IPv4 networking (this example uses `udp4`)
- UDP traffic allowed on the Acceptor side

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
=== Acceptor ===
Peer ID: peer-B
Listening UDP addr: 127.0.0.1:57293
Send this address to the dialer.
```

---

#### VPS / Global IP

```bash
go run . --ip=203.0.113.10
```

Example output:

```
Listening UDP addr: 203.0.113.10:41782
```

> The port is chosen automatically.
> Only the IP portion is controlled by the flag.

---

### 2. Start the Dialer

```bash
cd dialer
go run .
```

When prompted:

```
Enter acceptor public UDP address (host:port):
```

Paste the address printed by the Acceptor, for example:

```
127.0.0.1:57293
```

---

## Successful Connection

If NAT traversal succeeds, the Dialer prints:

```
Dialing...
Connected!
Peer Addr: 127.0.0.1:57293
Behavior : endpoint-independent-like
```

The Acceptor prints:

```
Accepted!
Peer ID   : peer-A
Peer Addr :  127.0.0.1:65487
```

At this point, **direct peer-to-peer UDP communication is established**.

---

## Interactive Messaging

After connection:

- Type a line and press Enter on either process
- The message will appear on the other side

Example:

```
hello
```

Remote output:

```
recv from 127.0.0.1:58274: "hello"
```

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

### NAT traversal timed out

Check:

- Correct IP and port were entered
- UDP is allowed through firewall
- Both sides are using IPv4
- Acceptor was started first
- The network is not using a symmetric NAT (common on some mobile networks)

---

## Firewall (VPS example)

```bash
sudo ufw allow 1:65535/udp
# or
sudo iptables -A INPUT -p udp -j ACCEPT
```

---

## Notes on NAT Behavior

- This example works with most consumer NATs
- **Symmetric NATs may fail** (expected)
- The reported NAT behavior is a heuristic, not a guarantee

---

## Why Bind Address and Advertised Address Are Separate

- Binding to `0.0.0.0` allows receiving packets on all interfaces
- The advertised address must be reachable by the remote peer
- Conflating the two leads to connection failures

This separation is fundamental to NAT traversal systems.

---

## What This Example Demonstrates

- Real UDP hole punching
- Separate Dial / Accept roles
- Cross-process communication
- Production-like control flow
- Correct handling of advertised addresses
- A reusable Session abstraction (ordered send/recv, keepalive, remote update)

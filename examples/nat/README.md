# nat examples

## UDP

[udp example](./udp)

This example demonstrates how to use the `nat` package to establish a peer-to-peer UDP connection using **NAT traversal (UDP hole punching)**.

## TCP Upgrade

[tcp upgrade example](./tcp_upgrade)

This example demonstrates how to use the `nat` package to establish a peer-to-peer connection using **UDP NAT traversal (hole punching)**, and then **upgrade the connection to TCP** for reliable data transfer. UDP is used as a **control and traversal plane**, and TCP is used as the **data plane** after connectivity is established.

## NAT Type

[nat type example](./nat_type)

This example demonstrates how to use the `nat` package to determine the NAT type of a client behind a NAT using a STUN server.

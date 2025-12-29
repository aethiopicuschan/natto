package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/aethiopicuschan/natto/stun"
)

// defaultSTUNServer is Google's public STUN server.
const defaultSTUNServer = "stun.l.google.com:19302"

func main() {
	// Parse command-line arguments.
	stunServer := flag.String(
		"stun",
		defaultSTUNServer,
		"STUN server address (host:port)",
	)
	timeout := flag.Duration(
		"timeout",
		2*time.Second,
		"STUN request timeout",
	)
	flag.Parse()

	fmt.Println("STUN server:", *stunServer)

	// Resolve STUN server address.
	raddr, err := net.ResolveUDPAddr("udp", *stunServer)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to resolve STUN server:", err)
		os.Exit(1)
	}

	// Create a local UDP address (ephemeral port).
	laddr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	}

	// Dial UDP (this creates a "connected" UDP socket).
	conn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to dial UDP:", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Local UDP address:", conn.LocalAddr())

	// Create STUN client.
	client := stun.NewClient()
	client.Timeout = *timeout

	// Context with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Perform STUN Binding Request.
	mapped, err := client.BindingRequestConn(ctx, conn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "STUN binding request failed:", err)
		os.Exit(1)
	}

	// Print the public (NAT-mapped) address.
	fmt.Println("Public mapped address:")
	fmt.Println("  IP  :", mapped.IP.String())
	fmt.Println("  Port:", mapped.Port)
}

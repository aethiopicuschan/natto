package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/aethiopicuschan/natto/nat"
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Detect NAT type
	result, err := nat.DetectNAT(ctx, conn)
	if err != nil {
		panic(err)
	}

	// Print NAT detection result
	fmt.Println("=== NAT Detection Result ===")
	fmt.Printf("Local Address     : %s\n", result.LocalAddr)
	fmt.Printf("Mapped Address #1 : %s:%d\n", result.MappedAddr1.IP, result.MappedAddr1.Port)
	fmt.Printf("Mapped Address #2 : %s:%d\n", result.MappedAddr2.IP, result.MappedAddr2.Port)
	fmt.Printf("NAT Type          : %s\n", result.Type)
	fmt.Printf("Mapping Behavior  : %s\n", result.Mapping)
	fmt.Printf("Filtering Behavior: %s\n", result.Filtering)
	fmt.Printf("UDP Punching OK   : %v\n", result.PunchingOK)
}

package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/aethiopicuschan/natto/nat"
)

func mustListen() *net.UDPConn {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 0,
	})
	if err != nil {
		panic(err)
	}
	return conn
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Dialer ===")
	fmt.Println("My peer ID: peer-A")
	fmt.Print("Enter acceptor UDP address (host:port): ")

	addrStr, _ := reader.ReadString('\n')
	addrStr = strings.TrimSpace(addrStr)

	remoteAddr, err := net.ResolveUDPAddr("udp", addrStr)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn := mustListen()
	defer conn.Close()

	fmt.Println("Local UDP addr:", conn.LocalAddr().String())

	mux := nat.NewMux(conn)
	mux.Start(ctx)

	peerB := &nat.Peer{
		ID:   "peer-B",
		Addr: remoteAddr,
	}

	fmt.Println("Dialing...")

	sess, res, err := nat.Dial(
		ctx,
		mux,
		"peer-A",
		peerB,
		nat.DialOptions{
			Interval:          200 * time.Millisecond,
			Queue:             16,
			KeepaliveInterval: 5 * time.Second,
		},
	)
	if err != nil {
		fmt.Println("dial error:", err)
		os.Exit(1)
	}

	fmt.Println("Connected!")
	fmt.Println("Peer ID   :", res.PeerID)
	fmt.Println("Peer Addr :", res.Addr)
	fmt.Println("Behavior  :", res.Behavior)

	// ---- receive loop ----
	go func() {
		for {
			msg, addr, err := sess.Recv(ctx)
			if err != nil {
				fmt.Println("recv error:", err)
				return
			}
			if len(msg) == 0 {
				fmt.Printf("recv keepalive from %s\n", addr)
				continue
			}
			fmt.Printf("recv from %s: %q\n", addr, msg)
		}
	}()

	// ---- stdin → send ----
	fmt.Println("Type messages and press Enter to send.")

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		if err := sess.Send([]byte(line)); err != nil {
			fmt.Println("send error:", err)
			return
		}
	}
}

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

var publicIP = flag.String("ip", "127.0.0.1", "address to show to dialer")

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
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := mustListen()
	defer conn.Close()

	fmt.Println("=== Acceptor ===")
	fmt.Println("My peer ID: peer-B")
	local := conn.LocalAddr().(*net.UDPAddr)
	fmt.Printf(
		"Listening UDP addr: %s:%d\n",
		*publicIP,
		local.Port,
	)
	fmt.Println("Send this address to the dialer.")
	fmt.Println()

	mux := nat.NewMux(conn)
	mux.Start(ctx)

	acceptor := nat.NewAcceptor(
		mux,
		"peer-B",
		nat.AcceptOptions{
			Queue:             16,
			KeepaliveInterval: 5 * time.Second,
		},
	)

	fmt.Println("Waiting for incoming punch...")

	sess, res, err := acceptor.Accept(ctx)
	if err != nil {
		fmt.Println("accept error:", err)
		os.Exit(1)
	}

	fmt.Println("Accepted!")
	fmt.Println("Peer ID   :", res.PeerID)
	fmt.Println("Peer Addr :", res.Addr)

	// ---- simple receive loop ----
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
		var line string
		if _, err := fmt.Scanln(&line); err != nil {
			return
		}
		if err := sess.Send([]byte(line)); err != nil {
			fmt.Println("send error:", err)
			return
		}
	}
}

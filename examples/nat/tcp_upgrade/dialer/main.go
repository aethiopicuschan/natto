package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aethiopicuschan/natto/nat"
)

func mustListenUDP() *net.UDPConn {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 0,
	})
	if err != nil {
		panic(err)
	}
	return conn
}

func parseTCPPortMsg(b []byte) (int, bool) {
	s := strings.TrimSpace(string(b))
	if !strings.HasPrefix(s, "TCP_PORT:") {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(s, "TCP_PORT:"))
	return n, err == nil && n > 0 && n < 65536
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Dialer (UDP -> TCP upgrade) ===")
	fmt.Println("My peer ID: peer-A")
	fmt.Print("Enter acceptor UDP address (host:port): ")

	addrStr, _ := reader.ReadString('\n')
	addrStr = strings.TrimSpace(addrStr)

	host, _, err := net.SplitHostPort(addrStr)
	if err != nil {
		fmt.Println("invalid address:", err)
		os.Exit(1)
	}

	remoteUDP, err := net.ResolveUDPAddr("udp4", addrStr)
	if err != nil {
		panic(err)
	}

	udpConn := mustListenUDP()
	defer udpConn.Close()

	fmt.Println("Local UDP addr:", udpConn.LocalAddr().String())

	mux := nat.NewMux(udpConn)

	// mux はプロセス寿命で動かす
	mux.Start(context.Background())

	// Dial 専用 context（成功したら死んでよい）
	dialCtx, cancelDial := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelDial()

	peerB := &nat.Peer{
		ID:   "peer-B",
		Addr: remoteUDP,
	}

	fmt.Println("Dialing UDP...")

	udpSess, res, err := nat.Dial(
		dialCtx,
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
		fmt.Println("udp dial error:", err)
		os.Exit(1)
	}

	udpSess.UpdateRemote(res.Addr)

	fmt.Println("UDP connected!")
	fmt.Println("Peer ID   :", res.PeerID)
	fmt.Println("Peer Addr :", res.Addr)
	fmt.Println("Behavior  :", res.Behavior)
	fmt.Println()

	// ---- Run context ----
	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	fmt.Println("Waiting for TCP port announcement via UDP...")
	var tcpPort int

	for {
		msg, _, err := udpSess.RecvData(runCtx)
		if err != nil {
			fmt.Println("udp recv error:", err)
			os.Exit(1)
		}
		if len(msg) == 0 {
			continue // keepalive
		}
		if p, ok := parseTCPPortMsg(msg); ok {
			tcpPort = p
			break
		}
	}

	fmt.Println("Got TCP port:", tcpPort)

	// ---- TCP connect ----
	tcpAddr := net.JoinHostPort(host, strconv.Itoa(tcpPort))
	fmt.Println("Dialing TCP to:", tcpAddr)

	tcpConn, err := net.DialTimeout("tcp4", tcpAddr, 10*time.Second)
	if err != nil {
		fmt.Println("tcp dial error:", err)
		os.Exit(1)
	}
	defer tcpConn.Close()

	fmt.Println("TCP connected!")
	fmt.Println()

	// ---- TCP receive loop ----
	go func() {
		r := bufio.NewReader(tcpConn)
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				fmt.Println("tcp recv error:", err)
				cancelRun()
				return
			}
			line = strings.TrimRight(line, "\r\n")
			fmt.Printf("tcp recv: %q\n", line)
		}
	}()

	// ---- stdin -> TCP send ----
	fmt.Println("Type messages and press Enter to send (over TCP).")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		if !strings.HasSuffix(line, "\n") {
			line += "\n"
		}
		if _, err := tcpConn.Write([]byte(line)); err != nil {
			fmt.Println("tcp send error:", err)
			return
		}
	}
}

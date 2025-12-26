package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/aethiopicuschan/natto/nat"
)

var publicIP = flag.String("ip", "127.0.0.1", "address to show to dialer")

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

func mustListenTCP() *net.TCPListener {
	ln, err := net.ListenTCP("tcp4", &net.TCPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 0, // pick random
	})
	if err != nil {
		panic(err)
	}
	return ln
}

func main() {
	flag.Parse()

	// mux / session の寿命（プロセス寿命）
	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	udpConn := mustListenUDP()
	defer udpConn.Close()

	fmt.Println("=== Acceptor (UDP -> TCP upgrade) ===")
	fmt.Println("My peer ID: peer-B")

	udpLocal := udpConn.LocalAddr().(*net.UDPAddr)
	fmt.Printf("UDP listen addr to share: %s:%d\n", *publicIP, udpLocal.Port)
	fmt.Println()

	mux := nat.NewMux(udpConn)
	mux.Start(runCtx)

	acceptor := nat.NewAcceptor(
		mux,
		"peer-B",
		nat.AcceptOptions{
			Queue:             16,
			KeepaliveInterval: 5 * time.Second,
		},
	)

	fmt.Println("Waiting for UDP punch...")

	// Accept 専用（成功したら死んでよい / mux とは分離）
	acceptCtx, cancelAccept := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelAccept()

	udpSess, res, err := acceptor.Accept(acceptCtx)
	if err != nil {
		fmt.Println("accept error:", err)
		os.Exit(1)
	}

	udpSess.UpdateRemote(res.Addr)

	fmt.Println("UDP session established!")
	fmt.Println("Peer ID   :", res.PeerID)
	fmt.Println("Peer Addr :", res.Addr)
	fmt.Println()

	// --- TCP listen ---
	tcpLn := mustListenTCP()
	defer tcpLn.Close()
	tcpPort := tcpLn.Addr().(*net.TCPAddr).Port

	// Notify dialer via UDP session.
	notify := fmt.Sprintf("TCP_PORT:%d", tcpPort)
	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		for range 10 {
			_ = udpSess.SendData([]byte(notify))
			<-ticker.C
		}
	}()

	fmt.Println("TCP listening on:", tcpLn.Addr().String())
	fmt.Println("Sent to dialer via UDP:", notify)
	fmt.Println("Waiting for TCP connect...")

	tcpConn, err := tcpLn.AcceptTCP()
	if err != nil {
		fmt.Println("tcp accept error:", err)
		os.Exit(1)
	}
	defer tcpConn.Close()

	fmt.Println("TCP connected from:", tcpConn.RemoteAddr().String())
	fmt.Println()

	// --- TCP receive loop ---
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

	// --- stdin -> TCP send ---
	fmt.Println("Type messages and press Enter to send (over TCP).")
	stdin := bufio.NewReader(os.Stdin)
	for {
		line, err := stdin.ReadString('\n')
		if err != nil {
			return
		}
		// Keep newline so the other side can ReadString('\n').
		if !strings.HasSuffix(line, "\n") {
			line += "\n"
		}
		if _, err := tcpConn.Write([]byte(line)); err != nil {
			fmt.Println("tcp send error:", err)
			return
		}
	}
}

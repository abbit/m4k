package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/abbit/m4k/internal/protocol"
)

// TODO: handle signals
// TODO: remove timeout
// TOOD: measure performance with HTTP

func handleConnection(conn net.Conn) {
	p := protocol.New(conn)
	defer p.Close()

	err := p.ReceiveManga()
	if err != nil {
		log.Fatalf("Error when receiving manga: %v\n", err)
	}
}

type Flags struct {
	port, pidfile string
}

func parseFlags() *Flags {
	flags := &Flags{}
	flag.StringVar(&flags.port, "port", "49494", "Port for receiver")
	flag.StringVar(&flags.pidfile, "pidfile", "", "Path to where store pid file")
	flag.Parse()

	if flags.pidfile == "" {
		log.Fatalf("-pidfile option is required.\n")
	}

	return flags
}

func main() {
	flags := parseFlags()

	pidfile, err := os.Create(flags.pidfile)
	if err != nil {
		log.Fatalf("Error when creating pid file: %v\n", err)
	}
	defer func() {
		if err := pidfile.Close(); err != nil {
			log.Printf("Error when closing pid file: %v\n", err)
		}
		if err := os.Remove(flags.pidfile); err != nil {
			log.Printf("Error when removing pid file: %v\n", err)
		}
	}()

	if _, err := fmt.Fprint(pidfile, os.Getpid()); err != nil {
		log.Fatalf("Error when writing to pid file: %v\n", err)
	}

	l, err := net.Listen("tcp", ":"+flags.port)
	if err != nil {
		log.Fatalf("Error when starting server: %v\n", err)
	}
	defer l.Close()
	log.Printf("Listening on %s\n", l.Addr().String())

	connChan := make(chan net.Conn, 1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalf("Error when accepting connection: %v\n", err)
		}
		conn.SetDeadline(time.Now().Add(15 * time.Minute))
		connChan <- conn
	}()

	select {
	case conn := <-connChan:
		log.Printf("%s connected, starting file receiving...\n", conn.LocalAddr().String())
		handleConnection(conn)
	case <-time.After(2 * time.Minute):
		log.Println("Hit timeout")
	}

    log.Println("Exiting.")
}

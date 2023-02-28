package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/abbit/m4k/internal/protocol"
)

// TODO: handle signals
// TODO: remove timeout
// TOOD: measure performance with HTTP
// TODO: incapsulate logic in server struct
// TODO: handle multiple connections?

func handleConnection(conn net.Conn, destdir string) {
	p := protocol.New(conn)
	defer p.Close()

	err := p.ReceiveManga(destdir)
	if err != nil {
		log.Printf("Error when receiving manga: %v\n", err)
	}
}

type Flags struct {
	port string
    pidfile string
    destdir string
}

func parseFlags() *Flags {
	flags := &Flags{}
	flag.StringVar(&flags.pidfile, "pidfile", "", "Path to where store pid file")
	flag.StringVar(&flags.port, "port", "49494", "Port for receiver")
    flag.StringVar(&flags.destdir, "destdir", "/mnt/us/documents/Manga", "Path destination directory")
	flag.Parse()

	if flags.pidfile == "" {
		log.Fatalf("-pidfile option is required.\n")
	}

    absdest, err := filepath.Abs(flags.destdir)
    if err != nil {
        log.Fatalf("Error when resolving absolute destination directory path: %v.\n", err)
    }

    flags.destdir = absdest

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
	log.Printf("Listening on %s, destination directory - %s\n", l.Addr().String(), flags.destdir)

	connChan := make(chan net.Conn, 1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
            // TODO: dont print error if listener is closed when exiting
			log.Printf("Error when accepting connection: %v\n", err)
            return
		}
		conn.SetDeadline(time.Now().Add(15 * time.Minute))
		connChan <- conn
	}()

    exitsig := make(chan os.Signal, 1) 
    signal.Notify(exitsig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case conn := <-connChan:
		log.Printf("%s connected, starting file receiving...\n", conn.LocalAddr().String())
		handleConnection(conn, flags.destdir)
	case <-time.After(5 * time.Minute):
		log.Println("Hit timeout")
    case <-exitsig:
        log.Println("Received exit signal.")
	}

    log.Println("Exiting...")
}

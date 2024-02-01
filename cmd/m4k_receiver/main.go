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

// TODO: wait for clients to close their connections when shutting down gracefully
// TODO: measure performance with HTTP
// TODO: handle multiple connections?
// TODO: resumable upload?

type server struct {
	addr    string
	destDir string
}

func NewServer(addr, destdir string) *server {
	return &server{
		addr:    addr,
		destDir: destdir,
	}
}

func (srv *server) ListenAndServe() error {
	l, err := net.Listen("tcp", srv.addr)
	if err != nil {
		return err
	}
	log.Printf("Listening on %s, destination directory - %s\n", l.Addr().String(), srv.destDir)
	return srv.Serve(l)
}

func (srv *server) Serve(l net.Listener) error {
	defer l.Close()

	connChan := make(chan net.Conn, 1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			// TODO: dont print error if listener is closed when exiting
			log.Printf("Error when accepting connection: %v\n", err)
			return
		}
		conn.SetDeadline(time.Now().Add(2 * time.Hour))
		connChan <- conn
	}()

	// TODO: handle signals outside of server
	exitsig := make(chan os.Signal, 1)
	signal.Notify(exitsig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case conn := <-connChan:
		srv.handleConnection(conn)
	case <-exitsig:
		log.Println("Received exit signal, exiting...")
	}

	return nil
}

func (srv *server) handleConnection(conn net.Conn) {
	log.Printf("%s connected, starting file receiving...\n", conn.LocalAddr().String())
	p := protocol.New(conn)
	defer p.Close()

	err := p.ReceiveManga(srv.destDir)
	if err != nil {
		log.Printf("Error when receiving manga: %v\n", err)
	}
}

type Flags struct {
	port    string
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

	// TODO: move handling of pid file to separate functions
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

	srv := NewServer(":"+flags.port, flags.destdir)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Error while serving: %v\n", err)
	}
}

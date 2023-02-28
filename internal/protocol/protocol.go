package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

type Protocol struct {
	conn net.Conn
}

func New(conn net.Conn) *Protocol {
	return &Protocol{conn}
}

func (p *Protocol) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// writes length-prefixed bytes data
func (p *Protocol) write(b []byte) (n int64, err error) {
	data := new(bytes.Buffer)
	if err = binary.Write(data, binary.LittleEndian, uint32(len(b))); err != nil {
		return
	}

	if _, err = data.Write(b); err != nil {
		return
	}

	n, err = io.Copy(p.conn, data)
	n -= 4 // remove header size
	return
}

// reads length-prefixed bytes data
func (p *Protocol) read() (b []byte, err error) {
	header := make([]byte, 4)
	if _, err = io.ReadFull(p.conn, header); err != nil {
		err = fmt.Errorf("reading header: %v", err)
		return
	}

	var numBytesUint32 uint32
	if err = binary.Read(bytes.NewReader(header), binary.LittleEndian, &numBytesUint32); err != nil {
		return
	}

	b = make([]byte, int(numBytesUint32))
	if _, err = io.ReadFull(p.conn, b); err != nil {
		return
	}
	return
}

func (p *Protocol) SendManga(name string, r io.Reader) error {
	nameBytes := []byte(name)
	n, err := p.write(nameBytes)
	if err != nil {
		return fmt.Errorf("writing file name: %v", err)
	}
	if int(n) != len(nameBytes) {
		fmt.Fprintf(os.Stderr, "name bytes len - %d, sent bytes - %d\n", len(nameBytes), int(n))
	}

	if _, err := io.Copy(p.conn, r); err != nil {
		return fmt.Errorf("sending bytes: %v", err)
	}

	return nil
}

func (p *Protocol) ReceiveManga(destdir string) error {
	// receive file name
	nameBytes, err := p.read()
	if err != nil {
		return fmt.Errorf("reading name from conn: %v", err)
	}
	name := string(nameBytes)

	// create dest file
	// TODO: handle situation when file already exists
	file, err := os.Create(filepath.Join(destdir, fmt.Sprintf("%s.cbz", filepath.Base(name))))
	if err != nil {
		return fmt.Errorf("creating receiving file: %v", err)
	}
	defer file.Close()

	// receive file data
	if _, err := io.Copy(file, p.conn); err != nil {
		return fmt.Errorf("reading bytes: %v", err)
	}

	return nil
}

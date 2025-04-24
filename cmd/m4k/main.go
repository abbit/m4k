package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/abbit/m4k/internal/comicbook"
	"github.com/abbit/m4k/internal/log"
	"github.com/abbit/m4k/internal/protocol"
	"github.com/abbit/m4k/internal/transform"
	"github.com/abbit/m4k/internal/util"
	"github.com/schollz/progressbar/v3"
)

const (
	KindlePW5Width  = 1236 // px
	KindlePW5Height = 1648 // px
)

// TODO: add a way to only send file without processing

func saveComicBookToFile(path string, cb *comicbook.ComicBook) error {
	file, err := os.Create(filepath.Join(path, cb.FileName()))
	if err != nil {
		return err
	}
	defer file.Close()

	cbReader, err := cb.Reader()
	if err != nil {
		os.Remove(path)
		return err
	}
	progress := progressbar.DefaultBytes(
		cbReader.Size(),
		"saving...",
	)

	if _, err = io.Copy(io.MultiWriter(file, progress), cbReader); err != nil {
		os.Remove(path)
		return err
	}

	return nil
}

// upload comicbook to kindle over sftp
func sendComicBookToKindle(addr string, cb *comicbook.ComicBook) error {
	log.Info.Println("Connecting to server...")
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		log.Error.Fatalln(err)
	}
	p := protocol.New(conn)
	defer p.Close()

	log.Info.Println("Connected, sending file...")
	conn.SetDeadline(time.Now().Add(10 * time.Minute))

	cbReader, err := cb.Reader()
	if err != nil {
		return err
	}
	progress := progressbar.DefaultBytes(
		cbReader.Size(),
		"uploading...",
	)

	return p.SendManga(cb.Name, io.TeeReader(cbReader, progress))
}

type Flags struct {
	srcdir     string
	dstdir     string
	rotatepage bool
	name       string
	addr       string
	save       bool
	upload     bool
	cleanup    bool
}

func parseFlags() *Flags {
	flags := &Flags{}
	flag.StringVar(&flags.srcdir, "src", "", "Path to directory with .cbz files")
	flag.StringVar(&flags.name, "name", "", "Name for combined .cbz file without extension")
	flag.StringVar(&flags.dstdir, "dst", "", "Path to directory to where save merged file (Default: same as srcdir)")
	flag.BoolVar(&flags.rotatepage, "rotatepage", false, "Rotate page")
	flag.BoolVar(&flags.save, "save", false, "Save combined file")
	flag.BoolVar(&flags.upload, "upload", false, "Upload combined file to Kindle")
	flag.StringVar(&flags.addr, "addr", "", "Address (host or host:port) of Kindle's receiver server. If port is not specified, default 49494 will be used")
	flag.BoolVar(&flags.cleanup, "cleanup", false, "Remove merged .cbz files")
	flag.Parse()

	// check if required options are specified
	if flags.srcdir == "" {
		log.Error.Fatalf("-src option is required.\n")
	}
	if flags.name == "" {
		log.Error.Fatalf("-name option is required.\n")
	}
	if !flags.save && !flags.upload {
		log.Error.Fatalf("-save or -upload is required.\n")
	}

	// check if required options are specified for '-upload' action
	if flags.upload {
		if flags.addr == "" {
			log.Error.Fatalf("-addr option is required.\n")
		}

		// add default port if not specified
		if strings.LastIndex(flags.addr, ":") == -1 {
			flags.addr += ":49494"
		}
	}

	// set default values
	if flags.dstdir == "" {
		flags.dstdir = flags.srcdir
	}

	return flags
}

func main() {
	flags := parseFlags()

	if err := validateName(flags.name); err != nil {
		log.Error.Fatalf("failed validating name: %v\n", err)
	}

	log.Info.Println("Searching cbz files...")
	cbzFiles, err := util.FilterDirFilePaths(flags.srcdir, func(p string) bool { return path.Ext(p) == ".cbz" })
	if err != nil {
		log.Error.Fatalf("%v\n", err)
	}

	log.Info.Println("Reading cbz files...")
	var comicbooks []*comicbook.ComicBook
	for _, path := range cbzFiles {
		cb, err := comicbook.ReadComicBook(path)
		if err != nil {
			log.Error.Fatalf("failed reading comicbook from path %s: %v\n", path, err)
		}
		comicbooks = append(comicbooks, cb)
	}

	log.Info.Println("Merging cbz files...")
	combined := comicbook.MergeComicBooks(comicbooks, flags.name)

	log.Info.Println("Transforming combined file for Kindle...")
	progress := progressbar.Default(int64(len(combined.Pages)), "Transforming pages...")
	transformOpts := &transform.Options{
		Rotate:   flags.rotatepage,
		Width:    KindlePW5Width,
		Height:   KindlePW5Height,
		Encoding: "jpg",
		Callback: func() { progress.Add(1) },
	}
	if err := transform.TransformComicBook(combined, transformOpts); err != nil {
		log.Error.Fatalf("while transforming pages: %v\n", err)
	}

	if flags.save {
		log.Info.Println("Saving combined file...")
		if err := saveComicBookToFile(flags.dstdir, combined); err != nil {
			log.Error.Fatalf("while saving combined file: %v\n", err)
		}
	}

	if flags.upload {
		log.Info.Println("Uploading combined file to Kindle...")
		if err := sendComicBookToKindle(flags.addr, combined); err != nil {
			log.Error.Fatalf("while sending to Kindle: %v\n", err)
		}
	}

	if flags.cleanup {
		log.Info.Println("Removing merged files...")
		if err := util.RemoveFiles(cbzFiles); err != nil {
			log.Error.Fatalf("while removing merged files: %v\n", err)
		}
	}

	log.Info.Println("Done!")
}

var disallowedSymbols = []rune("/:")

func validateName(name string) error {
	for _, disallowedSymbol := range disallowedSymbols {
		if strings.ContainsRune(name, disallowedSymbol) {
			return fmt.Errorf("%q contains disallowed symbol %q", name, disallowedSymbol)
		}
	}

	return nil
}

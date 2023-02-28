package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/abbit/m4k/internal/protocol"
	"github.com/abbit/m4k/internal/util"
	"github.com/disintegration/imaging"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"
)

// TODO: add a way to only send file without processing
// TODO: print merged file size

const (
	KindlePW5Width  = 1236 // px
	KindlePW5Height = 1648 // px
)

var logError *log.Logger = log.New(os.Stderr, "Error: ", 0)
var logInfo *log.Logger = log.New(os.Stdout, "", 0)

// returns chapter name without padded index
func withoutPaddedIndex(name string) string {
	before, after, found := strings.Cut(name, "_")
	if found {
		return after
	}
	return before
}

// checks if file is actual manga page or metadata file
func isPage(name string) bool {
	ext := filepath.Ext(name)
	if ext == ".jpg" ||
		ext == ".jpeg" ||
		ext == ".png" {
		return true
	}

	return false
}

type Page struct {
	Data      []byte
	Number    uint64
	Extension string
}

func PageFromFile(zfile *zip.File) (*Page, error) {
	file, err := zfile.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(file)
	if err != nil {
		return nil, err
	}

	number, err := strconv.ParseUint(util.PathStem(zfile.Name), 10, 64)
	if err != nil {
		return nil, err
	}

	return &Page{
		Data:      buf.Bytes(),
		Number:    number,
		Extension: filepath.Ext(zfile.Name),
	}, nil
}

func (p *Page) FileName() string {
	return fmt.Sprintf("%06d.%s", p.Number, p.Extension)
}

func (p *Page) TransformForKindle() error {
	// decode image
	buf := bytes.NewBuffer(p.Data)
	img, err := imaging.Decode(buf)
	if err != nil {
		return err
	}

	// transform image
	if imgsize := img.Bounds().Size(); imgsize.X > imgsize.Y {
		img = imaging.Rotate90(img) // rotate if Dx > Dy
	}
	img = imaging.Resize(img, KindlePW5Width, KindlePW5Height, imaging.Lanczos)
	img = imaging.Grayscale(img)

	// encode image
	if err = imaging.Encode(buf, img, imaging.JPEG, imaging.JPEGQuality(75)); err != nil {
		return err
	}
	p.Data = buf.Bytes()
	p.Extension = ".jpg"

	return nil
}

type ComicBook struct {
	Pages   []*Page
	Name    string
	cbzData []byte
}

func readComicBook(path string) (*ComicBook, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}

	var pages []*Page
	for _, f := range r.File {
		if isPage(f.Name) {
			page, err := PageFromFile(f)
			if err != nil {
				return nil, err
			}
			pages = append(pages, page)
		}
	}
	// sort pages by page number
	sort.Slice(pages, func(i, j int) bool { return pages[i].Number < pages[j].Number })

	return &ComicBook{Pages: pages, Name: withoutPaddedIndex(util.PathStem(path))}, nil
}

func (cb *ComicBook) FileName() string {
	return cb.Name + ".cbz"
}

func (cb *ComicBook) TransformForKindle() error {
	g := &errgroup.Group{}

    progress := progressbar.Default(int64(len(cb.Pages)), "Transforming pages...")
	for _, p := range cb.Pages {
		p := p
		g.Go(func() error {
			if err := p.TransformForKindle(); err != nil {
				return err
			}
            progress.Add(1)
			return nil
		})
	}

	return g.Wait()
}

func (cb *ComicBook) WriteTo(wr io.Writer) (n int64, err error) {
	w := zip.NewWriter(wr)
	defer w.Close()

	// write pages to zip archive
	for _, page := range cb.Pages {
		file, err := w.Create(page.FileName())
		if err != nil {
			return n, err
		}

		nfile, err := file.Write(page.Data)
		n += int64(nfile)
		if err != nil {
			return n, err
		}
	}

	return
}

func (cb *ComicBook) fillCbzData() error {
    buf := new(bytes.Buffer)
    if _, err := cb.WriteTo(buf); err != nil {
        return err
    }
    cb.cbzData = buf.Bytes()
    return nil
}

func (cb *ComicBook) Reader() (*bytes.Reader, error) {
	if cb.cbzData == nil {
        if err := cb.fillCbzData(); err != nil {
            return nil, err
        }
	}

	return bytes.NewReader(cb.cbzData), nil
}

func mergeComicBooks(comicbooks []*ComicBook, name string) *ComicBook {
	var pages []*Page
	pageNumber := uint64(0)
	for _, comicbook := range comicbooks {
		for _, p := range comicbook.Pages {
			pageNumber++
			pages = append(pages, &Page{
				Data:      p.Data,
				Extension: p.Extension,
				Number:    pageNumber,
			})
		}
	}

	return &ComicBook{Name: name, Pages: pages}
}

func saveComicBookToFile(path string, cb *ComicBook) error {
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
func sendComicBookToKindle(addr string, cb *ComicBook) error {
	fmt.Println("Connecting to server...")
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		log.Fatalln(err)
	}
	p := protocol.New(conn)
	defer p.Close()

	fmt.Println("Connected, sending file...")
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
	srcdir  string
	dstdir  string
	name    string
	addr    string
	save    bool
	upload  bool
	cleanup bool
}

func parseFlags() *Flags {
	flags := &Flags{}
	flag.StringVar(&flags.srcdir, "src", "", "Path to directory with .cbz files")
	flag.StringVar(&flags.name, "name", "", "Name for combined .cbz file without extension")
	flag.StringVar(&flags.dstdir, "dst", "", "Path to directory to where save merged file (Default: same as srcdir)")
	flag.BoolVar(&flags.save, "save", false, "Save combined file")
	flag.BoolVar(&flags.upload, "upload", false, "Upload combined file to Kindle")
	flag.StringVar(&flags.addr, "addr", "192.168.0.109:49494", "Address (host:port) of Kindle's SSH server")
	flag.BoolVar(&flags.cleanup, "cleanup", false, "Remove merged .cbz files")
	flag.Parse()

	if flags.srcdir == "" {
		logError.Fatalf("-src option is required.\n")
	}
	if flags.name == "" {
		logError.Fatalf("-name option is required.\n")
	}
	if !flags.save && !flags.upload {
		logError.Fatalf("-save or -upload is required.\n")
	}
	if flags.dstdir == "" {
		flags.dstdir = flags.srcdir
	}

	return flags
}

func main() {
	flags := parseFlags()

	logInfo.Println("Searching cbz files...")
	cbzFiles, err := util.FindFilesWithExt(flags.srcdir, ".cbz")
	if err != nil {
		logError.Fatalf("%v\n", err)
	}

	logInfo.Println("Reading cbz files...")
	var comicbooks []*ComicBook
	for _, f := range cbzFiles {
		cb, err := readComicBook(f)
		if err != nil {
			logError.Fatalf("failed reading comicbook from path %s: %v\n", f, err)
		}
		comicbooks = append(comicbooks, cb)
	}

	logInfo.Println("Merging cbz files...")
	combined := mergeComicBooks(comicbooks, flags.name)

	logInfo.Println("Transforming combined file for Kindle...")
	if err := combined.TransformForKindle(); err != nil {
		logError.Fatalf("while transforming pages: %v\n", err)
	}

	if flags.save {
		logInfo.Println("Saving combined file...")
		if err := saveComicBookToFile(flags.dstdir, combined); err != nil {
			logError.Fatalf("while saving combined file: %v\n", err)
		}
	}

	if flags.cleanup {
		logInfo.Println("Removing merged files...")
		if err := util.RemoveFiles(cbzFiles); err != nil {
			logError.Fatalf("while removing merged files: %v\n", err)
		}
	}

	if flags.upload {
		logInfo.Println("Uploading combined file to Kindle...")
		if err := sendComicBookToKindle(flags.addr, combined); err != nil {
			logError.Fatalf("while sending to Kindle: %v\n", err)
		}
	}

	logInfo.Println("Done!")
}

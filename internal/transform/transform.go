package transform

import (
	"bytes"
	"fmt"
	"runtime"

	"github.com/abbit/m4k/internal/comicbook"
	"github.com/disintegration/imaging"
	"golang.org/x/sync/errgroup"
)

const (
	defaultJpegQuality = 75
)

var (
	ErrZeroWidthHeight = fmt.Errorf("width and height must be greater than 0")
	ErrNoEncoding      = fmt.Errorf("encoding must be specified")
)

type Options struct {
	// required

	Width, Height int
	Encoding      string

	// optional

	Rotate      bool
	JpegQuality int
	// Callback to be called after page transformation
	Callback func()
}

func TransformImage(data []byte, opts *Options) ([]byte, error) {
	width, height := opts.Width, opts.Height

	if width <= 0 || height <= 0 {
		return nil, ErrZeroWidthHeight
	}

	if len(opts.Encoding) == 0 {
		return nil, ErrNoEncoding
	}

	// decode image
	buf := bytes.NewBuffer(data)
	img, err := imaging.Decode(buf)
	if err != nil {
		return nil, fmt.Errorf("while decoding image: %w", err)
	}

	// transform image
	twopages := false
	// check if image is in landscape mode
	if imgsize := img.Bounds().Size(); imgsize.X > imgsize.Y {
		if opts.Rotate {
			img = imaging.Rotate90(img)
		} else {
			twopages = true
		}
	}
	if twopages {
		width *= 2
	}
	img = imaging.Resize(img, width, height, imaging.Lanczos)
	img = imaging.Grayscale(img)

	// encode image
	buf.Reset()
	switch opts.Encoding {
	case "png":
		err = imaging.Encode(buf, img, imaging.PNG)
	case "jpeg", "jpg":
		jpegQuality := opts.JpegQuality
		if jpegQuality == 0 {
			jpegQuality = defaultJpegQuality
		}
		err = imaging.Encode(buf, img, imaging.JPEG, imaging.JPEGQuality(jpegQuality))
	default:
		err = fmt.Errorf("unsupported encoding: %s", opts.Encoding)
	}
	if err != nil {
		return nil, fmt.Errorf("while encoding image: %w", err)
	}

	// call callback
	if opts.Callback != nil {
		opts.Callback()
	}

	return buf.Bytes(), nil
}

func TransformPage(p *comicbook.Page, opts *Options) error {
	// transform page image
	transformed, err := TransformImage(p.Data, opts)
	if err != nil {
		return fmt.Errorf("while transforming image: %w", err)
	}

	// update page
	p.Data = transformed
	p.Extension = "." + opts.Encoding

	return nil
}

func TransformComicBook(cb *comicbook.ComicBook, opts *Options) error {
	var eg errgroup.Group
	// limit number of goroutines for image processing to cpu cores - 1
	// to leave some space for other tasks
	eg.SetLimit(max(1, runtime.NumCPU()-1))

	for _, p := range cb.Pages {
		p := p
		eg.Go(func() error {
			if err := TransformPage(p, opts); err != nil {
				return fmt.Errorf("while transforming page %s: %w", p.Filepath(), err)
			}
			return nil
		})
	}

	return eg.Wait()
}

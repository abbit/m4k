package transform

import (
	"bytes"
	"fmt"
	"runtime"

	"github.com/abbit/m4k/internal/comicbook"
	"github.com/disintegration/imaging"
	"golang.org/x/sync/errgroup"
)

var ErrZeroWidthHeight = fmt.Errorf("width and height must be greater than 0")

type Options struct {
	Rotate        bool
	Width, Height int
	Callback      func()
}

func TransformPage(p *comicbook.Page, opts Options) error {
	width, height := opts.Width, opts.Height

	if width <= 0 || height <= 0 {
		return ErrZeroWidthHeight
	}

	// decode image
	buf := bytes.NewBuffer(p.Data)
	img, err := imaging.Decode(buf)
	if err != nil {
		return fmt.Errorf("while decoding image: %w", err)
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
	if err = imaging.Encode(buf, img, imaging.JPEG, imaging.JPEGQuality(75)); err != nil {
		return fmt.Errorf("while encoding image: %w", err)
	}

	// update page
	p.Data = buf.Bytes()
	p.Extension = ".jpg"

	return nil
}

func TransformComicBook(cb *comicbook.ComicBook, opts Options) error {
	g := &errgroup.Group{}
	// limit number of goroutines for image processing to cpu cores - 1
	// to leave some space for other tasks
	g.SetLimit(runtime.NumCPU() - 1)

	for _, p := range cb.Pages {
		p := p
		g.Go(func() error {
			if err := TransformPage(p, opts); err != nil {
				return fmt.Errorf("while transforming page %s: %w", p.Filepath(), err)
			}
			if opts.Callback != nil {
				opts.Callback()
			}
			return nil
		})
	}

	return g.Wait()
}

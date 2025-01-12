package log

import (
	"log"
	"os"
)

// TODO: remove in favor of slog
var (
	Error *log.Logger = log.New(os.Stderr, "Error: ", 0)
	Info  *log.Logger = log.New(os.Stdout, "", 0)
)

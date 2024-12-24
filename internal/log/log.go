package log

import (
	"log"
	"os"
)

var (
	Error *log.Logger = log.New(os.Stderr, "Error: ", 0)
	Info  *log.Logger = log.New(os.Stdout, "", 0)
)

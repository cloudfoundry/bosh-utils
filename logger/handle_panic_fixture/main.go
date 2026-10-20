package main

import (
	"flag"
	"os"

	"github.com/cloudfoundry/bosh-utils/logger"
)

func main() {
	var (
		async   = flag.Bool("async", false, "use the async writer logger")
		noPanic = flag.Bool("no-panic", false, "defer HandlePanic without panicking")
	)
	flag.Parse()

	newLogger := logger.NewWriterLogger
	if *async {
		newLogger = logger.NewAsyncWriterLogger
	}
	l := newLogger(logger.LevelError, os.Stderr)

	defer l.HandlePanic("test-tag")

	if *noPanic {
		return
	}

	panic("boom")
}

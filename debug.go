package main

import (
	"fmt"
	"os"
	"time"
)

var debugMode = false

func dbg(format string, args ...any) {
	if !debugMode {
		return
	}
	ts := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "[DBG %s] %s\n", ts, msg)
}

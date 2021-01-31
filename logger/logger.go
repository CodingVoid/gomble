package logger

import (
	"fmt"
	"os"
)

type Loglevel int

var loglevel Loglevel

const (
    TRACE Loglevel = iota // 0
	DEBUG Loglevel = iota // 1
	INFO  Loglevel = iota // 2
	WARN  Loglevel = iota // 3
	ERROR Loglevel = iota // 4
	FATAL Loglevel = iota // 5
)

// writes data to filename
func Tracefile(filename string, data []byte) error {
    if loglevel <= TRACE {
        file, err := os.OpenFile(filename, os.O_TRUNC | os.O_CREATE | os.O_WRONLY, 0644)
        if err != nil {
            return err
        }
        _, err = file.Write(data)
        return err
    }
    return nil
}

func Debugf(format string, a ...interface{}) {
    if loglevel <= DEBUG {
        fmt.Printf("DEBUG: "+format, a...)
    }
}

func Infof(format string, a ...interface{}) {
    if loglevel <= INFO {
        fmt.Printf("INFO: "+format, a...)
    }
}

func Warnf(format string, a ...interface{}) {
    if loglevel <= WARN {
        fmt.Printf("WARN: "+format, a...)
    }
}

func Errorf(format string, a ...interface{}) {
    if loglevel <= ERROR {
        fmt.Printf("ERROR: "+format, a...)
    }
}

func Fatalf(format string, a ...interface{}) {
    if loglevel <= FATAL {
        fmt.Printf("FATAL: "+format, a...)
        os.Exit(-1)
    }
}

func SetLogLevel(level Loglevel) {
    loglevel = level
}


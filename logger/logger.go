package logger

import "log"
import "os"
import "fmt"

type Loglevel int

var loglevel Loglevel

const (
	DEBUG Loglevel = iota // 0
	INFO  Loglevel = iota // 1
	WARN  Loglevel = iota // 2
	ERROR Loglevel = iota // 3
	FATAL Loglevel = iota // 4
)

func Debug(str string) {
	if loglevel <= DEBUG {
		fmt.Print("DEBUG: " + str)
	}
}

func Debugf(format string, a ...interface{}) {
	if loglevel <= DEBUG {
		fmt.Printf("DEBUG: " + format, a...)
	}
}

func Info(str string) {
	if loglevel <= INFO {
		fmt.Print("INFO: " + str)
	}
}

func Infof(format string, a ...interface{}) {
	if loglevel <= INFO {
		fmt.Printf("INFO: " + format, a...)
	}
}

func Warn(str string) {
	if loglevel <= WARN {
		fmt.Print("WARN: " + str)
	}
}

func Warnf(format string, a ...interface{}) {
	if loglevel <= WARN {
		fmt.Printf("WARN: " + format, a...)
	}
}

func Error(str string) {
	if loglevel <= ERROR {
		fmt.Print("ERROR: " + str)
	}
}

func Errorf(format string, a ...interface{}) {
	if loglevel <= ERROR {
		fmt.Printf("ERROR: " + format, a...)
	}
}

func Fatal(str string) {
	if loglevel <= FATAL {
		fmt.Print("FATAL: " + str)
		os.Exit(-1)
	}
}

func Fatalf(format string, a ...interface{}) {
	if loglevel <= FATAL {
		fmt.Printf("FATAL: " + format, a...)
		os.Exit(-1)
	}
}

func SetLogLevel(level Loglevel) {
	loglevel = level
}

func Init() {
	log.SetFlags(0)
}

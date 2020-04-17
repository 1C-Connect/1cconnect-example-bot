package logger

import (
	"fmt"
	"log"
)

var (
	isDebug = false
)

func InitLogger(debug bool) {
	isDebug = debug

	log.SetPrefix("[APP] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)
}

func Info(v ...interface{}) {
	log.Print("[INFO] ", fmt.Sprintln(v...))
}

func Warning(v ...interface{}) {
	log.Print("[WARNING] ", fmt.Sprintln(v...))
}

func Debug(v ...interface{}) {
	if isDebug {
		log.Print("[DEBUG] ", fmt.Sprintln(v...))
	}
}

package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
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
		message := new(bytes.Buffer)

		for _, str := range v {
			v, ok := str.(string)
			if ok {
				_, _ = fmt.Fprintf(message, "%s ", v)
			} else {
				s, _ := json.MarshalIndent(str, "", " ")
				_, _ = fmt.Fprintf(message, "%s ", string(s))
			}
		}

		log.Print("[DEBUG] ", message)
	}
}

func Crit(v ...interface{}) {
	log.Printf("Critical error: %s", v)
	time.Sleep(5 * time.Second)
	os.Exit(1)
}

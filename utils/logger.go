package utils

import "log"

type Logger string

func (l *Logger) Println(v ...interface{}) {
	log.Printf("[bucket=%s] %v\n", *l, v)
}

func (l *Logger) Printf(format string, v ...interface{}) {

}

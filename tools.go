package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

var layout = "2006-01-02T15:04:05"

func getTimeMoscow(st string) time.Time {
	location, _ := time.LoadLocation("Europe/Moscow")
	tz, e := time.Parse(time.RFC3339, st)
	if e != nil {
		return time.Time{}
	}
	p := tz.In(location)
	return p
}

func SaveStack() {
	if p := recover(); p != nil {
		var buf [4096]byte
		n := runtime.Stack(buf[:], false)
		file, err := os.OpenFile(string(FileLog), os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		defer file.Close()
		if err != nil {
			fmt.Println("Ошибка записи stack log", err)
			return
		}
		fmt.Fprintln(file, fmt.Sprintf("Fatal Error %v", p))
		fmt.Fprintf(file, "%v  ", string(buf[:n]))
	}

}

func GetConformity(conf string) int {
	s := strings.ToLower(conf)
	switch {
	case strings.Index(s, "открыт") != -1:
		return 5
	case strings.Index(s, "аукцион") != -1:
		return 1
	case strings.Index(s, "котиров") != -1:
		return 2
	case strings.Index(s, "предложен") != -1:
		return 3
	case strings.Index(s, "единств") != -1:
		return 4
	default:
		return 6
	}

}

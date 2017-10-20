package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

func getTime() {
	location, _ := time.LoadLocation("Europe/Moscow")
	t := "2012-10-19T10:30:56+06:00"
	tz, _ := time.Parse(time.RFC3339, t)
	p := tz.In(location)
	fmt.Println(p)
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

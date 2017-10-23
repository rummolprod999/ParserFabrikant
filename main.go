package main

import "fmt"

func init() {
	CreateLogFile()
	GetSetting()

}

var Addtender = 0

func main() {
	defer SaveStack()
	Logging("Start parsing")
	Parser()
	Logging("End parsing")
	Logging(fmt.Sprintf("Добавили тендеров %d", Addtender))
}

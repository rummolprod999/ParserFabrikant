package main

import (
	"fmt"
)

func init() {
	CreateLogFile()
	GetSetting()

}

func main() {
	Logging("Start parsing")
	Parser()


	Logging("End parsing")
}

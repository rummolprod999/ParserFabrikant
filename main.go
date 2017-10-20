package main

import (

)

func init() {
	CreateLogFile()
	GetSetting()

}

func main() {
	defer SaveStack()
	Logging("Start parsing")
	Parser()
	Logging("End parsing")
}

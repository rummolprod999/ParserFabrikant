package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

var Prefix string
var UserDb string
var PassDb string
var Server string
var Port int
var User string
var Pass string

type Settings struct {
	Prefix string `xml:"prefix"`
	UserDb string `xml:"userdb"`
	PassDb string `xml:"passdb"`
	Server string `xml:"server"`
	Port   int    `xml:"port"`
	User   string `xml:"user"`
	Pass   string `xml:"pass"`
}

func GetSetting() {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	xmlFile, err := os.Open(fmt.Sprintf("%s/set.xml", dir))
	defer xmlFile.Close()
	if err != nil {
		Logging(err)
	}
	byteValue, _ := ioutil.ReadAll(xmlFile)
	var settings Settings
	e := xml.Unmarshal(byteValue, &settings)
	if e != nil {
		Logging(e)
	}
	Prefix = settings.Prefix
	UserDb = settings.UserDb
	PassDb = settings.PassDb
	Server = settings.Server
	Port = settings.Port
	User = settings.User
	Pass = settings.Pass
}

package main

import "fmt"

func Parser() {
	for i := 1; i <= 50; i++ {
		ParserPage(i)
	}
}

func ParserPage(i int) {
	defer func() {
		if p := recover(); p != nil {
			Logging(p)
		}
	}()

	UrlXml = fmt.Sprintf("https://www.fabrikant.ru/trade-list/index.php?xml&method=GetTradeList&status=actual&page=%v&perpage=50", i)
	r := DownloadPage(UrlXml)
	if r != "" {
		ParsingString(r)
	}
}

func ParsingString(s string) {
	fmt.Println(s)
}

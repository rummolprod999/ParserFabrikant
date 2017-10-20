package main

import "fmt"

func Parser(){
	for i := 1; i < 50; i++ {
		s := fmt.Sprintf("https://www.fabrikant.ru/trade-list/index.php?xml&method=GetTradeList&status=actual&page=%v&perpage=50", i)
		r := GetPage(s)
		fmt.Println(r)
		fmt.Println(s)
	}
}

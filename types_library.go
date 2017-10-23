package main

type FileProtocols struct {
	TradeList []Trade `xml:"TradeList>Trade"`
}

type Trade struct {
	TradeId          string `xml:"TradeId"`
	PublicationDate  string `xml:"PublicationDate"`
	UnsealDate       string `xml:"UnsealDate"`
	FinishDate       string `xml:"FinishDate"`
	TradeUri         string `xml:"TradeUri"`
	TradeType        string `xml:"TradeType"`
	Title            string `xml:"Title"`
	CommonName       string `xml:"CommonName"`
	DocumentationUrl string `xml:"DocumentationUrl"`
	Lots             []Lot  `xml:"Lot"`
	Currency         string `xml:"Currency>ISO"`
	Customer         string `xml:"Customer"`
	Organizer
	ContactName
}

type Organizer struct {
	OrganizerName string `xml:"Organizer>Name"`
	OrganizerINN  string `xml:"Organizer>INN"`
	OrganizerKPP  string `xml:"Organizer>KPP"`
	OrganizerOGRN string `xml:"Organizer>OGRN"`
}

type ContactName struct {
	FirstName  string `xml:"ContactName>FirstName"`
	MiddleName string `xml:"ContactName>MiddleName"`
	LastName   string `xml:"ContactName>LastName"`
	Phone      string `xml:"ContactName>Phone"`
	Email      string `xml:"ContactName>Email"`
}

type Lot struct {
	MaxPrice            string `xml:"PriceInfo>Price"`
	ContractSubject     string `xml:"ContractSubject"`
	ContractSubjectText string `xml:"ContractSubjectText"`
	Description         string `xml:"Description"`
	Quantity            string `xml:"Quantity"`
	MeasureUnit         string `xml:"MeasureUnit"`
}

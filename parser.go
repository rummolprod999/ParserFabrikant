package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"strings"
	"time"
)

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
	//fmt.Println(s)
	var FileProt FileProtocols
	if err := xml.Unmarshal([]byte(s), &FileProt); err != nil {
		Logging("Ошибка при парсинге строки", err)
		return
	}
	var Dsn = fmt.Sprintf("%s:%s@/%s?charset=utf8&parseTime=true&readTimeout=60m&maxAllowedPacket=0&timeout=60m&writeTimeout=60m&autocommit=true", UserDb, PassDb, DbName)
	db, err := sql.Open("mysql", Dsn)
	defer db.Close()
	//db.SetMaxOpenConns(2)
	db.SetConnMaxLifetime(time.Second * 3600)
	if err != nil {
		Logging("Ошибка подключения к БД", err)
	}
	for _, t := range FileProt.TradeList {
		e := ParsingTrade(t, db)
		if e != nil {
			Logging("Ошибка парсера в протоколе", e)
			continue
		}
	}
}

func ParsingTrade(t Trade, db *sql.DB) error {
	TradeId := t.TradeId
	PublicationDate := getTimeMoscow(t.PublicationDate)
	DateUpdated := PublicationDate
	fmt.Println(DateUpdated)
	IdXml := TradeId
	Version := 0
	stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_tender FROM %stender WHERE purchase_number = ? AND date_version = ?", Prefix))
	res, err := stmt.Query(TradeId, DateUpdated)
	stmt.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	if res.Next() {
		Logging("Такой тендер уже есть", TradeId)
		res.Close()
		return nil
	}
	res.Close()
	var cancelStatus = 0
	if TradeId != "" {
		stmt, err := db.Prepare(fmt.Sprintf("SELECT id_tender, date_version FROM %stender WHERE purchase_number = ? AND cancel=0", Prefix))
		rows, err := stmt.Query(TradeId)
		stmt.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		for rows.Next() {
			var idTender int
			var dateVersion time.Time
			err = rows.Scan(&idTender, &dateVersion)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			//fmt.Println(DateUpdated.Sub(dateVersion))
			if dateVersion.Sub(DateUpdated) <= 0 {
				stmtupd, _ := db.Prepare(fmt.Sprintf("UPDATE %stender SET cancel=1 WHERE id_tender = ?", Prefix))
				_, err = stmtupd.Exec(idTender)
				stmtupd.Close()

			} else {
				cancelStatus = 1
			}

		}
		rows.Close()
		//fmt.Println(cancelStatus)
	}
	Href := t.TradeUri
	Title := t.Title
	CommonName := t.CommonName
	PurchaseObjectInfo := strings.TrimSpace(fmt.Sprintf("%s %s", Title, CommonName))
	NoticeVersion := ""
	PrintForm := Href
	IdOrganizer := 0
	if t.OrganizerINN != "" {
		stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_organizer FROM %sorganizer WHERE inn = ? AND kpp = ?", Prefix))
		rows, err := stmt.Query(t.OrganizerINN, t.OrganizerKPP)
		stmt.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		if rows.Next() {
			err = rows.Scan(&IdOrganizer)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			rows.Close()
		} else {
			rows.Close()
			ContactPerson := strings.TrimSpace(fmt.Sprintf("%s %s %s", t.LastName, t.FirstName, t.MiddleName))
			stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %sorganizer SET full_name = ?, inn = ?, kpp = ?, contact_email = ?, contact_phone = ?, contact_person = ?", Prefix))
			res, err := stmt.Exec(t.OrganizerName, t.OrganizerINN, t.OrganizerKPP, t.Email, t.Phone, ContactPerson)
			stmt.Close()
			if err != nil {
				Logging("Ошибка вставки организатора", err)
				return err
			}
			id, err := res.LastInsertId()
			IdOrganizer = int(id)
		}
	}

	IdPlacingWay := 0
	if t.TradeType != "" {
		stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_placing_way FROM %splacing_way WHERE name = ? LIMIT 1", Prefix))
		rows, err := stmt.Query(t.TradeType)
		stmt.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		if rows.Next() {
			err = rows.Scan(&IdPlacingWay)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			rows.Close()
		} else {
			rows.Close()
			conf := GetConformity(t.TradeType)
			stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %splacing_way SET name = ?, conformity = ?", Prefix))
			res, err := stmt.Exec(t.TradeType, conf)
			stmt.Close()
			if err != nil {
				Logging("Ошибка вставки placing way", err)
				return err
			}
			id, err := res.LastInsertId()
			IdPlacingWay = int(id)

		}
	}

	IdEtp := 0
	etpName := "ЭТП «Фабрикант»"
	etpUrl := "https://www.fabrikant.ru"
	if true {
		stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_etp FROM %setp WHERE name = ? AND url = ? LIMIT 1", Prefix))
		rows, err := stmt.Query(etpName, etpUrl)
		stmt.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		if rows.Next() {
			err = rows.Scan(&IdEtp)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			rows.Close()
		} else {
			rows.Close()
			stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %setp SET name = ?, url = ?, conf=0", Prefix))
			res, err := stmt.Exec(etpName, etpUrl)
			stmt.Close()
			if err != nil {
				Logging("Ошибка вставки etp", err)
				return err
			}
			id, err := res.LastInsertId()
			IdEtp = int(id)
		}
	}
	var EndDate = time.Time{}
	EndDate = getTimeMoscow(t.UnsealDate)
	FinishDate := getTimeMoscow(t.FinishDate)
	if EndDate == (time.Time{}) {
		EndDate = FinishDate
	}
	typeFz := 2
	idTender := 0
	stmtt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %stender SET id_region = 0, id_xml = ?, purchase_number = ?, doc_publish_date = ?, href = ?, purchase_object_info = ?, type_fz = ?, id_organizer = ?, id_placing_way = ?, id_etp = ?, end_date = ?, cancel = ?, date_version = ?, num_version = ?, notice_version = ?, xml = ?, print_form = ?", Prefix))
	rest, err := stmtt.Exec(IdXml, TradeId, PublicationDate, Href, PurchaseObjectInfo, typeFz, IdOrganizer, IdPlacingWay, IdEtp, EndDate, cancelStatus, DateUpdated, Version, NoticeVersion, UrlXml, PrintForm)
	stmtt.Close()
	if err != nil {
		Logging("Ошибка вставки tender", err)
		return err
	}
	idt, err := rest.LastInsertId()
	idTender = int(idt)
	fmt.Println(idTender)
	Addtender++
	return nil
}

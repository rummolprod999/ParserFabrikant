package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func Parser() {
	for i := Count; i >= 1; i-- {
		ParserPage(i)
		time.Sleep(time.Second * 10)
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
	} else {
		Logging("Получили пустую строку", UrlXml)
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
	if len(FileProt.TradeList) == 0 {
		Logging("Нет процедур в файле", UrlXml, s)
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
	//fmt.Println(DateUpdated)
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
		//Logging("Такой тендер уже есть", TradeId)
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
	Addtender++
	if t.DocumentationUrl != "" {
		attachName := fmt.Sprintf("Ссылка на страницу с документацией торговой процедуры: %s", PurchaseObjectInfo)
		stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %sattachment SET id_tender = ?, file_name = ?, url = ?", Prefix))
		_, err := stmt.Exec(idTender, attachName, t.DocumentationUrl)
		stmt.Close()
		if err != nil {
			Logging("Ошибка вставки attachment", err)
			return err
		}

	}
	var LotNumber = 1
	for _, lot := range t.Lots {
		idLot := 0
		MaxPrice := lot.MaxPrice
		//fmt.Println(MaxPrice)
		stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %slot SET id_tender = ?, lot_number = ?, max_price = ?, currency = ?", Prefix))
		res, err := stmt.Exec(idTender, LotNumber, MaxPrice, t.Currency)
		stmt.Close()
		if err != nil {
			Logging("Ошибка вставки lot", err)
			return err
		}
		id, _ := res.LastInsertId()
		idLot = int(id)

		idCustomer := 0
		if t.Customer != "" {
			stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_customer FROM %scustomer WHERE full_name LIKE ? LIMIT 1", Prefix))
			rows, err := stmt.Query(t.Customer)
			stmt.Close()
			if err != nil {
				Logging("Ошибка выполения запроса", err)
				return err
			}
			if rows.Next() {
				err = rows.Scan(&idCustomer)
				if err != nil {
					Logging("Ошибка чтения результата запроса", err)
					return err
				}
				rows.Close()
			} else {
				rows.Close()
				out, err := exec.Command("uuidgen").Output()
				if err != nil {
					Logging("Ошибка генерации UUID", err)
					return err
				}
				stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %scustomer SET full_name = ?, is223=1, reg_num = ?", Prefix))
				res, err := stmt.Exec(t.Customer, out)
				stmt.Close()
				if err != nil {
					Logging("Ошибка вставки организатора", err)
					return err
				}
				id, err := res.LastInsertId()
				idCustomer = int(id)
			}
		}
		//fmt.Println(idLot, idCustomer)
		okpd2Code := lot.ContractSubject
		okpdName := lot.ContractSubjectText
		Name := strings.TrimSpace(fmt.Sprintf("%s %s", lot.ContractSubjectText, lot.Description))
		okpd2GroupCode, okpd2GroupLevel1Code := GetOkpd(okpd2Code)
		stmtr, _ := db.Prepare(fmt.Sprintf("INSERT INTO %spurchase_object SET id_lot = ?, id_customer = ?, okpd2_code = ?, okpd2_group_code = ?, okpd2_group_level1_code = ?, okpd_name = ?, name = ?, quantity_value = ?, customer_quantity_value = ?, okei = ?, price = ?", Prefix))
		_, errr := stmtr.Exec(idLot, idCustomer, okpd2Code, okpd2GroupCode, okpd2GroupLevel1Code, okpdName, Name, lot.Quantity, lot.Quantity, lot.MeasureUnit, lot.MaxPrice)
		stmtr.Close()
		if errr != nil {
			Logging("Ошибка вставки purchase_object", errr)
			return err
		}
	}
	e := TenderKwords(db, idTender)
	if e != nil {
		Logging("Ошибка обработки TenderKwords", e)
	}

	e1 := AddVerNumber(db, TradeId)
	if e1 != nil {
		Logging("Ошибка обработки AddVerNumber", e1)
	}
	return nil
}

func TenderKwords(db *sql.DB, idTender int) error {
	resString := ""
	stmt, _ := db.Prepare(fmt.Sprintf("SELECT DISTINCT po.name, po.okpd_name FROM %spurchase_object AS po LEFT JOIN %slot AS l ON l.id_lot = po.id_lot WHERE l.id_tender = ?", Prefix, Prefix))
	rows, err := stmt.Query(idTender)
	stmt.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows.Next() {
		var name sql.NullString
		var okpdName sql.NullString
		err = rows.Scan(&name, &okpdName)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		if name.Valid {
			resString = fmt.Sprintf("%s %s ", resString, name.String)
		}
		if okpdName.Valid {
			resString = fmt.Sprintf("%s %s ", resString, okpdName.String)
		}
	}
	rows.Close()
	stmt1, _ := db.Prepare(fmt.Sprintf("SELECT DISTINCT file_name FROM %sattachment WHERE id_tender = ?", Prefix))
	rows1, err := stmt1.Query(idTender)
	stmt1.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows1.Next() {
		var attName sql.NullString
		err = rows1.Scan(&attName)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		if attName.Valid {
			resString = fmt.Sprintf("%s %s ", resString, attName.String)
		}
	}
	rows1.Close()
	idOrg := 0
	stmt2, _ := db.Prepare(fmt.Sprintf("SELECT purchase_object_info, id_organizer FROM %stender WHERE id_tender = ?", Prefix))
	rows2, err := stmt2.Query(idTender)
	stmt2.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows2.Next() {
		var idOrgNull sql.NullInt64
		var purOb sql.NullString
		err = rows2.Scan(&purOb, &idOrgNull)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		if idOrgNull.Valid {
			idOrg = int(idOrgNull.Int64)
		}
		if purOb.Valid {
			resString = fmt.Sprintf("%s %s ", resString, purOb.String)
		}

	}
	rows2.Close()
	if idOrg != 0 {
		stmt3, _ := db.Prepare(fmt.Sprintf("SELECT full_name, inn FROM %sorganizer WHERE id_organizer = ?", Prefix))
		rows3, err := stmt3.Query(idOrg)
		stmt3.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		for rows3.Next() {
			var innOrg sql.NullString
			var nameOrg sql.NullString
			err = rows3.Scan(&nameOrg, &innOrg)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			if innOrg.Valid {

				resString = fmt.Sprintf("%s %s ", resString, innOrg.String)
			}
			if nameOrg.Valid {
				resString = fmt.Sprintf("%s %s ", resString, nameOrg.String)
			}

		}
		rows3.Close()
	}
	stmt4, _ := db.Prepare(fmt.Sprintf("SELECT DISTINCT cus.inn, cus.full_name FROM %scustomer AS cus LEFT JOIN %spurchase_object AS po ON cus.id_customer = po.id_customer LEFT JOIN %slot AS l ON l.id_lot = po.id_lot WHERE l.id_tender = ?", Prefix, Prefix, Prefix))
	rows4, err := stmt4.Query(idTender)
	stmt4.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows4.Next() {
		var innC sql.NullString
		var fullNameC sql.NullString
		err = rows4.Scan(&innC, &fullNameC)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		if innC.Valid {

			resString = fmt.Sprintf("%s %s ", resString, innC.String)
		}
		if fullNameC.Valid {
			resString = fmt.Sprintf("%s %s ", resString, fullNameC.String)
		}
	}
	rows4.Close()
	re := regexp.MustCompile(`\s+`)
	resString = re.ReplaceAllString(resString, " ")
	stmtr, _ := db.Prepare(fmt.Sprintf("UPDATE %stender SET tender_kwords = ? WHERE id_tender = ?", Prefix))
	_, errr := stmtr.Exec(resString, idTender)
	stmtr.Close()
	if errr != nil {
		Logging("Ошибка вставки TenderKwords", errr)
		return err
	}
	return nil
}

func AddVerNumber(db *sql.DB, RegistryNumber string) error {
	verNum := 1
	mapTenders := make(map[int]int)
	stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_tender FROM %stender WHERE purchase_number = ? ORDER BY UNIX_TIMESTAMP(date_version) ASC", Prefix))
	rows, err := stmt.Query(RegistryNumber)
	stmt.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows.Next() {
		var rNum int
		err = rows.Scan(&rNum)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		mapTenders[verNum] = rNum
		verNum++
	}
	rows.Close()
	for vn, idt := range mapTenders {
		stmtr, _ := db.Prepare(fmt.Sprintf("UPDATE %stender SET num_version = ? WHERE id_tender = ?", Prefix))
		_, errr := stmtr.Exec(vn, idt)
		stmtr.Close()
		if errr != nil {
			Logging("Ошибка вставки NumVersion", errr)
			return err
		}
	}

	return nil
}

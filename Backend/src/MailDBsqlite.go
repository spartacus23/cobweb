package main

import (
	"time"
	"log"
	"strconv"
	"code.google.com/p/gosqlite/sqlite"
	)

type MailDB struct{
        In 		chan *Event
        Out 		chan *Event
        NextID		uint64
}

func (p *MailDB) PushEvent(event *Event){
        p.In <- event
}

func (p *MailDB) PullEvent()*Event{
        return <-p.Out
}

func NewMailDB(backlog int)*MailDB{
	erg := new(MailDB)
	erg.In = make(chan *Event,backlog)
	erg.Out= make(chan *Event,backlog)
	go erg.backend()
	return erg
}

type Mail struct{
	Sender	string
	Subject	string
	Text	string
	Id	int
	Time	int
}

func NewMail(sender,subject,text string) *Mail {
	mail := new(Mail)
	mail.Sender = sender
	mail.Subject = subject
	mail.Text = text
	return mail
}

func (mdb *MailDB) SaveMail(mail *Mail){
	conn, err := sqlite.Open("maildb.db")
	if err != nil {
		log.Print("Unable to open the database: ", err)
		return
	}
	defer conn.Close()
	insertsql := "INSERT INTO mails (sender,subject,text,time) VALUES (?,?,?,?) ;"
	
	stmt,err := conn.Prepare(insertsql)
	if err!=nil {
		log.Print("maildb insert fail @prepare: ",err)
		return
	}
	err = stmt.Exec(mail.Sender,mail.Subject,mail.Text,time.Now().Unix())
	if err!=nil{
		log.Print("maildb insert fail @exec: ",err)
		return
	}
	stmt.Next()
	err = stmt.Finalize()
	if err != nil {
		log.Print("finalize fail")
		return
	}
	log.Print("New mail from ",mail.Sender)
}

func (mdb *MailDB) GetMail(id int) *Mail {
	conn, err := sqlite.Open("maildb.db")
	if err != nil {
		log.Print("Unable to open the database: ", err)
		return nil
	}
	selectstmt,err := conn.Prepare("SELECT id,sender,subject,text,time FROM mails WHERE id="+strconv.Itoa(id)+";")
	if err != nil {
		log.Print("Error preparing the selectstatement")
		return nil
	}
	err = selectstmt.Exec()
	if err!=nil {
		log.Print("Error execution of select stmt")
		return nil
	}
	if selectstmt.Next() {
		var mail Mail
		err = selectstmt.Scan(&mail.Id, &mail.Sender, &mail.Subject, &mail.Text, &mail.Time)
		if err != nil {
			log.Print("Error while scanning row")
			return nil
		}
		return &mail
	}
	return nil
}

func (mdb *MailDB) GetMailList(maxcount int) []*Mail {
	conn, err := sqlite.Open("maildb.db")
	if err != nil {
		log.Print("Unable to open the database: ", err)
		return nil
	}
	s := "SELECT id,sender,subject,time FROM mails ORDER BY id ASC LIMIT "+strconv.Itoa(maxcount)+";"
	selectstmt,err := conn.Prepare(s)
	if err != nil {
		log.Print("Error preparing the selectstatement: ",s)
		return nil
	}
	err = selectstmt.Exec()
	if err!=nil {
		log.Print("Error execution of select stmt")
		return nil
	}
	erglist := make([]*Mail,0)
	for{
		if selectstmt.Next() {
			var mail Mail
			err = selectstmt.Scan(&mail.Id, &mail.Sender, &mail.Subject, &mail.Time)
			if err != nil {
				log.Print("Error while scanning row")
				return erglist
			}
			erglist = append(erglist,&mail)
		}else{
			break
		}
	}
	return erglist
}

func (mdb *MailDB) DeleteMail(id int) bool{
	conn, err := sqlite.Open("maildb.db")
	if err != nil {
		log.Print("Unable to open the database: ", err)
		return false
	}
	defer conn.Close()
	s := "DELETE FROM mails WHERE id=?;"
	stmt,err := conn.Prepare(s)
	if err != nil {
		log.Print("Error preparing the selectstatement: ",s)
		return false
	}
	err = stmt.Exec(id)
	if err!=nil {
		log.Print("Error execution of select stmt")
		return false
	}
	stmt.Next()
	stmt.Finalize()
	return true
}

type GetMailPayload struct {
	Return		chan *Mail
	Id		int
}
func NewGetMailPayload(id int) *GetMailPayload {
	return &GetMailPayload{make(chan *Mail),id}
}

type GetMailListPayload struct {
	Return 		chan []*Mail
	Max		int
}
func NewGetMailListPayload(max int) *GetMailListPayload {
	return &GetMailListPayload{make(chan []*Mail),max}
}

func (mdb *MailDB) backend(){
	//init db
	conn, err := sqlite.Open("maildb.db")
	if err != nil {
		log.Print("Unable to open the database: ", err)
		return
	}
	conn.Exec("CREATE TABLE mails(id INTEGER PRIMARY KEY AUTOINCREMENT, sender VARCHAR, subject VARCHAR, text VARCHAR,time INTEGER);")
	conn.Close()
	for{
		event := <-mdb.In
		if event.Topic == "SaveMail" {
			mdb.SaveMail(event.Payload.(*Mail))
		}else if event.Topic == "GetMailList" {
			pay := event.Payload.(*GetMailListPayload)
			list := mdb.GetMailList(pay.Max)
			pay.Return <- list
		}else if event.Topic == "GetMail" {
			pay := event.Payload.(*GetMailPayload)
			pay.Return <- mdb.GetMail(pay.Id)
		}else if event.Topic == "DeleteMail" {
			pay := event.Payload.(*GetMailPayload)
			ok := mdb.DeleteMail(pay.Id)
			if ok {
				pay.Return <- &Mail{Id:1}
			}else{
				pay.Return <- &Mail{Id:0}
			}
		}
	}
}

package main

import (
	"net"
	"bufio"
	"bytes"
	"log"
	)

type MailServer struct{
        In 		chan *Event
        Out 		chan *Event
}
func (p *MailServer) PushEvent(event *Event){
        p.In <- event
}
func (p *MailServer) PullEvent()*Event{
        return <-p.Out
}

func NewMailServer(backlog int)*MailServer{
	erg := new(MailServer)
	erg.In = make(chan *Event,backlog)
	erg.Out= make(chan *Event,backlog)
	go erg.backend()
	return erg
}

func (es *MailServer) handleClient(conn net.Conn,id string){
	defer conn.Close()
	reader := bufio.NewReader(conn)
	buff := new(bytes.Buffer)
	//read subject (first line)
	for{
		part,prefix,err := reader.ReadLine()
		if err!=nil {
			log.Print("MailServer readline err: ",err)
			return
		}
		buff.Write(part)
		if !prefix {
			break
		}
	}
	subject := buff.String()
	buff.Reset()
	//read message
	tmpbuff := make([]byte,1024)
	for{
		n,err := reader.Read(tmpbuff)
		if err != nil {
			break
		}
		buff.Write(tmpbuff[:n])
	}
	message := buff.String()
	log.Print("Accepted mail, will save now")
	es.Out <- NewEvent("SaveMail",NewMail(id,subject,message))
}

func (es *MailServer) backend(){
	for{
		event := <-es.In
		if event.Topic == "MailServerRequest" {
			log.Print("New Mailserver request!")
			pay:=event.Payload.(*AuthenticatedLocalClientPayload)
			go es.handleClient(pay.Conn,pay.Hash)
		}
	}
}

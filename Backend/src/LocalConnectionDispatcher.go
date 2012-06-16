package main

import (
	"strconv"
	"log"
	)

type LocalConnectionDispatcher struct{
        In 		chan *Event
        Out 		chan *Event
}
func (p *LocalConnectionDispatcher) PushEvent(event *Event){
        p.In <- event
}
func (p *LocalConnectionDispatcher) PullEvent()*Event{
        return <-p.Out
}

func NewLocalConnectionDispatcher(backlog int)*LocalConnectionDispatcher{
	erg := new(LocalConnectionDispatcher)
	erg.In = make(chan *Event,backlog)
	erg.Out= make(chan *Event,backlog)
	go erg.backend()
	return erg
}

func (p *LocalConnectionDispatcher) backend(){
	for{
		event := <-p.In
		if event.Topic == "AuthenticatedLocalClient" {
			pay:=event.Payload.(*AuthenticatedLocalClientPayload)
			log.Print("New Authenticated Local Client to port: "+strconv.FormatUint(uint64(pay.Port),10))
			/*
			ACL Abfrage
			*/
			accesspayload := NewGetAccessPayload(pay.Hash,":"+strconv.FormatUint(uint64(pay.Port),10))
			p.Out <- NewEvent("GetAccess",accesspayload)
			result := <-accesspayload.Result
			if result==false {
				log.Print("ACL said no!")
				pay.Conn.Close()
				continue
			}
			/*END ACL*/
			payload := NewPortMapRequestPayload(pay.Port)
			p.Out <- NewEvent("GetPortMapEntry",payload)
			entry := <-payload.Result
			if entry == nil {
				pay.Conn.Close()
				continue
			}else if entry.TcpEntry {
				p.Out <- NewEvent("TcpConnection",pay)
			}else{
				p.Out <- NewEvent(entry.Topic,pay)
			}
		}
	}
}

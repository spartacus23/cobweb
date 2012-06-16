package main

import (
	"log"	
	)

type RequestDispatcher struct {
	In chan *Event
	Out chan *Event
}

func (p *RequestDispatcher) PushEvent(event *Event){
        p.In <- event
}
func (p *RequestDispatcher) PullEvent()*Event{
        return <-p.Out
}

func (rd *RequestDispatcher) backend(){
	for{
		event := <-rd.In
		payload := event.Payload.(*AuthenticatedConnectionPayload)
		conn := payload.Conn
		node := payload.Nodeinfo
		buff := make([]byte,1)
		_,err := conn.Read(buff)
		if err!=nil {
			conn.Close()
		}else{
			switch buff[0]{
				case 1:{
					pay := NewUpdatePayload(conn,node,true)
					rd.Out <- NewEvent("UpdateRequest",pay)
				}
				case 0:{
					pay := NewUpdatePayload(conn,node,false)
					rd.Out <- NewEvent("UpdateRequest",pay)
				}
				case 2:{
					log.Print("New Connect request!")
					rd.Out <- NewEvent("ConnectRequest",conn)
					}
				default:{
					log.Print("Unknown Request from "+node.NodeID)
					conn.Close()
				}
			}
		}
	}
}

func NewRequestDispatcher(backlog int)*RequestDispatcher{
	rd := new(RequestDispatcher)
	rd.In = make(chan *Event,backlog)
	rd.Out = make(chan *Event,backlog)
	go rd.backend()
	log.Print("RequestDispatcher up and running")
	return rd
}

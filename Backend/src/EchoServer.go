package main

import (
	"net"
	)

type EchoServer struct{
        In 		chan *Event
        Out 		chan *Event
}
func (p *EchoServer) PushEvent(event *Event){
        p.In <- event
}
func (p *EchoServer) PullEvent()*Event{
        return <-p.Out
}

func NewEchoServer(backlog int)*EchoServer{
	erg := new(EchoServer)
	erg.In = make(chan *Event,backlog)
	erg.Out= make(chan *Event,backlog)
	go erg.backend()
	return erg
}

func (es *EchoServer) handleClient(conn net.Conn){
	for{
		buff:= make([]byte,1024)
		b,err := conn.Read(buff)
		if err != nil {
			break
		}
		_,err = conn.Write(buff[:b])
		if err != nil {
			break
		}
	}
}

func (es *EchoServer) backend(){
	for{
		event := <-es.In
		if event.Topic == "EchoServerRequest" {
			pay:=event.Payload.(*AuthenticatedLocalClientPayload)
			go es.handleClient(pay.Conn)
		}
	}
}

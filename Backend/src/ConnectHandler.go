package main

import (
	"net"
	"log"
	"strconv"
	//"crypto/tls"
	"encoding/binary"
	_bytes "bytes"
	)

type ConnectHandler struct{
        In 		chan *Event
        Out 		chan *Event
		MyID		string
}
func (p *ConnectHandler) PushEvent(event *Event){
        p.In <- event
}
func (p *ConnectHandler) PullEvent()*Event{
        return <-p.Out
}

func NewConnectHandler(backlog int)*ConnectHandler{
	erg := new(ConnectHandler)
	erg.In = make(chan *Event,backlog)
	erg.Out= make(chan *Event,backlog)
	go erg.backend()
	return erg
}

func (ch *ConnectHandler) backend(){
	ret := make(chan string)
	ch.Out <- NewEvent("GetGlobalID",ret)
	ch.MyID = <-ret
	for{
		event := <-ch.In
		if event.Topic == "ConnectRequest" {
			conn := event.Payload.(net.Conn)
			go ch.HandleConnectRequest(conn)
		}
	}
}

type ConnectForwardPayload struct{
	Conn		net.Conn
	Target		string
	ttl		uint8
	Port		uint32
}

func (ch *ConnectHandler) HandleConnectRequest(conn net.Conn){
	buff := make([]byte,33)
	bytes,err := conn.Read(buff)
	if err!=nil || bytes!=33 {
		conn.Close()
		log.Print("Malformed Connect packet")
		return
	}
	ttl := uint8(buff[0])
	var port uint32
	binary.Read(_bytes.NewBuffer(buff[1:5]),binary.LittleEndian,&port)
	target := string(buff[5:bytes])
	if target == ch.MyID {
		log.Print("New ConnectLocalRequest to port: "+strconv.FormatUint(uint64(port),10))
		ch.Out <- NewEvent("ConnectLocalRequest",&ConnectLocalPayload{conn,port})
		return
	}
	if ttl == 0 {
		log.Print("TTL == 0 -> dont forward")
		conn.Close()
		return
	}
	log.Print("New ConnectForwardRequest")
	ch.Out <- NewEvent("ConnectForwardRequest",&ConnectForwardPayload{conn,target,ttl,port})	
}

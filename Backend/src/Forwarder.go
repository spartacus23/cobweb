package main

import (
	"crypto/tls"
	"log"
	"net"
	"strconv"
	"encoding/binary"
	_bytes "bytes"
)

type Forwarder struct {
	In  chan *Event
	Out chan *Event
}

func (p *Forwarder) PushEvent(event *Event) {
	p.In <- event
}
func (p *Forwarder) PullEvent() *Event {
	return <-p.Out
}

func NewForwarder(backlog int) *Forwarder {
	erg := new(Forwarder)
	erg.In = make(chan *Event, backlog)
	erg.Out = make(chan *Event, backlog)
	go erg.backend()
	return erg
}

func shovel(c1, c2 net.Conn) {
	//causes error ?!
	/*defer func(){
		if c1!=nil {
			c1.Close()
		}
		if c2!=nil{
			c2.Close()
		}	
	}()
	*/
	buff := make([]byte, 1024)
	var err error
	var bytes int
	for {
		bytes, err = c1.Read(buff)
		if err != nil {
			log.Print("Shovel Error: " + err.Error())
			c2.Close()
			return
		}
		bytes, err = c2.Write(buff[:bytes])
		if err != nil {
			log.Print("Shovel Error: " + err.Error())
			c1.Close()
			return
		}
	}
}

func (fw *Forwarder) SendConnectPacket(target string, port uint32, ttl uint8, conn net.Conn) {
	packet := make([]byte, 34)
	packet[0] = 2
	packet[1] = ttl
	buf := new(_bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, port)
	copy(packet[2:6],buf.Bytes())
	copy(packet[6:], target)
	conn.Write(packet)
	//log.Print("Send connectpacket ok!")
}

func (fw *Forwarder) Forward(target string, port uint32,ttl uint8, conn1 net.Conn) {
	p := NewGetBestNextNodeIDPayload(target)
	fw.Out <- NewEvent("GetBestNextNodeID", p)
	nextnodeid := <-p.ret
	if nextnodeid == "" {
		log.Print("No more routes available")
		conn1.Close()
		return
	}
	p1 := NewGetNodePayload(nextnodeid)
	fw.Out <- NewEvent("GetNode", p1)
	node := <-p1.ret
	ret := make(chan *tls.Config)
	fw.Out <- NewEvent("GetTlsConfig", ret)
	config := <-ret
	conn2, err := tls.Dial("tcp", node.Hostname+":"+strconv.Itoa(node.Port), config)
	if err != nil {
		fw.Out <- NewEvent("AnnounceOffline", node)
		fw.Out <- NewEvent("ConnectForwardRequest", &ConnectForwardPayload{conn1, target, ttl, port})
		return
	}
	fw.SendConnectPacket(target,port, ttl-1, conn2)

	go shovel(conn1, conn2)
	go shovel(conn2, conn1)

}

func (fw *Forwarder) backend() {
	for {
		event := <-fw.In
		if event.Topic == "ConnectForwardRequest" {
			pay := event.Payload.(*ConnectForwardPayload)
			target := pay.Target
			ttl := pay.ttl
			conn1 := pay.Conn
			go fw.Forward(target,pay.Port, ttl, conn1)
		} else if event.Topic == "InitialConnectRequest" {
			pay := event.Payload.(*InitialConnectPayload)
			pay.Return <- fw.InitialConnect(pay.Target,pay.Port, pay.TTL)
		}
	}
}

type InitialConnectPayload struct {
	Target string
	Port uint32
	TTL uint8
	Return chan *tls.Conn
}

func NewInitialConnectPayload(target string,port uint32,ttl uint8) *InitialConnectPayload {
	erg := new(InitialConnectPayload)
	erg.Target = target
	erg.Return = make(chan *tls.Conn)
	erg.Port = port
	return erg
}

func (fw *Forwarder) InitialConnect(target string,port uint32, ttl uint8) *tls.Conn {
	p := NewGetBestNextNodeIDPayload(target)
	fw.Out <- NewEvent("GetBestNextNodeID", p)
	nextnodeid := <-p.ret
	if nextnodeid == "" {
		log.Print("No more routes available")
		return nil
	}
	p1 := NewGetNodePayload(nextnodeid)
	fw.Out <- NewEvent("GetNode", p1)
	node := <-p1.ret
	ret := make(chan *tls.Config)
	fw.Out <- NewEvent("GetTlsConfig", ret)
	config := <-ret
	conn, err := tls.Dial("tcp", node.Hostname+":"+strconv.Itoa(node.Port), config)
	if err != nil {
		fw.Out <- NewEvent("AnnounceOffline", node)
		return nil
	}
	fw.SendConnectPacket(target, port, ttl-1, conn)
	crypted := tls.Client(conn, config)
	err = crypted.Handshake()
	if err != nil {
		log.Print("Initial Connect Handshake failed: "+err.Error())
		return nil
	}
	return crypted
}

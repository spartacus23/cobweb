package main

import (
	"net"
	"log"
	//"strconv"
	"crypto/tls"
	)

type ConnectionAuthenticator struct{
        In chan *Event
        Out chan *Event
}
func (p *ConnectionAuthenticator) PushEvent(event *Event){
        p.In <- event
}
func (p *ConnectionAuthenticator) PullEvent()*Event{
        return <-p.Out
}

type AuthenticatedConnectionPayload struct {
	Conn 		net.Conn
	Nodeinfo	*Node
}

func NewAuthenticatedConnectionPayload(conn net.Conn,node *Node)*AuthenticatedConnectionPayload{
	erg := new(AuthenticatedConnectionPayload)
	erg.Conn = conn
	erg.Nodeinfo = node
	return erg
}

func (p *ConnectionAuthenticator) AuthenticateConnection(conninfo *UnauthConnectionInfo){
	//CheckIfKnown *CheckIfKnownPayload
	payload := NewCheckIfKnownPayload(conninfo.CertHash)
	p.Out <- NewEvent("CheckIfKnown",payload)
	node := <- payload.Return
	if node != nil {
		node.Online = true
		//log.Print("Authenticated connection request from "+conninfo.CertHash+" ("+node.Hostname+":"+strconv.Itoa(node.Port)+")")
		p.Out <- NewEvent("NewAuthenticatedConnection",NewAuthenticatedConnectionPayload(conninfo.Conn,node))
	}else{
		log.Print("Can't authenticate request, dismiss")
		conninfo.Conn.(*tls.Conn).Close()
	}	
}

func NewConnectionAuthenticator(backlog int)*ConnectionAuthenticator{
        erg := new(ConnectionAuthenticator)
        erg.In = make(chan *Event,backlog)
        erg.Out = make(chan *Event,backlog)
        go func(){
                for{
                        event := <- erg.In
                        go erg.AuthenticateConnection(event.Payload.(*UnauthConnectionInfo))
                }
        }()
        log.Print("ConnectionAuthenticator is up and running")
        return erg
}


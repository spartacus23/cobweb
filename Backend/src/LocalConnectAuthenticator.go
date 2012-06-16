package main

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"log"
	"net"
)

type LocalConnectAuthenticator struct {
	In  chan *Event
	Out chan *Event
}

func (p *LocalConnectAuthenticator) PushEvent(event *Event) {
	p.In <- event
}
func (p *LocalConnectAuthenticator) PullEvent() *Event {
	return <-p.Out
}

func NewLocalConnectAuthenticator(backlog int) *LocalConnectAuthenticator {
	erg := new(LocalConnectAuthenticator)
	erg.In = make(chan *Event, backlog)
	erg.Out = make(chan *Event, backlog)
	go erg.backend()
	return erg
}

type ConnectLocalPayload struct{
	Conn net.Conn
	Port uint32
}

func (ca *LocalConnectAuthenticator) backend() {
	for {
		event := <-ca.In
		if event.Topic == "ConnectLocalRequest" {
			pay := event.Payload.(*ConnectLocalPayload)
			go ca.AuthenticateClient(pay)
		}
	}
}

type AuthenticatedLocalClientPayload struct {
	Conn net.Conn
	Port uint32
	Hash string
}

func (ca *LocalConnectAuthenticator) AuthenticateClient(payload *ConnectLocalPayload) {
	conn := payload.Conn
	port := payload.Port
	ret := make(chan *tls.Config)
	ca.Out <- NewEvent("GetTlsConfig", ret)
	conf := <-ret
	authed := tls.Server(conn, conf)
	err := authed.Handshake()
	if err != nil {
		log.Print("LocalAuth failed during Handshake")
		return
	}
	hash := sha1.New()
	hash.Write(authed.ConnectionState().PeerCertificates[0].Raw)
	sha1hash := hash.Sum(nil)
	encoder := base64.NewEncoding("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ+/")
	hashstring := encoder.EncodeToString(sha1hash)
	log.Print("Got new authenticated connection from " + hashstring)
	ca.Out <- NewEvent("AuthenticatedLocalClient", &AuthenticatedLocalClientPayload{authed, port, hashstring})
}

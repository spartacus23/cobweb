package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
	"time"
)

type ConnectionAcceptor struct {
	in        chan *Event
	out       chan *Event
	Tlsconfig tls.Config
	Myhash    string
}

func (ca *ConnectionAcceptor) PushEvent(event *Event) {
	ca.in <- event
}

func (ca *ConnectionAcceptor) PullEvent() *Event {
	return <-ca.out
}

func (ca *ConnectionAcceptor) LoadKeys() error {
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Print("must generate Keys, this may take a while")
		//Generate keys (from go source)
		priv, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			log.Fatalf("failed to generate private key: %s", err)
			return errors.New("keyfail")
		}

		now := time.Now()
		serial,_ := rand.Int(rand.Reader,big.NewInt(int64(1)<<62))
		template := x509.Certificate{
			SerialNumber: serial,
			Subject: pkix.Name{
				CommonName:   "127.0.0.1",
				Organization: []string{"Acme Co"},
			},
			NotBefore: now.Add(-5 * time.Minute).UTC(),
			NotAfter:  now.AddDate(1, 0, 0).UTC(), // valid for 1 year.

			SubjectKeyId: []byte{1, 2, 3, 4},
			KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		}

		derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
		if err != nil {
			log.Fatalf("Failed to create certificate: %s", err)
			return errors.New("keyfail")
		}

		certOut, err := os.Create("cert.pem")
		if err != nil {
			log.Fatalf("failed to open cert.pem for writing: %s", err)
			return errors.New("keyfail")
		}
		pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
		certOut.Close()
		log.Print("written cert.pem\n")

		keyOut, err := os.OpenFile("key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			log.Print("failed to open key.pem for writing:", err)
			return errors.New("keyfail")
		}
		pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
		keyOut.Close()
		log.Print("written key.pem\n")
		log.Print("Generated Keys successfully")
	}
	cert, err = tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		return errors.New("failed to load/create keys")
	}
	certs := make([]tls.Certificate, 1)
	certs[0] = cert
	ca.Tlsconfig.Certificates = certs
	ca.Tlsconfig.ClientAuth = tls.RequestClientCert
	ca.Tlsconfig.InsecureSkipVerify = true
	return nil
}

type UnauthConnectionInfo struct {
	Conn     net.Conn
	CertHash string
}

func (ca *ConnectionAcceptor) HandleConnection(conn net.Conn) {
	err := conn.(*tls.Conn).Handshake()
	if err != nil {
		log.Print("New connection failed during Handshake")
		log.Print(err)
	} else {
		state := conn.(*tls.Conn).ConnectionState()
		if len(state.PeerCertificates) == 0 {
			log.Print("New connection failed: no certificate present")
			conn.Close()
			return
		}
		hash := sha1.New()
		hash.Write(state.PeerCertificates[0].Raw)
		sha1hash := hash.Sum(nil)
		encoder := base64.NewEncoding("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ+/")
		hashstring := encoder.EncodeToString(sha1hash)
		if hashstring == ca.Myhash {
			log.Print("New admin connection")
			ca.out <- NewEvent("NewAdminConnection", conn)
		} else {
			ci := &UnauthConnectionInfo{conn, hashstring}
			event := NewEvent("NewUnauthenticatedConnection", ci)
			//log.Print("New unauthenticated connection")
			ca.out <- event
		}
	}
}

func (ca *ConnectionAcceptor) ServeEventRequests() {
	for {
		event := <-ca.in
		if event.Topic == "GetTlsConfig" {
			event.Payload.(chan *tls.Config) <- &ca.Tlsconfig
		} else if event.Topic == "GetGlobalID" {
			event.Payload.(chan string) <- ca.Myhash
		}
	}
}

func NewConnectionAcceptor(capacity int) *ConnectionAcceptor {
	ca := new(ConnectionAcceptor)
	ca.in = make(chan *Event, capacity)
	ca.out = make(chan *Event, capacity)
	err := ca.LoadKeys()
	if err != nil {
		return nil
	}

	go func() {
		ret := make(chan int)
		ca.out <- NewEvent("GetPort", ret)
		port := <-ret
		listener, err := tls.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port), &ca.Tlsconfig)
		port = listener.Addr().(*net.TCPAddr).Port
		log.Print("ConnectionAcceptor is up and running on port " + strconv.Itoa(port))
		if err != nil {
			log.Print("ConnectionAcceptor failed to start... is the port busy?")
			return
		}
		go ca.ServeEventRequests()
		hash := sha1.New()
		hash.Write(ca.Tlsconfig.Certificates[0].Certificate[0])
		sha1hash := hash.Sum(nil)
		encoder := base64.NewEncoding("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ+/")
		ca.Myhash = encoder.EncodeToString(sha1hash)
		log.Print("Our certhash is " + ca.Myhash)
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Print("ConnectionAcceptor failed to accept...")
				break
			}
			//log.Print("Accepted new socket")
			go ca.HandleConnection(conn)
		}
	}()
	return ca
}

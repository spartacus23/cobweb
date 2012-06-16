package main

import (
	"net"
	"os"
	"bytes"
	"strings"
	"errors"
	"log"
	"regexp"
	)

type FileServer struct{
        In 			chan *Event
        Out 		chan *Event
        pathregexp	*regexp.Regexp
}
func (p *FileServer) PushEvent(event *Event){
        p.In <- event
}
func (p *FileServer) PullEvent()*Event{
        return <-p.Out
}

func NewFileServer(backlog int)*FileServer{
	erg := new(FileServer)
	erg.In = make(chan *Event,backlog)
	erg.Out= make(chan *Event,backlog)
	erg.pathregexp = regexp.MustCompile("/^\\.[a-zA-Z0-9_%-/]+$/")
	go erg.backend()
	return erg
}

func (es *FileServer) handleClient(conn net.Conn){
	log.Print("Begin handling fileserver request")
	for{
		request := new(bytes.Buffer)
		for {
			buff := make([]byte,1024)
			b,err := conn.Read(buff)
			if err!=nil {
				return
			}
			request.Write(buff[:b])
			if buff[b-1]==byte('\n'){
				break
			}
		}
		words := strings.Split(request.String()," ")
		if len(words)!=2 {
			conn.Write([]byte("Error while parsing request\n"))
			continue
		}
		if words[0]=="GET" {
			path := strings.Trim(words[1],"\r\n ")
			err := es.AwnserGetRequest(conn,"."+path)
			if err!=nil {
				conn.Write([]byte(err.Error()+"\n"))
			}
		}
	}
}

func (es *FileServer) backend(){
	for{
		event := <-es.In
		if event.Topic == "FileServerRequest" {
			pay:=event.Payload.(*AuthenticatedLocalClientPayload)
			go es.handleClient(pay.Conn)
		}
	}
}

func (fs *FileServer) TestIfPathIsValid(path string) bool {
	return fs.pathregexp.MatchString(path)
}

func (fs *FileServer) AwnserGetRequest(conn net.Conn,path string) error{
	log.Print("Get request: "+path)
	valid := fs.TestIfPathIsValid(path)
	if !valid {
		conn.Write([]byte("Don't try this again dude!\n"))
		return errors.New("HackingAlert")
	}
	out,err := fs.GenerateDirectoryIndex(path)
	if err!=nil { // path is a file not a dir
		file,err := os.Open(path)
		if err!=nil {
			return err
		}
		defer file.Close()
		for{
			buff := make([]byte,1024)
			b,err := file.Read(buff)
			if err!=nil {
				break
			}
			_,err = conn.Write(buff[:b])
			if err!=nil {
				return errors.New("Client disconnected")
			}
		}	
	}else{
		conn.Write([]byte(out))
	}
	return nil
}

func (fs *FileServer) GenerateDirectoryIndex(dirname string) (string,error) {
	stat,err := os.Lstat(dirname)
	if err!=nil || !stat.IsDir() {
		return "",errors.New(dirname+" is not a directory")
	}
	dir,err := os.Open(dirname)
	if err!=nil {
		return "",err
	}
	defer dir.Close()
	names,err := dir.Readdirnames(0)
	if err!=nil {
		return "",err
	}
	out := ""
	for _,val := range names {
		out += val+"\n"
	}
	return out,nil
}


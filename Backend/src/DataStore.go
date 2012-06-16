package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

type Node struct {
	NodeID   string
	Alias    string
	Hostname string
	Port     int
	Online   bool
}

func NewNode(id, host string, port int) *Node {
	erg := new(Node)
	erg.NodeID = id
	erg.Hostname = host
	erg.Port = port
	erg.Alias = ""
	erg.Online = true
	return erg
}

type DataStore struct {
	In        chan *Event
	Out       chan *Event
	Filename  string
	Nodes     []*Node
	LocalPort int
	FrontendPort int
}

func (p *DataStore) PushEvent(event *Event) {
	p.In <- event
}
func (p *DataStore) PullEvent() *Event {
	return <-p.Out
}

func (ds *DataStore) LoadFromFile() {
	file, err := os.Open(ds.Filename)
	ds.Nodes = make([]*Node, 0)
	if err != nil {
		log.Print("No Configfile!!!")
		return
	}
	reader := bufio.NewReaderSize(file, 4096)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		words := strings.Split(string(line), " ")
		switch words[0] {
		case "Node":
			{
				wordcount := len(words)
				if wordcount >= 4 {
					name := words[1]
					host := words[2]
					port, err := strconv.Atoi(strings.Trim(words[3], "\r\n "))
					if err != nil {
						log.Print("Malformed line in configfile (" + string(line) + ")")
						break
					}
					ds.Nodes = append(ds.Nodes, NewNode(name, host, port))
					if wordcount > 4 {
						ds.Nodes[len(ds.Nodes)-1].Alias = strings.Trim(words[4], "\r\n ")
					}
				}
			}
		case "Port":
			{
				wordcount := len(words)
				if wordcount >= 2 {
					port, err := strconv.Atoi(words[1])
					if err == nil {
						ds.LocalPort = port
					}
				}
			}
		case "FrontendPort":
			{
				wordcount := len(words)
				if wordcount >= 2 {
					port, err := strconv.Atoi(words[1])
					if err == nil {
						ds.FrontendPort = port
					}
				}
			}
		default:
			{
				log.Print("Malformed line in configfile (" + string(line) + ")")
			}
		}
	}
}

func (ds *DataStore) SaveToFile() {
	file, err := os.Create(ds.Filename)
	if err != nil {
		log.Print("Error writing configfile")
		return
	}
	for _, node := range ds.Nodes {
		line := "Node " + node.NodeID + " " + node.Hostname + " " + strconv.Itoa(node.Port)
		if node.Alias != "" {
			line += " " + node.Alias + "\n"
		} else {
			line += "\n"
		}
		file.WriteString(line)
	}
	file.WriteString("Port " + strconv.Itoa(ds.LocalPort) + "\n")
	file.WriteString("FrontendPort " + strconv.Itoa(ds.FrontendPort) + "\n")
}

type AliasPayload struct {
	NodeID string
	Alias  string
}

func NewAliasPayload(id, a string) *AliasPayload {
	erg := new(AliasPayload)
	erg.NodeID = id
	erg.Alias = a
	return erg
}

type CheckIfKnownPayload struct {
	Name   string
	Return chan *Node
}

func NewCheckIfKnownPayload(name string) *CheckIfKnownPayload {
	erg := new(CheckIfKnownPayload)
	erg.Name = name
	erg.Return = make(chan *Node)
	return erg
}

type GetNodePayload struct {
	id  string
	ret chan *Node
}

func NewGetNodePayload(id string) *GetNodePayload {
	erg := new(GetNodePayload)
	erg.id = id
	erg.ret = make(chan *Node)
	return erg
}

func NewDataStore(filename string, capacity int) *DataStore {
	ds := new(DataStore)
	ds.In = make(chan *Event, capacity)
	ds.Out = make(chan *Event, capacity)
	ds.Filename = filename
	ds.LoadFromFile()
	log.Print("DataStore is up and running")
	go func() {
		for {
			// AddNode 	*Node
			// SetAlias 	*AliasPayload
			// CheckIfKnown *CheckIfKnownPayload
			// GetNodeList	chan []*Node
			// DeleteNode	string
			// GetPort	chan int
			event := <-ds.In
			if event.Topic == "AddNode" {
				ds.Nodes = append(ds.Nodes, event.Payload.(*Node))
				ds.SaveToFile()
			} else if event.Topic == "GetNode" {
				pay := event.Payload.(*GetNodePayload)
				for _, node := range ds.Nodes {
					if node.NodeID == pay.id {
						pay.ret <- node
						break
					}
				}
			} else if event.Topic == "SetAlias" {
				p := event.Payload.(*AliasPayload)
				id := p.NodeID
				a := p.Alias
				for _, node := range ds.Nodes {
					if node.NodeID == id {
						node.Alias = a
						break
					}
				}
				ds.SaveToFile()
			} else if event.Topic == "CheckIfKnown" {
				p := event.Payload.(*CheckIfKnownPayload)
				name := p.Name
				known := false		
				var pos int
				for idx, node := range ds.Nodes {
					if node.NodeID == name {
						pos = idx
						known = true
						break
					}
				}
				if known {
					p.Return <- ds.Nodes[pos]
				} else {
					p.Return <- nil
				}
			} else if event.Topic == "GetNodeList" {
				Return := event.Payload.(chan []*Node)
				erg := make([]*Node, len(ds.Nodes))
				copy(erg, ds.Nodes)
				Return <- erg
			} else if event.Topic == "DeleteNode" {
				id := event.Payload.(string)
				log.Print("Try to delete node " + id)
				var pos int = -1
				for idx, node := range ds.Nodes {
					if id == node.NodeID {
						pos = idx
						break
					}
				}
				if pos != -1 {
					log.Print("Found node, deleting")
					newnodes := make([]*Node, len(ds.Nodes)-1)
					copy(newnodes, ds.Nodes[:pos])
					copy(newnodes[pos:], ds.Nodes[pos+1:])
					ds.Nodes = newnodes
					ds.SaveToFile()
				}
			} else if event.Topic == "GetPort" {
				ret := event.Payload.(chan int)
				ret <- ds.LocalPort
			} else if event.Topic == "GetFrontendPort" {
				ret := event.Payload.(chan int)
				ret <- ds.FrontendPort
			} else if event.Topic == "SavePortMapping" {
				m := event.Payload.(map[uint32]*PortMapEntry)
				f,err := os.Create("portmapping.txt")
				if err!=nil {
					log.Print("Cant open portmapping.txt for writing")
					continue
				}
				for key,val := range m {
					f.Write([]byte(strconv.FormatUint(uint64(key),10)+" "))
					if val.TcpEntry {
						f.Write([]byte(strconv.FormatUint(uint64(val.LocalPort),10)+"\n"))
					}else{
						f.Write([]byte(val.Topic+"\n"))
					}
				}
				f.Close()
			} else if event.Topic == "LoadPortMapping" {
				ret := event.Payload.(chan map[uint32]*PortMapEntry)
				f,err := os.Open("portmapping.txt")
				if err!=nil {
					log.Print("cant open portmapping.txt for reading")
					ret <- nil
					continue
				}
				reader := bufio.NewReader(f)
				portmap := make(map[uint32]*PortMapEntry)
				for{
					line,_,err := reader.ReadLine()
					if err!=nil {
						break
					}
					strs := strings.Split(string(line)," ")
					if len(strs)!=2 {
						log.Print("Invalid wordcount in portmapping.txt")
						continue
					}
					_cwport,err := strconv.ParseUint(strs[0],10,32)
					if err!=nil {
						log.Print("Invalid first word in portmapping.txt")
						continue
					}
					cwport := uint32(_cwport)
					__targ := strings.Trim(strs[1],"\n")
					_targ,err := strconv.ParseUint(__targ,10,32)
					if err!=nil {
						portmap[cwport]=NewPortMapEntry(false,__targ,0)
					}else{
						portmap[cwport]=NewPortMapEntry(true,"",uint16(_targ))
					}	
				}
				f.Close()
				ret <- portmap
			}
		}
	}()
	return ds
}

package main

import (
	"strconv"
	"log"
	)
/*
"AddPortMapEntry" func NewPortMapPayload(key uint32,tcp bool,topic string,port int) *PortMapPayload
"DelPortMapEntry" func NewPortMapPayload(key uint32,tcp bool,topic string,port int) *PortMapPayload
"GetPortMapEntry" func NewPortMapRequestPayload(key uint32) *PortMapRequestPayload			(payload.Result)
"GetPortMap"	  chan map[uint32]*PortMapEntry
*/


type PortMapEntry struct {
	TcpEntry	bool
	Topic		string
	LocalPort	uint16
}

func NewPortMapEntry(tcp bool,topic string,port uint16) *PortMapEntry{
	pe := new(PortMapEntry)
	pe.TcpEntry = tcp
	pe.Topic    = topic
	pe.LocalPort= port
	return pe
}

type PortMapPayload struct {
	Key		uint32
	Entry		*PortMapEntry
}

func NewPortMapPayload(key uint32,tcp bool,topic string,port uint16) *PortMapPayload {
	payload := new(PortMapPayload)
	payload.Key = key
	payload.Entry = NewPortMapEntry(tcp,topic,port)
	return payload
}

type PortMapRequestPayload struct {
	Key	uint32
	Result	chan *PortMapEntry
}

func NewPortMapRequestPayload(key uint32) *PortMapRequestPayload {
	payload := new(PortMapRequestPayload)
	payload.Key = key
	payload.Result = make(chan *PortMapEntry)
	return payload
}

type PortMapper struct{
        In 		chan *Event
        Out 		chan *Event
        Map		map[uint32]*PortMapEntry
}
func (p *PortMapper) PushEvent(event *Event){
        p.In <- event
}
func (p *PortMapper) PullEvent()*Event{
        return <-p.Out
}

func NewPortMapper(backlog int) *PortMapper {
	pm := new(PortMapper)
	pm.In  = make(chan *Event,backlog)
	pm.Out = make(chan *Event,backlog)
	pm.Map = make(map[uint32]*PortMapEntry)
	go func(){
		ret := make(chan map[uint32]*PortMapEntry)
		pm.Out <- NewEvent("LoadPortMapping",ret)
		erg := <-ret
		if erg!=nil {
			pm.Map = erg
		}
		for{
			event := <- pm.In
			switch(event.Topic){
				case "AddPortMapEntry" : {
					payload := event.Payload.(*PortMapPayload)
					pm.Map[payload.Key]=payload.Entry
					var target string
					if payload.Entry.TcpEntry {
						target = strconv.FormatInt(int64(payload.Entry.LocalPort),10)
					}else{
						target = payload.Entry.Topic
					}
					log.Print("Added PortMapEntry: "+strconv.FormatUint(uint64(payload.Key),10)+" zu "+target)
					pm.Out <- NewEvent("SavePortMapping",pm.Map)
				}
				case "DelPortMapEntry" : {
					payload := event.Payload.(*PortMapPayload)
					delete(pm.Map,payload.Key)
					pm.Out <- NewEvent("SavePortMapping",pm.Map)
				}
				case "GetPortMapEntry" : {
					payload := event.Payload.(*PortMapRequestPayload)
					payload.Result <- pm.Map[payload.Key]					
				}
				case "GetPortMap" : {
					event.Payload.(chan map[uint32]*PortMapEntry) <- pm.Map
				}
			}
		}
	}()
	return pm
}



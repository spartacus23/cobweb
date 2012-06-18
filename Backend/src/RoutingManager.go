package main

import (
	"net"
)

type RoutingEntry struct {
	Target	string
	Dist	float32
	TTL		int
}

type RoutingList []*RoutingEntry

type RoutingTable struct{
	//All of my informations next->routinglist
	MyView		map[string]RoutingList
	//their point of view of my information
	TheirView	map[string]RoutingList
}

func NewRoutingTable() *RoutingTable {
	result := new(RoutingTable)
	result.MyView = make(map[string]RoutingList)
	result.TheirView = make(map[string]RoutingList)
	return result
}

func (t *RoutingTable) GetMinList() RoutingList {
	result := make(RoutingList,0)
	//laufe über alle RoutingEntries			
	for _,list := range t.MyView {
		for _,entry := range list {
			//Test whether its shorter or not
			insert := true
			for idx,res := range result {
				if entry.Target == res.Target{
					insert = false
					if entry.Dist < res.Dist {
						result[idx]=entry
					}
				}
			}
			if insert==true {
				result = append(result,entry)
			}
		}
	}
	return result
}

func (t *RoutingTable) GetDistributeableMinList() RoutingList {
	list := t.GetMinList()
	result := make(RoutingList,0)
	for _,val := range list {
		//kick all, which has an TTL of 0
		if val.TTL > 0 {
			result = append(result,val)
		}
	}
	for idx,_ := range result {
		result[idx].TTL-=1
		result[idx].Dist+=1
	}
	return result
}

type UpdatePayload struct {
	Conn     net.Conn
	Nodeinfo *Node
	Online   bool
}

func NewUpdatePayload(c net.Conn, node *Node, on bool) *UpdatePayload {
	erg := new(UpdatePayload)
	erg.Conn = c
	erg.Nodeinfo = node
	erg.Online = on
	return erg
}

type GetBestNextNodeIDPayload struct {
	target string
	ret    chan string
}

func NewGetBestNextNodeIDPayload(target string) *GetBestNextNodeIDPayload {
	erg := new(GetBestNextNodeIDPayload)
	erg.target = target
	erg.ret = make(chan string)
	return erg
}

type CheckIfReachablePayload struct {
	Target string
	Return chan bool
}

type PersonInfo struct {
	Name		string
	Distance	string
}



type RoutingManager struct {
	In		chan *Event
	Out 	chan *Event
	Table	*RoutingTable		
}

func (p *RoutingManager) PushEvent(event *Event) {
	p.In <- event
}
func (p *RoutingManager) PullEvent() *Event {
	return <-p.Out
}

func NewRoutingManager(backlog int) *RoutingManager {
	man := new(RoutingManager)
	man.In = make(chan *Event,backlog)
	man.Out = make(chan *Event,backlog)
	man.Table = NewRoutingTable()
	go man.backend()
	return man
}

func (manager *RoutingManager) backend(){
	for{
		event := <-manager.In
		if event.Topic == "UpdateRequest" {
			
		}
	}
}



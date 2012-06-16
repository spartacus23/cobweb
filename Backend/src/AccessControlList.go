package main

import(
	"log"
	"os"
	"encoding/gob"
)

type BlackWhiteLists struct{
		BlackLists	map[string][]string
		WhiteLists	map[string][]string
		Policy		bool
}

type AccessControlList struct{
        In 			chan *Event
        Out 		chan *Event
		Lists		BlackWhiteLists
}
func (p *AccessControlList) PushEvent(event *Event){
        p.In <- event
}
func (p *AccessControlList) PullEvent()*Event{
        return <-p.Out
}

func (p *AccessControlList) AddBlackList(object string){
	_,ok := p.Lists.WhiteLists[object]
	if ok {
		delete(p.Lists.WhiteLists,object)
	}
	p.Lists.BlackLists[object] = make([]string,0)
}
func (p *AccessControlList) AddWhiteList(object string){
	_,ok := p.Lists.BlackLists[object]
	if ok {
		delete(p.Lists.BlackLists,object)
	}
	p.Lists.WhiteLists[object] = make([]string,0)
}

func (p *AccessControlList) AddToWhiteList(object,id string){
	list,ok := p.Lists.WhiteLists[object]
	if !ok {
		p.AddWhiteList(object)
		list = p.Lists.WhiteLists[object]
	}
	list = append(list,id)
	p.Lists.WhiteLists[object] = list
	p.Save()
}

func (p *AccessControlList) AddToBlackList(object,id string){
	list,ok := p.Lists.BlackLists[object]
	if !ok {
		p.AddBlackList(object)
		list = p.Lists.BlackLists[object]
	}
	list = append(list,id)
	p.Lists.BlackLists[object] = list
	p.Save()
}

func (p *AccessControlList) GetAccess(object,id string) bool {
	whitelist,ok := p.Lists.WhiteLists[object]
	if !ok {
		blacklist,ok := p.Lists.BlackLists[object]
		if !ok {
			//no whitelist, no blacklist
			return p.Lists.Policy
		}
		for _,val := range blacklist {
			if id==val {
				// id in blacklist!
				return false
			}
		}
		// id not in blacklist
		return true
	}
	for _,val := range whitelist {
		if id==val {
			// id in whitelist
			return true
		}
	}
	// id not in whitelist
	return false
}

func (p *AccessControlList) Save() error{
	f,err := os.Create(".blackwhitelists.gob")
	if err!=nil{
		p.Lists.Policy = true;
		return err
	}
	encoder := gob.NewEncoder(f)
	encoder.Encode(p.Lists)
	f.Close()
	return nil
}

func (p *AccessControlList) Load() error {
	f,err := os.Open(".blackwhitelists.gob")
	p.Lists.BlackLists = make(map[string][]string)
	p.Lists.WhiteLists = make(map[string][]string)
	p.Lists.Policy = true;
	if err!=nil{
		return err
	}
	decoder := gob.NewDecoder(f)
	decoder.Decode(&p.Lists)
	f.Close()
	return nil
}

func NewAccessControlList(backlog int)*AccessControlList{
	erg := new(AccessControlList)
	erg.In = make(chan *Event,backlog)
	erg.Out= make(chan *Event,backlog)
	erg.Load()
	go erg.backend()
	return erg
}

type AddToACLPayload struct {
	Blacklist	bool
	Object		string
	Id			string
}

func NewAddToACLPayload(black bool,object,id string) *AddToACLPayload {
	return &AddToACLPayload{
				Blacklist 	: black,
				Object 		: object,
				Id 			: id,
			}
}

type DelFromACLPayload struct {
	Object	string
	Id		string
}

func NewDelFromACLPayload(obj,id string) *DelFromACLPayload {
	return &DelFromACLPayload{
			Object : obj,
			Id : id,
		}
}

func (p *AccessControlList) DelFromACL(obj,id string) {
	list,ok := p.Lists.BlackLists[obj]
	if !ok {
		list,ok = p.Lists.WhiteLists[obj]
		if !ok {
			return
		}
		newlist := make([]string,0,len(list)-1)
		for _,val := range list {
			if val!=id {
				newlist = append(newlist,val)
			}
		}
		if len(newlist)==0 {
			delete(p.Lists.WhiteLists,obj)
		}else{
			p.Lists.WhiteLists[obj]=newlist
		}
	}else{
		newlist := make([]string,0,len(list)-1)
		for _,val := range list {
			if val!=id {
				newlist = append(newlist,val)
			}
		}
		if len(newlist)==0 {
			delete(p.Lists.BlackLists,obj)
		}else{
			p.Lists.BlackLists[obj]=newlist
		}
	}
}

type GetAccessPayload struct {
	Subject		string
	Object		string
	Result		chan bool
}
func NewGetAccessPayload(sub,ob string) *GetAccessPayload {
	return &GetAccessPayload{
			Subject: sub,
			Object : ob,
			Result : make(chan bool),
		}
}


func (p *AccessControlList) backend(){
	log.Print("ACL up and running...")
	for{
		event := <-p.In
		if event.Topic == "AddToACL" {
			pay := event.Payload.(*AddToACLPayload)
			if pay.Blacklist == true {
				p.AddToBlackList(pay.Object,pay.Id)
			}else{
				p.AddToWhiteList(pay.Object,pay.Id)
			}
		}
		if event.Topic == "DelFromACL" {
			pay := event.Payload.(*DelFromACLPayload)
			p.DelFromACL(pay.Object,pay.Id)
		}
		if event.Topic == "GetACL" {
			event.Payload.(chan *BlackWhiteLists) <- &p.Lists
		}
		if event.Topic == "SetPolicy" {
			p.Lists.Policy=event.Payload.(bool)
			p.Save()
		}
		if event.Topic == "GetPolicy" {
			event.Payload.(chan bool) <- p.Lists.Policy
		}
		if event.Topic == "GetAccess" {
			pay := event.Payload.(*GetAccessPayload)
			erg := p.GetAccess(pay.Object,pay.Subject)
			pay.Result <- erg
		}		
	}
}


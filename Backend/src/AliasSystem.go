package main

import(
	"log"
	"io/ioutil"
	"strings"
	"os"
)

type AliasSystem struct{
        In 		chan *Event
        Out 		chan *Event
		KeyValue	map[string]string
		ValueKey	map[string]string
}
func (p *AliasSystem) PushEvent(event *Event){
        p.In <- event
}
func (p *AliasSystem) PullEvent()*Event{
        return <-p.Out
}

func (p *AliasSystem) DelAlias(key,value string){
	delete(p.KeyValue,key)
	delete(p.ValueKey,value)
	p.Save()
}

func (p *AliasSystem) AddAlias(key,value string){
	p.KeyValue[key]=value
	p.ValueKey[value]=key
	p.Save()
}

func (p *AliasSystem) Save() error{
	f,err := os.Create(".aliasmap.txt")
	if err!=nil {
		return err
	}
	for idx,val := range p.KeyValue {
		f.Write([]byte(idx+" "+val+"\n"))
	}
	f.Close()
	return nil
}

func (p *AliasSystem) Read() error {
	file,err := ioutil.ReadFile(".aliasmap.txt")
	f := string(file)
	lines := strings.Split(f,"\n")
	p.KeyValue = make(map[string]string)
	p.ValueKey = make(map[string]string)
	for _,line := range lines {
		str := string(line)
		words := strings.Split(str," ")
		if len(words)<2 {
			continue
		}
		p.KeyValue[words[0]]=words[1]
		p.ValueKey[words[1]]=words[0]
	}
	return err
}

func NewAliasSystem(backlog int)*AliasSystem{
	erg := new(AliasSystem)
	erg.In = make(chan *Event,backlog)
	erg.Out= make(chan *Event,backlog)
	erg.KeyValue = make(map[string]string)
	erg.ValueKey = make(map[string]string)
	err := erg.Read()
	if err!=nil {
		log.Print(err.Error())
	}
	go erg.backend()
	return erg
}

type GetAliasPayload struct {
	Query	string
	Return	chan string
}

func NewGetAliasPayload(query string) *GetAliasPayload {
	erg := new(GetAliasPayload)
	erg.Query = query
	erg.Return = make(chan string)
	return erg
}

type SetAliasPayload struct {
	Id	  string
	Alias string
}

func NewSetAliasPayload(id,alias string) *SetAliasPayload {
	erg := new(SetAliasPayload)
	erg.Id = id
	erg.Alias = alias
	return erg
}

func (p *AliasSystem) backend(){
	log.Print("AliasSystem up and running")
	for{
		event := <-p.In
		if event.Topic == "GetAlias" {
			pay := event.Payload.(*GetAliasPayload)
			erg,ok := p.KeyValue[pay.Query]
			if !ok {
				erg,ok = p.ValueKey[pay.Query]
				if !ok {
					pay.Return <- ""
				}else{
					pay.Return <- erg
				}
			}else{
				pay.Return <- erg
			}
		}
		if event.Topic == "GetAliasFromId" {
			pay := event.Payload.(*GetAliasPayload)
			alias := p.KeyValue[pay.Query]
			pay.Return <- alias
		}
		if event.Topic == "GetIdFromAlias" {
			pay := event.Payload.(*GetAliasPayload)
			alias := p.ValueKey[pay.Query]
			pay.Return <- alias
		}
		if event.Topic == "SetAlias" {
			pay := event.Payload.(*SetAliasPayload)
			p.AddAlias(pay.Id,pay.Alias)
		}
		if event.Topic == "GetAliasList" {
			event.Payload.(chan map[string] string) <- p.KeyValue
		}
		if event.Topic == "DelAlias" {
			pay := event.Payload.(*SetAliasPayload)
			p.AddAlias(pay.Id,pay.Alias)
		}
	}
}



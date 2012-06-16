package main

import (
	"time"
	"net"
	"log"
	"strings"
	"strconv"
	)

type AdminHandler struct{
	In 		chan *Event
	Out 		chan *Event
}
func (ca *AdminHandler) PushEvent(event *Event){
	ca.In <- event
}
func (ca *AdminHandler) PullEvent()*Event{
	return <-ca.Out
}

func NewAdminHandler(backlog int)*AdminHandler{
	handler := new(AdminHandler)
	handler.In = make(chan *Event,backlog)
	handler.Out = make(chan *Event,backlog)
	go func(){
		log.Print("AdminHandler is up and running")
		for{
			event := <-handler.In
			if event.Topic == "NewAdminConnection"{
				go func(){
					conn := event.Payload.(net.Conn)
					for{
						buff := make([]byte,1024)
						bytes,err := conn.Read(buff)
						if err != nil {
							break
						}
						words := strings.Split(strings.Trim(string(buff[:bytes])," \n\r")," ")
						cmd := words[0]
						log.Print("Admin cmd: "+cmd)
						switch cmd{
							case "Help","help","h","H":{
								conn.Write([]byte("AddNode <certhash> <host> <port>\n"))
								conn.Write([]byte("DeleteNode <certhash>\n"))
								conn.Write([]byte("ShowNodes\n"))
								conn.Write([]byte("AddPortMapping [internal|external] CobWebPort [localport|topic]\n"))
								conn.Write([]byte("DeletePortMapping <CobWebPort>\n"))
								conn.Write([]byte("ShowPortMapping\n"))
								conn.Write([]byte("Connect <certhash> <port>\n"))
								conn.Write([]byte("Who\n"))
								conn.Write([]byte("ShowInbox\n"))
								conn.Write([]byte("GetMail <id>\n"))
								conn.Write([]byte("DeleteMail <id>\n"))
								conn.Write([]byte("ShowACL\n"))
								conn.Write([]byte("AddACL <black|white> <object> <subject>\n"))
								conn.Write([]byte("DelACL <object> <subject>\n"))
								conn.Write([]byte("SetACLPolicy [TRUE|FALSE]\n"))
								conn.Write([]byte("GetACLPolicy\n"))
								conn.Write([]byte("SetAlias <id> <alias>\n"))
								conn.Write([]byte("DelAlias <id> <alias>\n"))
								conn.Write([]byte("ShowAlias\n"))
								
							}
							case "AddNode":{
								if len(words) >= 4{
									id := words[1]
									host := words[2]
									port,err := strconv.Atoi(words[3])
									if err != nil || port > 65535{
										log.Print("AddNode failed, invalid port")
										break
									}
									node := NewNode(id,host,port)
									if len(words) > 4 {
										node.Alias = words[4]
									}
									handler.Out <- NewEvent("AddNode",node)
									conn.Write([]byte("Added "+id+" to nodelist\n"))
								}
							}
							case "DeleteNode":{
								if len(words) >= 2 {
									id := words[1]
									handler.Out <- NewEvent("DeleteNode",id)
									conn.Write([]byte("Removed "+id+" from nodelist\n"))
								}
							}
							case "ShowNodes":{
								ret := make(chan []*Node)
								handler.Out <- NewEvent("GetNodeList",ret)
								list := <- ret
								for _,val := range list {
									conn.Write([]byte(val.NodeID+" "+val.Hostname+":"+strconv.Itoa(val.Port)+"\n"))
								}
								if len(list)==0 {
									conn.Write([]byte("No nodes are known\n"))
								}
							}
							case "Connect":{
								if len(words)>=3 {
									target := words[1]
									_port,err := strconv.ParseUint(words[2],10,32)
									if err!=nil {
										conn.Write([]byte("Failed parsing port\n"))
										continue
									}
									port := uint32(_port)
									log.Print("AdminHandler:95 uint32 port-> ",port)
									ret := make(chan bool)
									handler.Out <- NewEvent("CheckIfReachable",&CheckIfReachablePayload{target,ret})
									avail := <-ret
									if !avail {
										conn.Write([]byte(target+" is offline"))
									}else{
										payload := NewInitialConnectPayload(target,port,5)/*TODO TTL dynamisch*/
										handler.Out <- NewEvent("InitialConnectRequest",payload)
										tunnel := <-payload.Return
										if tunnel == nil {
											log.Print("Tunnel could not be established")
											conn.Write([]byte("Tunnel could not be established\n"))
											continue
										}
										go shovel(conn,tunnel)
										go shovel(tunnel,conn)
								                return	
									}	
								}
							}
							case "Who":{
		 						ret := make(chan string)
								handler.Out <- NewEvent("GetListOfOnlineNodes",ret)
								conn.Write([]byte(<-ret))
							}
							case "AddPortMapping":{
								//AddPortMapping [internal|external] CobWebPort [localport|topic]
								if len(words)>=4 {
									inex := words[1]
									cwport,err := strconv.ParseInt(words[2],0,32)
									if err!=nil {
										conn.Write([]byte("Syntax Error: AddPortMapping [internal|external] CobWebPort [localport|topic]\n"))
										conn.Write([]byte("Invalid integer\n"))
										break
									}
									if inex[0]!='i' && inex[0]!='e' {
										conn.Write([]byte("Syntax Error: AddPortMapping [internal|external] CobWebPort [localport|topic]\n"))
										break;
									}
									if inex[0]=='i' {
										topic := words[3]
										payload := NewPortMapPayload(uint32(cwport),false,topic,0)
										handler.Out <- NewEvent("AddPortMapEntry",payload)
										conn.Write([]byte("Success\n"))
									}
									if inex[0]=='e' {
										local := words[3]
										l_port,err := strconv.ParseUint(local,0,32)
										if err!=nil{
											conn.Write([]byte("failed parsing integer\n"))
											break
										}
										payload := NewPortMapPayload(uint32(cwport),true,"",uint16(l_port))
										handler.Out <- NewEvent("AddPortMapEntry",payload)
										conn.Write([]byte("Success\n"))
									}
								}else{
									conn.Write([]byte("Syntax Error: AddPortMapping [internal|external] CobWebPort [localport|topic]\n"))
		
								}
							}
							case "DeletePortMapping":{
								if len(words)>=2 {
									cwport,err := strconv.ParseInt(words[1],0,32)
									if err!=nil {
										conn.Write([]byte("Syntax Error: DeletePortMapping <CobWebPort>\n"))
										break
									}
									payload := NewPortMapPayload(uint32(cwport),true,"",0)
									handler.Out <- NewEvent("DelPortMapEntry",payload)
									conn.Write([]byte("Success\n"))
								}else{
									conn.Write([]byte("Syntax Error: DeletePortMapping <CobWebPort>\n"))
								}
							}
							case "ShowPortMapping":{
								ret := make(chan map[uint32]*PortMapEntry)
								handler.Out <- NewEvent("GetPortMap",ret)
								_map := <-ret
								for cwport,entry := range _map {
									conn.Write([]byte(strconv.FormatUint(uint64(cwport),10)+" -> "))
									if entry.TcpEntry {
										conn.Write([]byte(strconv.FormatUint(uint64(entry.LocalPort),10)+"\n"))
									}else{
										conn.Write([]byte(entry.Topic+"\n"))
									}
								}
							}
							case "ShowInbox":{
								pay := NewGetMailListPayload(20)
								handler.Out <- NewEvent("GetMailList",pay)
								list := <-pay.Return
								if list == nil || len(list)==0{
									conn.Write([]byte("(Empty inbox)\n"))
									continue
								}
								for _,mail := range list {
									conn.Write([]byte("ID:"+strconv.FormatInt(int64(mail.Id),10)+" "))
									conn.Write([]byte("TIME:"+time.Unix(int64(mail.Time),0).Format(time.RFC1123)+" "))
									conn.Write([]byte("SENDER:"+mail.Sender+" "))
									conn.Write([]byte("SUBJECT:"+mail.Subject+"\n"))
								}
							}
							case "GetMail":{
								if len(words) >= 2 {
									id,err := strconv.ParseUint(words[1],10,64)
									if err!=nil {
										conn.Write([]byte("Error parsing id\n"))
										continue
									}
									pay := NewGetMailPayload(int(id))
									handler.Out <- NewEvent("GetMail",pay)
									mail := <-pay.Return
									if mail== nil {
										conn.Write([]byte("Mail not found\n"))
										continue
									}
									conn.Write([]byte("ID:"+strconv.FormatInt(int64(mail.Id),10)+" "))
									conn.Write([]byte("TIME:"+time.Unix(int64(mail.Time),0).Format(time.RFC1123)+" "))
									conn.Write([]byte("SENDER:"+mail.Sender+" "))
									conn.Write([]byte("SUBJECT:"+mail.Subject+"\n"))
									conn.Write([]byte(mail.Text))
								}
							}
							case "DeleteMail":{
								if len(words) >= 2 {
									id,err := strconv.ParseUint(words[1],10,64)
									if err!=nil {
										conn.Write([]byte("Error parsing id\n"))
										continue
									}
									pay := NewGetMailPayload(int(id))
									handler.Out <- NewEvent("DeleteMail",pay)
									ok := <-pay.Return
									if ok.Id == 1 {
										conn.Write([]byte("success\n"))
									}else{
										conn.Write([]byte("fail\n"))	
									}
								}
							}
							case "ShowACL":{
								ret := make(chan *BlackWhiteLists)
								handler.Out <- NewEvent("GetACL",ret)
								lists := <-ret
								conn.Write([]byte("Blacklists:\n"))
								for object,list := range lists.BlackLists {
									conn.Write([]byte("Object: "+object+":\n"))
									for _,id := range list {
										conn.Write([]byte("\t"+id+"\n"))
									}
								}
								conn.Write([]byte("Whitelists:\n"))
								for object,list := range lists.WhiteLists {
									conn.Write([]byte("Object: "+object+":\n"))
									for _,id := range list {
										conn.Write([]byte("\t"+id+"\n"))
									}
								}
							}
							case "AddACL": {
								if len(words)>=4 {
									blackwhite := words[1]
									object := words[2]
									subject := words[3]
									if blackwhite[0]=='b' {
										handler.Out <- NewEvent("AddToACL",NewAddToACLPayload(true,object,subject))
									}else{
										handler.Out <- NewEvent("AddToACL",NewAddToACLPayload(false,object,subject))
									}
									conn.Write([]byte("success\n"))
								}
							}
							case "DelACL": {
								if len(words)>=3 {
									object := words[1]
									subject := words[2]
									handler.Out <- NewEvent("DelFromACL",NewDelFromACLPayload(object,subject))
									conn.Write([]byte("success\n"))
								}
							}
							case "SetACLPolicy": {
								if len(words)>=2 {
									policy,err := strconv.ParseBool(words[1])
									if err==nil {
										handler.Out <- NewEvent("SetPolicy",policy)
									}
								}
							}
							case "GetACLPolicy": {
								conn.Write([]byte("Policy:\n"))
								result := make(chan bool)
								handler.Out <- NewEvent("GetPolicy",result)
								erg := <-result
								if erg {
									conn.Write([]byte("TRUE\n"))
								}else{
									conn.Write([]byte("FALSE\n"))
								}
							}
							case "SetAlias": {
								if len(words)>=3 {
									handler.Out<-NewEvent("SetAlias",NewSetAliasPayload(words[1],words[2]))
									conn.Write([]byte("success\n"))
								}
							}
							case "DelAlias": {
								if len(words)>=3 {
									handler.Out<-NewEvent("DelAlias",NewSetAliasPayload(words[1],words[2]))
									conn.Write([]byte("success\n"))
								}
							}
							case "ShowAlias" : {
								ret := make(chan map[string] string)
								handler.Out <- NewEvent("GetAliasList",ret)
								list := <-ret
								for key,val := range list {
									conn.Write([]byte(key+" "+val+"\n"))
								}
							}
						}
					}
					log.Print("Lost admin connection")
					conn.Close()
				}()
			}
		}
	}()
	return handler
}

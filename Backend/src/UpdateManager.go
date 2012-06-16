package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"strconv"
	"time"
)

type MultiLocker struct {
	locks map[uint32]bool
}

func (ml *MultiLocker) Lock(id uint32) {
	ml.locks[id] = true
	//log.Print("lock "+strconv.Itoa(int(id)))
}
func (ml *MultiLocker) UnLock(id uint32) {
	delete(ml.locks, id)
	//log.Print("unlock "+strconv.Itoa(int(id)))
}
func (ml *MultiLocker) GetLock(id uint32) bool {
	_, ok := ml.locks[id]
	return ok
}
func NewMultiLocker() *MultiLocker {
	ml := new(MultiLocker)
	ml.locks = make(map[uint32]bool)
	return ml
}

type RoutingInfo struct {
	NextNodeID string
	Distance   float32
}

type RoutingTable struct {
	table map[string][]*RoutingInfo
	ttls  map[string]uint8
}

func (rt *RoutingTable) GetMinimalDistance(target string) float32 {
	mindist := float32(9999999.9)
	for _, entry := range rt.table[target] {
		if entry.Distance < mindist {
			mindist = entry.Distance
		}
	}
	return mindist
}

func (rt *RoutingTable) GetBestNextNodeID(target string) (string, error) {
	list := rt.table[target]
	if list == nil || len(list) == 0 {
		return "", errors.New("No Entry in Routingtable")
	}
	minname := ""
	mindist := float32(9999999.9)
	for _, entry := range list {
		if entry.Distance < mindist {
			minname = entry.NextNodeID
			mindist = entry.Distance
		}
	}
	return minname, nil
}

func NewRoutingTable() *RoutingTable {
	erg := new(RoutingTable)
	erg.table = make(map[string][]*RoutingInfo)
	erg.ttls = make(map[string]uint8)
	return erg
}
func (rt *RoutingTable) Update(target, nextnode string, distance float32, ttl uint8, online bool) {
	if online {
		found := false
		for idx, val := range rt.table[target] {
			if val.NextNodeID == nextnode {
				rt.table[target][idx] = &RoutingInfo{nextnode, distance}
				found = true
				break
			}
		}
		if found == false {
			rt.table[target] = append(rt.table[target], &RoutingInfo{nextnode, distance})
			if len(rt.table[target]) == 1 {
				log.Print("User " + target + " is now ONLINE")
			}
		}
		rt.ttls[target] = ttl
	} else {
		_, ok := rt.table[target]
		if ok {
			log.Print("User " + target + " is now OFFLINE")
		}
		delete(rt.table, target)
		delete(rt.ttls, target)
	}
}

func (rt *RoutingTable) Cleanup(nextnodeid string) {
	for target, val := range rt.table {
		for idx, nodeinfo := range val {
			if nodeinfo.NextNodeID == nextnodeid {
				rt.table[target] = append(val[:idx], val[idx+1:]...)
				break
			}
		}
		if len(rt.table[target]) == 0 {
			delete(rt.table, target)
			log.Print("User " + target + " is now OFFLINE")
		}
	}
}

func (rt *RoutingTable) DeleteRoute(target, nextnode string) {
	list := rt.table[target]
	found := -1
	for i := 0; i < len(list); i++ {
		if list[i].NextNodeID == nextnode {
			found = i
		}
	}
	if found != -1 {
		newlist := make([]*RoutingInfo, len(list)-1)
		newlist = append(newlist, list[:found]...)
		newlist = append(newlist, list[found+1:]...)
		log.Print(fmt.Sprintf("oldlist: %v", list))
		log.Print(fmt.Sprintf("newlist: %v", newlist))
		rt.table[target] = newlist
	}
}

type UpdateManager struct {
	In           chan *Event
	Out          chan *Event
	routingtable *RoutingTable
	multilock    *MultiLocker
}

func (p *UpdateManager) PushEvent(event *Event) {
	p.In <- event
}
func (p *UpdateManager) PullEvent() *Event {
	return <-p.Out
}

func NewUpdateManager(backlog int) *UpdateManager {
	um := new(UpdateManager)
	um.In = make(chan *Event, backlog)
	um.Out = make(chan *Event, backlog)
	um.routingtable = NewRoutingTable()
	um.multilock = NewMultiLocker()
	go um.backend()
	//TEST
	go um.AnnounceOnline()
	//ret := make(chan string)
	//um.Out <- NewEvent("GetGlobalID",ret)
	//myid := <-ret
	//um.DistributeUpdate(rand.Uint32(),5,myid,0.0,nil,true)
	//ENDTEST
	log.Print("UpdateManager up and running")
	return um
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

func (um *UpdateManager) backend() {
	for {
		event := <-um.In
		switch event.Topic {
		case "UpdateRequest":
			go um.HandleUpdateRequest(event.Payload.(*UpdatePayload))
		case "AnnounceOffline":
			go um.AnnounceOffline(event.Payload.(*Node))
		case "GetBestNextNodeID":
			{
				pay := event.Payload.(*GetBestNextNodeIDPayload)
				name, err := um.routingtable.GetBestNextNodeID(pay.target)
				if err != nil {
					pay.ret <- ""
				} else {
					pay.ret <- name
				}
			}
		case "CheckIfReachable":
			{
				pay := event.Payload.(*CheckIfReachablePayload)
				_, ok := um.routingtable.table[pay.Target]
				pay.Return <- ok
			}
		case "GetListOfOnlineNodes": {
				Result := event.Payload.(chan string)
				erg := ""
				nodecount := len(um.routingtable.table)
				names := make([]string,nodecount)
				dists := make([]float32,nodecount)
				pos := 0
				for name,_ := range um.routingtable.table {
					names[pos] = name
					dists[pos] = um.routingtable.GetMinimalDistance(name)+1
					//dist := um.routingtable.GetMinimalDistance(name)+1
					//diststr := strconv.FormatFloat(float64(dist),'f',2,32)
					//erg += (name+" "+diststr+"\n")
					pos++
				}
				//sortieren
				for i:=0;i<len(names);i++{
					for j:=0;j<len(names)-1;j++ {
						if(dists[j]>dists[j+1]){
							d := dists[j]
							dists[j]=dists[j+1]
							dists[j+1]=d
							n := names[j]
							names[j]=names[j+1]
							names[j+1]=n
						}
					}
				}
				for i:=0;i<len(names);i++{
					erg+= (names[i]+" "+strconv.FormatFloat(float64(dists[i]),'f',2,32)+"\n")
				}
				Result <- erg
 			}
 		case "GetListOfOnlineNodes2": {
				Result := event.Payload.(chan []*PersonInfo)
				nodecount := len(um.routingtable.table)
				names := make([]string,nodecount)
				dists := make([]float32,nodecount)
				pos := 0
				for name,_ := range um.routingtable.table {
					names[pos] = name
					dists[pos] = um.routingtable.GetMinimalDistance(name)+1
					//dist := um.routingtable.GetMinimalDistance(name)+1
					//diststr := strconv.FormatFloat(float64(dist),'f',2,32)
					//erg += (name+" "+diststr+"\n")
					pos++
				}
				//sortieren 
				for i:=0;i<len(names);i++{
					for j:=0;j<len(names)-1;j++ {
						if(dists[j]>dists[j+1]){
							d := dists[j]
							dists[j]=dists[j+1]
							dists[j+1]=d
							n := names[j]
							names[j]=names[j+1]
							names[j+1]=n
						}
					}
				}
				resultvalues := make([]*PersonInfo,0)
				for idx,val := range names{
					_f := strconv.FormatFloat(float64(dists[idx]),'f',2,32)
					resultvalues = append(resultvalues,&PersonInfo{val,_f})
				}
				Result <- resultvalues
 			}
		}
	}
}

func (um *UpdateManager) HandleUpdateRequest(payload *UpdatePayload) {
	//log.Print("Got an UpdateRequest")
	conn := payload.Conn
	defer func() {
		conn.Write([]byte("\x00"))
		conn.Close()
	}()
	node := payload.Nodeinfo
	online := payload.Online
	buff := make([]byte, 37)
	b, err := conn.Read(buff)
	if err != nil || b < 37 {
		log.Print("Error while reading UpdateRequest from " + node.NodeID)
		return
	}
	_sid := buff[0:4]
	_ttl := buff[4]
	__dist := buff[5:9]
	_id := buff[9:b]

	var sid uint32
	buf := bytes.NewBuffer(_sid)
	binary.Read(buf, binary.LittleEndian, &sid)

	if um.multilock.GetLock(sid) {
		//log.Print("Dismiss UpdatePacket, SessionID is known")
		return
	}

	ttl := uint8(_ttl)

	var _dist uint32
	buf = bytes.NewBuffer(__dist)
	binary.Read(buf, binary.LittleEndian, &_dist)
	dist := math.Float32frombits(_dist)

	id := string(_id)

	//log.Print(fmt.Sprintf("UpdatePacket: sid: %v ttl: %v target: %v onoff: %v",sid,ttl,id,online))

	//log.Print(fmt.Sprintf("sid:%v ttl:%v dist:%v id: %v",sid,ttl,dist,id))
	/*
		if online {
			log.Print("Node "+node.NodeID+" says "+id+" is online")	
		}else{	
			log.Print("Node "+node.NodeID+" says "+id+" is offline")
		}
	*/
	
	ret := make(chan string)
	um.Out <- NewEvent("GetGlobalID", ret)
	myid := <-ret
	if myid != id {
		um.routingtable.Update(id, node.NodeID, dist, ttl-1, online)
	}
	if ttl > 0 {
		oldmindist := um.routingtable.GetMinimalDistance(id)
		if dist < oldmindist {
			um.DistributeUpdate(sid, ttl-1, id, dist+1, node, online)
		}else{
			um.DistributeUpdate(sid, ttl-1, id, oldmindist+1, node, online)
		}
	}
	//log.Print("Finished handling UpdateRequest")
}

func (um *UpdateManager) AnnounceOnline() {
	ret := make(chan string)
	um.Out <- NewEvent("GetGlobalID", ret)
	myid := <-ret
	for {
		//log.Print("Announce myself as online")
		um.DistributeUpdate(rand.Uint32(), 5, myid, 0.0, nil, true)
		secs := int64(30 + rand.Intn(5))
		time.Sleep(time.Duration(secs * 1000000000))
	}
}
func (um *UpdateManager) AnnounceOffline(node *Node) {
	node.Online = false
	um.routingtable.Cleanup(node.NodeID)
	um.routingtable.Update(node.NodeID, "", 0.0, 0, false)
	um.DistributeUpdate(rand.Uint32(), 5, node.NodeID, 1.0, node, false)
}

func (um *UpdateManager) DistributeUpdate(sid uint32, ttl uint8, id string, distance float32, notTo *Node, online bool) {
	//log.Print("Distribute UpdateRequest (but not to "+notTo.NodeID+")")
	ret := make(chan []*Node)
	um.Out <- NewEvent("GetNodeList", ret)
	list := <-ret

	packet := make([]byte, 38) //0sssstdddd[28*i]

	if online {
		packet[0] = 1
		//TEST
		/*
				if notTo!=nil {
					log.Print("someone is new online, send hello to all")
					ret := make(chan string)
				        um.Out <- NewEvent("GetGlobalID",ret)
			                myid := <-ret
					um.DistributeUpdate(rand.Uint32(),5,myid,0.0,nil,true)
				}
		*/
		//ENDTEST
	} else {
		packet[0] = 0 // UpdateRequest Marker
	}
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, sid)
	copy(packet[1:5], buf.Bytes()) // SID
	packet[5] = ttl                // TTL
	distuint := math.Float32bits(distance)
	buf = new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, distuint)
	copy(packet[6:10], buf.Bytes()) // DIST
	copy(packet[10:], []byte(id))   // ID

	ret2 := make(chan *tls.Config)
	um.Out <- NewEvent("GetTlsConfig", ret2)
	config := <-ret2

	ergchans := make([]chan bool, len(list))
	act_pos := 0

	um.multilock.Lock(sid)
	for _, node := range list {
		if node != notTo && node.Online == true {
			rchan := make(chan bool)
			ergchans[act_pos] = rchan
			act_pos++
			id := node.NodeID
			hostname := node.Hostname
			port := node.Port
			n := node
			go func() {
				defer func() { rchan <- true }()
				//log.Print("Try to connect to "+id+" ("+hostname+":"+strconv.Itoa(port)+")")
				conn, err := tls.Dial("tcp", hostname+":"+strconv.Itoa(port), config)
				if err != nil {
					//log.Print("Node is offline ("+id+")")
					log.Print("Error connecting Client:")
					log.Print(err)
					um.AnnounceOffline(n)
					return
				} else {
					b, err := conn.Write(packet)
					if err != nil || b != len(packet) {
						log.Print("Error while sending Updatepacket to " + id)
					}
					buff := make([]byte, 1)
					conn.Read(buff)
					conn.Close()
					//log.Print("Success")
				}
			}()
		}
	}
	for i := 0; i < act_pos; i++ {
		<-ergchans[i]
	}

	um.multilock.UnLock(sid)
}

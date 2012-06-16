package main

import (
	"net/http"
	"strconv"
	"html/template"
	"log"
)

type WebFrontend struct {
        In 		chan *Event
        Out 		chan *Event
}
func (p *WebFrontend) PushEvent(event *Event){
        p.In <- event
}
func (p *WebFrontend) PullEvent()*Event{
        return <-p.Out
}

type NodeListContent struct {
	NodeList	[]*Node
}
func (p *WebFrontend) NodeListHandler(w http.ResponseWriter, r *http.Request) {
    t,err := template.ParseFiles("./templates/NodeView.html")
	if err!=nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
        return
	}
    rchan := make(chan []*Node)
    p.Out <- NewEvent("GetNodeList",rchan)
    list := <-rchan
    var content NodeListContent
    content.NodeList = list
    err = t.Execute(w,content)
    if err!= nil {
    	log.Print(err)
    }
}

type WhoViewContent struct {
	WhoList	[]*PersonInfo
}
func (p *WebFrontend) WhoHandler(w http.ResponseWriter, r *http.Request) {
	t,err := template.ParseFiles("./templates/WhoView.html")
	if err!=nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
        return
	}
	ret := make(chan []*PersonInfo)
	p.Out <- NewEvent("GetListOfOnlineNodes2",ret)
	list := <-ret
	var content WhoViewContent
	for _,val := range list {
		content.WhoList = append(content.WhoList,val)
	}
	err = t.Execute(w,content)
	if err!=nil {
		log.Print(err)
	}
}

type MainViewContent struct{
	MyName	string
}
func (p *WebFrontend) MainHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.String() != "/" {
		http.Error(w, "Page not found", http.StatusInternalServerError)
        return
	}
	t,err := template.ParseFiles("./templates/MainView.html")
	if err!=nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
        return
	}
	ret := make(chan string)
	p.Out <- NewEvent("GetGlobalID",ret)
	t.Execute(w,MainViewContent{MyName: <-ret})
}

func (p *WebFrontend) SaveNodeHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	host := r.FormValue("host")
	port := r.FormValue("port")
	_port,err := strconv.ParseInt(port,10,32)
	if err != nil || len(name)!= 28 {
		http.Redirect(w, r, "/view/nodes", http.StatusFound)
		return
	}
	p.Out <- NewEvent("AddNode",NewNode(name,host,int(_port)))
	http.Redirect(w, r, "/view/nodes", http.StatusFound)
}

type WriteMailContent struct {Target string}
func (p *WebFrontend) WriteMailHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.String()[11:]
	t,err := template.ParseFiles("./templates/WriteMail.html")
	if err!=nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
        return
	}
	t.Execute(w,WriteMailContent{Target:target})
}

func (p *WebFrontend) DeleteNodeHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if len(name)!= 28 {
		http.Redirect(w, r, "/view/nodes", http.StatusFound)
		return
	}
	p.Out <- NewEvent("DeleteNode",name)
	http.Redirect(w, r, "/view/nodes", http.StatusFound)
}

func (p *WebFrontend) SendMailHandler(w http.ResponseWriter, r *http.Request) {
	target := r.FormValue("target")
	subject := r.FormValue("subject")
	text := r.FormValue("text")
	if len(target)!= 28 || len(subject)==0 || len(text)==0 {
		http.Error(w, "fail...", http.StatusInternalServerError)
		return
	}
	payload := NewInitialConnectPayload(target,25,5)/*TODO TTL dynamisch*/
	p.Out <- NewEvent("InitialConnectRequest",payload)
	tunnel := <-payload.Return
	defer tunnel.Close()
	if tunnel == nil {
		http.Error(w, "fail... tunnel couldn't be established", http.StatusInternalServerError)
		return
	}
	_,err := tunnel.Write([]byte(subject+"\n"))
	if err!=nil {
		log.Print("Error sending subject")
		http.Error(w, "fail... Error sending subject", http.StatusInternalServerError)
		return
	}
	_,err = tunnel.Write([]byte(text))
	if err!=nil {
		log.Print("Error sending text")
		http.Error(w, "fail... Error sending text", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/who", http.StatusFound)
}

type ShowInboxContent struct {
	List []*Mail
}
func (p *WebFrontend) ShowInboxHandler(w http.ResponseWriter, r *http.Request) {
	t,err := template.ParseFiles("./templates/ShowInbox.html")
	if err!=nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
        return
	}
	pay := NewGetMailListPayload(20)
	p.Out <- NewEvent("GetMailList",pay)
	list := <-pay.Return
	if list==nil {
		list = []*Mail{&Mail{Id:-1,Subject:"No mail!"}}
	}
	t.Execute(w,ShowInboxContent{list})
}

func (p *WebFrontend) ShowMailHandler(w http.ResponseWriter, r *http.Request) {
	t,err := template.ParseFiles("./templates/ShowMail.html")
	if err!=nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
        return
	}
	idstr := r.URL.String()[10:]
	id,err := strconv.ParseInt(idstr,10,32)
	if err!=nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
        return
	}
	pay := NewGetMailPayload(int(id))
	p.Out <- NewEvent("GetMail",pay)
	mail := <-pay.Return
	t.Execute(w,mail)
}

func NewWebFrontend(backlog, port int) *WebFrontend {
	f := new(WebFrontend)
	f.In = make(chan *Event,backlog)
	f.Out= make(chan *Event,backlog)
	
	http.HandleFunc("/view/nodes/",func(w http.ResponseWriter, r *http.Request){
					f.NodeListHandler(w,r)
				})
	http.HandleFunc("/view/who/",func(w http.ResponseWriter, r *http.Request){
					f.WhoHandler(w,r)
				})
	http.HandleFunc("/savenode/",func(w http.ResponseWriter, r *http.Request){
					f.SaveNodeHandler(w,r)
				})
	http.HandleFunc("/deletenode/",func(w http.ResponseWriter, r *http.Request){
					f.DeleteNodeHandler(w,r)
				})
	http.HandleFunc("/writemail/",func(w http.ResponseWriter, r *http.Request){
					f.WriteMailHandler(w,r)
				})
	http.HandleFunc("/sendmail/",func(w http.ResponseWriter, r *http.Request){
					f.SendMailHandler(w,r)
				})
	http.HandleFunc("/view/inbox/",func(w http.ResponseWriter, r *http.Request){
					f.ShowInboxHandler(w,r)
				})
	http.HandleFunc("/showmail/",func(w http.ResponseWriter, r *http.Request){
					f.ShowMailHandler(w,r)
				})
	http.HandleFunc("/",func(w http.ResponseWriter, r *http.Request){
					f.MainHandler(w,r)
				})
	
	go http.ListenAndServeTLS(":"+strconv.FormatInt(int64(port),10),"cert.pem","key.pem",nil)
	return f
}



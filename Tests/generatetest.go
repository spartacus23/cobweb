package main

import (
	"flag"
	"fmt"
	"text/template"
	"crypto/rand"
	"math/big"
	"os"
	"os/exec"
	"strconv"
)

var Instances *int = flag.Int("n",10,"Number of Instances") 
var Round	  *int = flag.Int("r",3,"Round 1 or 2")

func GenerateClearMatrix(num int) [][]bool {
	matrix := make([][]bool,num)
	for idx,_ := range matrix {
		matrix[idx] = make([]bool,num)
	}
	return matrix
}

func FillRandom(matrix [][]bool) [][]bool {
	num := len(matrix)
	for zeile,_ := range matrix {
		fehlende := 5
		for i:=0;i<len(matrix);i++ {
			if matrix[zeile][i]==true {
				fehlende-=1
			}
		}
		for i:=0;i<fehlende;i++ {
			pos,_ := rand.Int(rand.Reader,big.NewInt(int64(num)))
			p := pos.Int64()
			if p!=int64(zeile) {
				matrix[zeile][p] = true
				matrix[p][zeile] = true
			
			}
		}
	}
	return matrix
}

func GenerateDirs(num int){
	os.Mkdir("clients",0700)
	for i:=0;i<num;i++ {
		dirname := fmt.Sprintf("clients/client%04v",i)
		os.Mkdir(dirname,0700)
	}
}

type NodeInfo struct {
	Id		string
	Host	string
	Port	string
}

type ConfigFile struct {
	Port			int64
	FrontendPort	int64
	Nodes			[]*NodeInfo
}

func GenerateBasicConfigfile(num int) {
	for i:=0;i<num;i++ {
		numstr := fmt.Sprintf("%04v",i)
		configname := fmt.Sprintf("clients/client%04v/cobweb.conf",i)
		file,err := os.Create(configname)
		if err!=nil {
			fmt.Printf("Error while creating %v\n",configname)
			fmt.Printf("%v\n",err)
			continue
		}
		t,_ := template.ParseFiles("configtemplate.txt")
		portstr := "1"+numstr
		frontstr := "2"+numstr
		p,_ := strconv.ParseInt(portstr,10,32)
		f,_ := strconv.ParseInt(frontstr,10,32)
		t.Execute(file,ConfigFile{p,f,nil})
		file.Close()
	}
}

func Copy(num int) {
	for i:=0;i<num;i++ {
		dirname := fmt.Sprintf("./clients/client%04v/cobweb",i)
		cmd := exec.Command("cp","-f","../Backend/bin/cobweb_backend",dirname)
		cmd.Start()
		cmd.Wait()
	}
	fmt.Printf("All copied\n")
}

func CopyTemplatesAndPortmap(num int){
	for i:=0;i<num;i++ {
		dirname := fmt.Sprintf("clients/client%04v/",i)
		cmd:= exec.Command("cp","-rf","../Backend/src/templates/","portmapping.txt",dirname)
		cmd.Start()
		cmd.Wait()
	}
}

func RunAll(num int){
	for i:=0;i<num;i++ {
		dirname := fmt.Sprintf("clients/client%04v/",i)
		os.Chdir(dirname)
		cmd := exec.Command("xterm","-e","./cobweb")
		cmd.Start()
		os.Chdir("../../")
	}
}

func RoundOne(){
	GenerateDirs(*Instances)
	fmt.Printf("All %v directories created...\n",*Instances)
	GenerateBasicConfigfile(*Instances)
	fmt.Printf("All %v configfiles created...\n",*Instances)
	Copy(*Instances)
	CopyTemplatesAndPortmap(*Instances)
	RunAll(*Instances)
	fmt.Printf("Everythings running, wait for it to finish and killall cobweb\n")
}

func ReadIdSlice(num int) []string{
	ids := make([]string,num)
	for idx:=0;idx<num;idx++ {
		filename := fmt.Sprintf("clients/client%04v/myid.txt",idx)
		buf := make([]byte,28)
		file,err := os.Open(filename)
		if err!=nil {
			fmt.Printf("failed to open id file... run round one?!\n")
			return nil
		}
		n,err := file.Read(buf)
		if n!=28 || err!=nil {
			fmt.Println("fail... line: ",buf)
			continue
		}
		ids[idx]=string(buf)
	}
	return ids
}

func PrintMatrix(matrix [][]bool){
	for x,_ := range matrix {
		for y,_ := range matrix[x] {
			if matrix[x][y] {
				fmt.Printf("1 ")
			}else{
				fmt.Printf("0 ")
			}
			
		}
		fmt.Printf("\n")
	}
}

func GenerateNewConfigfiles(num int,matrix [][]bool,ids []string){
	for i:=0;i<num;i++ {
		numstr := fmt.Sprintf("%04v",i)
		configname := fmt.Sprintf("clients/client%04v/cobweb.conf",i)
		file,err := os.Create(configname)
		if err!=nil {
			fmt.Printf("Error while creating %v\n",configname)
			fmt.Printf("%v\n",err)
			continue
		}
		t,_ := template.ParseFiles("configtemplate.txt")
		portstr := "1"+numstr
		frontstr := "2"+numstr
		p,_ := strconv.ParseInt(portstr,10,32)
		f,_ := strconv.ParseInt(frontstr,10,32)
		nodeinfos := make([]*NodeInfo,0)
		for idx,val := range matrix[i] {
			if val==true {
				partnerport := fmt.Sprintf("1%04v",idx)
				partnerid := ids[idx]
				nodeinfos = append(nodeinfos,&NodeInfo{partnerid,"localhost",partnerport})
			}
		}
		t.Execute(file,ConfigFile{p,f,nodeinfos})
		file.Close()
	}
}

func RoundTwo(){
	ids := ReadIdSlice(*Instances)
	matrix := GenerateClearMatrix(*Instances)
	FillRandom(matrix)
	PrintMatrix(matrix)
	GenerateNewConfigfiles(*Instances,matrix,ids)
	//Copy(*Instances)
	//RunAll(*Instances)
}

func main(){	
	flag.Parse()
	if *Round == 1 {
		RoundOne()
	}else if *Round==2 {
		RoundTwo()
	}else{
		RunAll(*Instances)
	}

}

package main

import (
	"fmt"
	"flag"
	"os"
	"encoding/gob"
	"math/rand"
	"time"
	"./plotter"
)

var idstr *string = flag.String("id","123456","idstring des testcases")


type TestCaseData struct {
	Id		string
	Points	[]*plotter.Point
	Matrix	[][]int
}

func LoadData(id string)*TestCaseData {
	result := new(TestCaseData)
	f,_ := os.Open(id+".gob")
	decoder := gob.NewDecoder(f)
	decoder.Decode(&result)
	return result
}

func GetAllInDistance(matrix [][]int, id,dist int) []int {
	if dist<=0 {
		return []int{id}
	}
	erg := make([]int,0)
	for idx,val := range matrix[id] {
		if val==1 {
			erg = append(erg,GetAllInDistance(matrix,idx,dist-1)...)
		}
	}
	erg = append(erg,id)
	e := make([]int,0)
	for _,val := range erg {
		isin := false
		for _,v := range e {
			if val==v {
				isin = true
				break
			}
		}
		if !isin {
			e=append(e,val)
		}
	}
	return e
}

func main(){
	rand.Seed(time.Now().Unix())
	flag.Parse()
	data := LoadData(*idstr)
	availmap := make(map[int]float64)
	for i:=5;i<8;i++ {
		avail := 0.0
		sample := len(data.Points)/10
		for j:=0;j<sample;j++ {
			idx := rand.Intn(len(data.Points))
			fmt.Println(j,"/",sample)
			avail += (float64(len(GetAllInDistance(data.Matrix,idx,i)))/float64(len(data.Points)))*100
		}
		avail = avail/float64(sample)
		fmt.Println("Avail in",i,":",avail,"%")
		availmap[i]=avail
		f,_ := os.Create(*idstr+".avail.gob")
		encoder := gob.NewEncoder(f)
		encoder.Encode(availmap)
		f.Close()	
	}
}



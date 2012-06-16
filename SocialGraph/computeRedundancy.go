package main

import (
	"fmt"
	"flag"
	"os"
	"encoding/gob"
	"math"
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

func calcPathesFromTo(from,to int,matrix [][]int,ttl int) [][]int {
	if from==to {
		erg := make([][]int,1)
		erg[0] = make([]int,0)
		return erg
	}
	if ttl<=0 {
		return nil
	}
	erg := make([][]int,0)
	for idx,val := range matrix[from] {
		if val==1 {
			pathes := calcPathesFromTo(idx,to,matrix,ttl-1)
			for _,path := range pathes {
				path = append(path,from)
				erg = append(erg,path)
			}
		}
	}
	return erg
}

func IndependentPathes(pathes [][]int) [][]int {
	erg := make([][]int,0)
	for _,path := range pathes {
		completeNew := true
		for _,p:=range erg {
			for iidx,i := range p {
				if iidx==len(p)-1{
					continue
				}
				for _,j:=range path {
					if i==j {
						completeNew = false
						break
					}
				}
			}
		}
		if completeNew {
			erg = append(erg,path)
		}
	}
	return erg
}

func Distance(indi1,indi2 *plotter.Point) float64 {
	return math.Sqrt((indi1.X-indi2.X)*(indi1.X-indi2.X)+(indi1.Y-indi2.Y)*(indi1.Y-indi2.Y))
}

func main(){
	rand.Seed(time.Now().Unix())
	flag.Parse()
	data := LoadData(*idstr)
	avail := 0
	sample := len(data.Points)/10
	for j:=0;j<sample;j++ {
		//sample a few routes, prefere close routes
		idx1 := rand.Intn(len(data.Points))
		target_x := data.Points[idx1].X + rand.NormFloat64() * 150
		target_y := data.Points[idx1].Y + rand.NormFloat64() * 150
		idx2 := 0
		mindist := 999999.0
		for id,val := range data.Points {
			if id==idx1 {
				continue
			}
			dist := Distance(&plotter.Point{target_x,target_y},val)
			if dist < mindist {
				mindist = dist
				idx2 = id
			}
		}
		
		pathes := calcPathesFromTo(idx1,idx2,data.Matrix,6)
		indep := IndependentPathes(pathes)
		fmt.Println(j,"/",sample)
		avail += len(indep)
	}
	a := float64(avail)/float64(sample)
	f,_ := os.Create(*idstr+".indepPathes.txt")
	f.Write([]byte(fmt.Sprintf("%v\n",a)))
	fmt.Println(a)
}


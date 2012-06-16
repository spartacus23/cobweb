package main

import (
	"fmt"
	"flag"
	"encoding/gob"
	"os"
	
)

var idstr *string = flag.String("id","123456","idstring des testcases")


func LoadData(id string) map[int]float64 {
	result := make(map[int]float64)
	f,_ := os.Open(id+".avail.gob")
	decoder := gob.NewDecoder(f)
	decoder.Decode(&result)
	return result
}

func main(){
	flag.Parse()
	m := LoadData(*idstr)
	fmt.Println(*idstr)
	outstr := fmt.Sprintf("%v %v %v",m[5],m[6],m[7])
	fmt.Println(outstr)
}

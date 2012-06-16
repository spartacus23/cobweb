package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"time"

	"./plotter"
)

var AnzCluster *int64 = flag.Int64("cluster", 20, "Anzahl der Cluster der Flächenverteilung")
var ClusterBreite *float64 = flag.Float64("clusterdev", 50.0, "Breite der Clusters (Standartabweichung)")
var FieldSize *float64 = flag.Float64("maxsize", 800.0, "Breite des Feldes in dem die Cluster sind")

var Indis *int64 = flag.Int64("indis", 1000, "Anzahl individuen")
var AnzFriends *int64 = flag.Int64("friends", 6, "Anzahl freunde pro plotter.Point")
var FriendClusterBreite *float64 = flag.Float64("frienddev", 150.0, "Breite der Clusters um den Freund")

var Count *int64 = flag.Int64("count", 1, "Anzahl zu generierender Testfälle")
var Show *bool = flag.Bool("show", false, "Show the image?")

func CreateClusterCenter(max float64) (float64, float64) {
	x := rand.NormFloat64()*(max/4) + (max / 2)
	y := rand.NormFloat64()*(max/4) + (max / 2)
	return x, y
}

func CreateIndis(anzindis int64, stddev, center_x, center_y float64) []*plotter.Point {
	list := make([]*plotter.Point, 0, anzindis)
	for i := 0; i < int(anzindis); i++ {
		indi := new(plotter.Point)
		indi.X = (rand.NormFloat64() * stddev) + center_x
		indi.Y = (rand.NormFloat64() * stddev) + center_y
		list = append(list, indi)
	}
	return list
}

func CreateAllIndis() []*plotter.Point {
	list := make([]*plotter.Point, 0, *Indis)
	indispercluster := int64(*Indis / *AnzCluster)
	fmt.Println("plotter.Points per cluster:", indispercluster)
	for i := 0; i < int(*AnzCluster); i++ {
		x, y := CreateClusterCenter(*FieldSize)
		fmt.Println("Center", i, ":", x, y)
		list = append(list, CreateIndis(indispercluster, *ClusterBreite, x, y)...)
	}
	return list
}

func Distance(indi1, indi2 *plotter.Point) float64 {
	return math.Sqrt((indi1.X-indi2.X)*(indi1.X-indi2.X) + (indi1.Y-indi2.Y)*(indi1.Y-indi2.Y))
}

func CreateMatrix2(indis []*plotter.Point) [][]int {
	anz := len(indis)
	matrix := make([][]int, anz)
	for i := 0; i < anz; i++ {
		matrix[i] = make([]int, anz)
	}
	for tmp := 0; int64(tmp) < (*Indis)*(*AnzFriends); tmp++ {
		indi := rand.Intn(anz)
		anzfriends := 0
		for _, val := range matrix[indi] {
			if val == 1 {
				anzfriends++
			}
		}
		if int64(anzfriends) < (*AnzFriends) {
			target_x := indis[indi].X + rand.NormFloat64()*(*FriendClusterBreite)
			target_y := indis[indi].Y + rand.NormFloat64()*(*FriendClusterBreite)
			friendid := 0
			mindist := 999999.0
			for id, val := range indis {
				if id == indi {
					continue
				}
				dist := Distance(&plotter.Point{target_x, target_y}, val)
				if dist < mindist {
					mindist = dist
					friendid = id
				}
			}
			friendfriends := 0
			for _, val := range matrix[friendid] {
				if val == 1 {
					friendfriends++
				}
			}
			if int64(friendfriends) < (*AnzFriends) {
				matrix[indi][friendid] = 1
				matrix[friendid][indi] = 1
			}
		}
	}
	return matrix
}

func Normiere(indis []*plotter.Point) (float64, float64) {
	minx := indis[0].X
	miny := indis[0].Y
	maxx := indis[0].X
	maxy := indis[0].Y

	for _, val := range indis {
		if val.X < minx {
			minx = val.X
		}
		if val.Y < miny {
			miny = val.Y
		}
		if val.X > maxx {
			maxx = val.X
		}
		if val.Y > maxy {
			maxy = val.Y
		}
	}

	fmt.Println("minx: ", minx)
	var xadd float64 = 0
	if minx < 0 {
		xadd = math.Abs(minx)
	}
	fmt.Println("xadd: ", xadd)
	fmt.Println("miny: ", miny)
	var yadd float64 = 0
	if miny < 0 {
		yadd = math.Abs(miny)
	}
	fmt.Println("yadd: ", yadd)
	for idx := range indis {
		indis[idx].X += xadd
		indis[idx].Y += yadd
	}
	return (maxx + xadd), (maxy + yadd)
}

func GetAllInDistance(matrix [][]int, id, dist int) []int {
	if dist <= 0 {
		return []int{id}
	}
	erg := make([]int, 0)
	for idx, val := range matrix[id] {
		if val == 1 {
			erg = append(erg, GetAllInDistance(matrix, idx, dist-1)...)
		}
	}
	erg = append(erg, id)
	e := make([]int, 0)
	for _, val := range erg {
		isin := false
		for _, v := range e {
			if val == v {
				isin = true
				break
			}
		}
		if !isin {
			e = append(e, val)
		}
	}
	return e
}

func GenerateTestCase() ([]*plotter.Point, [][]int, string) {
	idstr := strconv.FormatInt(time.Now().Unix(), 10)
	indis := CreateAllIndis()
	sizex, sizey := Normiere(indis)
	fmt.Println("created indis")
	matrix := CreateMatrix2(indis)
	fmt.Println("created matrix")
	plotter.Plot(int(math.Floor(sizex)), int(math.Ceil(sizey)), indis, matrix, idstr)
	SaveData(&TestCaseData{idstr, indis, matrix})
	if *Show {
		ShowImage(idstr + ".png")
	}
	return indis, matrix, idstr
}

type TestCaseData struct {
	Id     string
	Points []*plotter.Point
	Matrix [][]int
}

func SaveData(data *TestCaseData) {
	f, _ := os.Create(data.Id + ".gob")
	encoder := gob.NewEncoder(f)
	encoder.Encode(data)
	f.Close()
}
func LoadData(id string) *TestCaseData {
	result := new(TestCaseData)
	f, _ := os.Open(id + ".gob")
	decoder := gob.NewDecoder(f)
	decoder.Decode(&result)
	return result
}

func ShowImage(path string) {
	cmd := exec.Command("qiv", "-fm", path)
	cmd.Run()
}

func main() {
	rand.Seed(time.Now().Unix())
	flag.Parse()
	for i := 0; int64(i) < *Count; i++ {
		GenerateTestCase()
	}
	//fmt.Println(idstr)
	/* indis,matrix,idstr := 
	for i:=5;i<8;i++ {
		avail := 0.0
		for j:=0;j<20;j++ {
			id := rand.Intn(int(len(indis)))
			avail += (float64(len(GetAllInDistance(matrix,id,i)))/float64(*Indis))*100
		}
		avail = avail/float64(20)
		fmt.Println("Avail in",i,":",avail,"%")	
	}
	*/
}

//-cluster=15 -clusterdev=50 -indis=1000 -frienddev=150 -friends=6 -maxsize=800
//go run matrixtest.go -cluster=10 -clusterdev=50 -indis=1500 -frienddev=180 -friends=6 -maxsize=800 && qiv -fm TestPath.png 

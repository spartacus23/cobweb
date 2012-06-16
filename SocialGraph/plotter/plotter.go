package plotter

import (
        "bufio"
        "fmt"
        "log"
        "os"
		"math"

        "code.google.com/p/draw2d/draw2d"
        "image"
        "image/png"
        "image/color"
)

type Point struct {
	X	float64
	Y	float64
}

func saveToPngFile(filePath string, m image.Image) {
        f, err := os.Create(filePath)
        if err != nil {
                log.Println(err)
                os.Exit(1)
        }
        defer f.Close()
        b := bufio.NewWriter(f)
        err = png.Encode(b, m)
        if err != nil {
                log.Println(err)
                os.Exit(1)
        }
        err = b.Flush()
        if err != nil {
                log.Println(err)
                os.Exit(1)
        }
        fmt.Printf("Wrote %s OK.\n", filePath)
}

func WhiteBG(gc *draw2d.ImageGraphicContext,xsize,ysize float64){
    gc.BeginPath()
    gc.MoveTo(0,0)
    gc.LineTo(xsize,0)
    gc.LineTo(xsize, ysize)
    gc.LineTo(0,ysize)
    gc.LineTo(0,0)
    gc.Close()
    gc.SetFillColor(color.RGBA{255, 255, 255,  0xff})
	gc.Fill()

}

func DrawCircle(gc *draw2d.ImageGraphicContext,x,y float64){
    gc.ArcTo(x, y, 3, 3, 0, 2*math.Pi)
    gc.SetFillColor(color.RGBA{255,0,0,255})
    gc.FillStroke()
}

func DrawLine(gc *draw2d.ImageGraphicContext,x1,y1,x2,y2 float64){
	gc.MoveTo(x1,y1)
	gc.LineTo(x2,y2)
	gc.FillStroke()
}

func Plot(xsize,ysize int, indis []*Point, matrix [][]int,name string) {
    img := image.NewRGBA(image.Rect(0, 0, xsize, ysize))
    gc := draw2d.NewGraphicContext(img)
    WhiteBG(gc,float64(xsize),float64(ysize))    
    
    for i:=0;i<len(matrix);i++ {
    	for j:=0;j<i;j++ {
    		if matrix[i][j]==1 {
    			DrawLine(gc,indis[i].X,indis[i].Y,indis[j].X,indis[j].Y)
    		}
    	}
    }
    for _,val := range indis {
    	DrawCircle(gc,val.X,val.Y)
    }
    
    saveToPngFile(name+".png", img)
}


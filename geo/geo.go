package geo

import (
	"fmt"
	"math"
)

// BBox is a simple box struct with optional SRID
type BBox struct {
	Minx, Miny, Maxx, Maxy float64
	Srid                   int
}

func (b *BBox) GoString() string {
	return fmt.Sprintf("BBox [minx = %.5f, minxy = %.5f, maxx = %.5f, maxy = %.5f, SRID = %d]", b.Minx, b.Miny, b.Maxx, b.Maxy, b.Srid)
}

// NewBBox will convert an x/y/z coordinate to a 3857 bounding box.
func NewBBox(x, y, z int) *BBox {
	north, west := ll(x, y, z)
	south, east := ll(x+1, y+1, z)

	maxx, maxy, _ := merc(north, east)
	minx, miny, _ := merc(south, west)

	// invert the maxy/miny here as 3857 is in screen coords, positive 'y' downwards
	return &BBox{Minx: minx, Miny: maxy, Maxx: maxx, Maxy: miny, Srid: 3857}
}

func merc(lat, long float64) (x, y float64, err error) {
	//http://www.maptiler.org/google-maps-coordinates-tile-bounds-projection/
	if math.Abs(long) > 180 {
		return 0, 0, fmt.Errorf("longitude is out of bounds - range is [-180 +180]: longitude = %f", long)
	}

	if math.Abs(lat) > 90 {
		return 0, 0, fmt.Errorf("latitude is out of bounds - range is [-90 +90]: latitude = %f", long)
	}

	x = long * 20037508.342789244 / 180
	y = math.Log(math.Tan((90+lat)*math.Pi/360.0)) / (math.Pi / 180.0)
	y = y * 20037508.342789244 / 180.0

	return x, y, nil
}

func ll(x, y, z int) (lat, long float64) {
	n := math.Pi - 2.0*math.Pi*float64(y)/math.Exp2(float64(z))
	lat = 180.0 / math.Pi * math.Atan(0.5*(math.Exp(n)-math.Exp(-n)))
	long = float64(x)/math.Exp2(float64(z))*360.0 - 180.0
	return lat, long

}

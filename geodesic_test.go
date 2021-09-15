package geodesic

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"os"
	"testing"
	"time"
)

const testDataPath = "test.data"

func TestGenerateData(t *testing.T) {
	if os.Getenv("GENDATA") != "1" {
		fmt.Printf("Use \"GENDATA=1 go test\" to generate a new test.data file.\n")
		return
	}
	var data []byte
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 5000; i++ {
		lat1 := rand.Float64()*180 - 90
		lon1 := rand.Float64()*360 - 180
		lat2 := rand.Float64()*180 - 90
		lon2 := rand.Float64()*360 - 180
		var s12, azi1, azi2 float64
		WGS84.Inverse(lat1, lon1, lat2, lon2, &s12, &azi1, &azi2)
		data = append(data, 'I')
		data = appendFloats(data, lat1, lon1, lat2, lon2, s12, azi1, azi2)
	}
	for i := 0; i < 100; i++ {
		lat1 := rand.Float64()*180 - 90
		lon1 := rand.Float64()*360 - 180
		steps := rand.Intn(10) + 4        // 4 - 14 steps
		dist := rand.Float64()*20000 + 10 // 10-20000 kilometers
		p := WGS84.PolygonInit(false)
		data = append(data, 'P')
		mark := len(data)
		data = append(data, 0)
		// generate the circle
		for azi := 0.0; azi <= 360.0; azi += 360.0 / float64(steps) {
			var lat2, lon2 float64
			WGS84.Direct(lat1, lon1, azi, dist, &lat2, &lon2, nil)
			p.AddPoint(lat2, lon2)
			data = appendFloats(data, lat2, lon2)
			data[mark]++
		}
		var area, peri float64
		p.Compute(false, false, &area, &peri)
		data = appendFloats(data, area, peri)
		p.Compute(true, false, &area, &peri)
		data = appendFloats(data, area, peri)
		p.Compute(true, true, &area, &peri)
		data = appendFloats(data, area, peri)
		p.Compute(false, true, &area, &peri)
		data = appendFloats(data, area, peri)
	}

	if err := os.WriteFile("test.data", data, 0666); err != nil {
		t.Fatal(err)
	}

}

func appendFloats(dst []byte, x ...float64) []byte {
	for _, x := range x {
		dst = append(dst, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.LittleEndian.PutUint64(dst[len(dst)-8:], math.Float64bits(x))
	}
	return dst
}
func eqish(x, y float64, prec int) bool {
	return math.Abs(x-y) < float64(1.0)/math.Pow10(prec)
}

func readFloats(src []byte, count int) []float64 {
	vals := make([]float64, count)
	for i := 0; i < count; i++ {
		vals[i] = math.Float64frombits(binary.LittleEndian.Uint64(src[i*8:]))
	}
	return vals
}

func TestInput(t *testing.T) {
	data, err := os.ReadFile(testDataPath)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(data); {
		if data[i] == 'I' {
			v := readFloats(data[i+1:], 7)
			i += 1 + 7*8
			testInverse(t, v[0], v[1], v[2], v[3], v[4], v[5], v[6])
			testDirect(t, v[0], v[1], v[2], v[3], v[4], v[5], v[6])
		} else if data[i] == 'P' {
			i++
			npoints := int(data[i])
			i++
			points := readFloats(data[i:], npoints*2)
			i += npoints * 2 * 8
			vals := readFloats(data[i:], 8)
			i += 8 * 8
			testPolygon(t, points, vals)
		} else {
			t.Fatalf("invalid %s", testDataPath)
		}
	}
}

func testPolygon(t *testing.T, points []float64, vals []float64) {
	p := WGS84.PolygonInit(false)
	for i := 0; i < len(points); i += 2 {
		p.AddPoint(points[i+0], points[i+1])
	}
	retvals := make([]float64, 8)
	p.Compute(false, false, &retvals[0], &retvals[1])
	p.Compute(true, false, &retvals[2], &retvals[3])
	p.Compute(true, true, &retvals[4], &retvals[5])
	p.Compute(false, true, &retvals[6], &retvals[7])
	for i := 0; i < len(vals); i++ {
		if !eqish(vals[i], retvals[i], 3) {
			t.Fatalf("expected %f, got %f", vals, retvals)
		}
	}
}

func testInverse(t *testing.T, lat1, lon1, lat2, lon2, s12, azi1, azi2 float64) {
	var s12ret, azi1ret, azi2ret float64
	WGS84.Inverse(lat1, lon1, lat2, lon2, &s12ret, &azi1ret, &azi2ret)
	if !eqish(s12ret, s12, 7) || !eqish(azi1ret, azi1, 7) || !eqish(azi2ret, azi2, 7) {
		t.Fatalf("expected '%f, %f, %f', got '%f, %f, %f'",
			s12, azi1, azi2, s12ret, azi1ret, azi2ret)
	}
}

func testDirect(t *testing.T, lat1, lon1, lat2, lon2, s12, azi1, azi2 float64) {
	var lat2ret, lon2ret, azi2ret float64
	WGS84.Direct(lat1, lon1, azi1, s12, &lat2ret, &lon2ret, &azi2ret)
	if !eqish(lat2ret, lat2, 7) || !eqish(lon2ret, lon2, 7) || !eqish(azi2ret, azi2, 7) {
		t.Fatalf("expected '%f, %f, %f', got '%f, %f, %f'",
			lat2, lon2, azi2, lat2ret, lon2ret, azi2ret)
	}
}

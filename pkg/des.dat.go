package pkg

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"

	"gonum.org/v1/gonum/stat" // go get gonum.org/v1/gonum/...
)

/*BYTES OUTPUT*/
func GetBytes(v any) []byte {
	var buffer bytes.Buffer
	err := binary.Write(&buffer, binary.BigEndian, v)
	if err != nil {
		fmt.Println(err)
	}
	return buffer.Bytes()
}


/*BYTES INPUT*/
func BytesToUInt16(bytes []byte) uint16 {
	x := make([]byte, 2)
	i := len(x) - len(bytes)
	n := len(bytes) - 1
	// fmt.Println("Received bytes:\t", n)
	for n >= 0 {
		x[n+i] = bytes[n]
		n--
	}
	// fmt.Printf("Final bytes:\t%d\t%x\n", len(x), x)
	return binary.BigEndian.Uint16(x)
}
func BytesToUInt16_L(bytes []byte) uint16 {
	x := make([]byte, 2)
	// fmt.Println("Received bytes:\t", len(bytes))
	for i, v := range bytes {
		x[i] = v
	}
	// fmt.Printf("Final bytes:\t%d\t%x\n", len(x), x)
	return binary.LittleEndian.Uint16(x)
}

func BytesToUInt32(bytes []byte) uint32 {
	x := make([]byte, 4)
	i := len(x) - len(bytes)
	n := len(bytes) - 1
	// fmt.Println("Received bytes:\t", n)
	for n >= 0 {
		x[n+i] = bytes[n]
		n--
	}
	// fmt.Printf("Final bytes:\t%d\t%x\n", len(x), x)
	return binary.BigEndian.Uint32(x)
}
func BytesToUInt32_L(bytes []byte) uint32 {
	x := make([]byte, 4)
	// fmt.Println("Received bytes:\t", len(bytes))
	for i, v := range bytes {
		x[i] = v
	}
	// fmt.Printf("Final bytes:\t%d\t%x\n", len(x), x)
	return binary.LittleEndian.Uint32(x)
}

func BytesToInt64(bytes []byte) int64 {
	return int64(binary.BigEndian.Uint64(bytes))
}

func BytesToFloat32(bytes []byte) float32 {
	return math.Float32frombits(BytesToUInt32(bytes))
}
func BytesToFloat32_L(bytes []byte) float32 {
	return math.Float32frombits(BytesToUInt32_L(bytes))
}

func BytesToBase64(bytes []byte) string {
	// usage := BytesToBase64([]byte("whatever"))
	str := base64.StdEncoding.EncodeToString(bytes)
	// fmt.Println(str)
	return str
}
func Base64ToBytes(b64 string) []byte {
	// usage := Base64ToBytes("FFFFFFFF")
	bytes, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		fmt.Println(err)
	}
	return bytes
}

// func Float32ToHex(f float32)

/*STRING INPUT*/
func StringToInt64(str string) int64 {
	out, err := strconv.ParseInt(strings.Trim(str, " "), 0, 64)
	if err != nil {
		pc, file, line, _ := runtime.Caller(1)
		name := runtime.FuncForPC(pc).Name()
		fmt.Printf("***ERROR***\nFile:\t%s\nFunc  :\t%s\nLine  :\t%d\nError :\n%s", file, name, line, err.Error())
		return 0
	}
	return out
}
func StringToInt32(str string) int32 {
	return int32(StringToInt64(str))
}

func StringToFloat64(str string) float64 {
	out, err := strconv.ParseFloat(strings.Trim(str, " "), 32)
	if err != nil {
		pc, file, line, _ := runtime.Caller(1)
		name := runtime.FuncForPC(pc).Name()
		fmt.Printf("***ERROR***\nFile:\t%s\nFunc  :\t%s\nLine  :\t%d\nError :\n%s", file, name, line, err.Error())
		return 0
	}
	return out
}
func StringToFloat32(str string) float32 {
	return float32(StringToFloat64(str))
}

func MinMaxUInt32(slice []uint32) (uint32, uint32) {

	min := slice[0]
	max := slice[0]
	for _, v := range slice {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	return min, max
}

func MinMaxFloat32(slice []float32, margin float32) (float32, float32) {

	min := slice[0]
	max := slice[0]
	for _, v := range slice {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	span := max - min
	min -= span * margin
	max += span * margin
	// fmt.Printf("MIN: %f, MAX: %f\n", min, max)
	return min, max
}

func MeanFloat32(slice []float32) float32 {
	var mean float32
	for _, val := range slice {
		mean += val
	}
	mean = mean / float32(len(slice))
	return mean
}

func SlopeAndIntercept(x []float32, y []float32) (float32, float32) {
	var (
		SP, SSx, m, b float32
	)

	Mx := MeanFloat32(x)
	My := MeanFloat32(y)

	for i, val := range x {
		SP += val * y[i]
		SSx += (val - Mx) * (val - Mx)
	}
	m = SP / SSx
	b = My - (m * Mx)
	return m, b
}

func MeanStdDev(iArr []float32) (mean, std float64) {
	var arr []float64

	for _, v := range iArr {
		arr = append(arr, float64(v))
	}

	return stat.MeanStdDev(arr, nil)
}

type TSXY struct {
	X []int64
	Y []float32
}

func (v TSXY) TSXs() []int64 {
	return v.X
}
func (v TSXY) TSYs() []float32 {
	return v.Y
}
func (v TSXY) MinMax(margin float32) (float32, float32) {
	return MinMaxFloat32(v.TSYs(), margin)
}
func (v TSXY) TSD(margin float32) TimeSeriesData {
	min, max := v.MinMax(margin)
	return TimeSeriesData{ TSDPoints(v), min, max }
}

type TSValues interface {
	TSXs() []int64
	TSYs() []float32
}
type TSDPoint struct {
	X int64   `json:"x"`
	Y float32 `json:"y"`
}

func TSDPoints(v TSValues) []TSDPoint {
	xs, ys := v.TSXs(), v.TSYs()
	points := []TSDPoint{}
	for i, x := range xs {
		point := TSDPoint{}
		point.X = x
		point.Y = ys[i]
		points = append(points, point)
	}

	return points
}

type TimeSeriesData struct {
	Data []TSDPoint `json:"data"`
	Min  float32    `json:"min"`
	Max  float32    `json:"max"`
}

/* Data Exchange Server (DES) is a component of the Datacan Data2Desk (D2D) Platform.
License:

	[PROPER LEGALESE HERE...]

	INTERIM LICENSE DESCRIPTION:
	In spirit, this license:
	1. Allows <Third Party> to use, modify, and / or distributre this software in perpetuity so long as <Third Party> understands:
		a. The software is porvided as is without guarantee of additional support from DataCan in any form.
		b. The software is porvided as is without guarantee of exclusivity.

	2. Prohibits <Third Party> from taking any action which might interfere with DataCan's right to use, modify and / or distributre this software in perpetuity.
*/

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
func GetBytes_B(v any) []byte {
	var buffer bytes.Buffer
	err := binary.Write(&buffer, binary.BigEndian, v)
	if err != nil {
		fmt.Println(err)
	}
	return buffer.Bytes()
}
func GetBytes_L(v any) []byte {
	var buffer bytes.Buffer
	err := binary.Write(&buffer, binary.LittleEndian, v)
	if err != nil {
		fmt.Println(err)
	}
	return buffer.Bytes()
}


/*BYTES INPUT*/
func BytesToUInt16_B(bytes []byte) uint16 {
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

func BytesToUInt32_B(bytes []byte) uint32 {
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
func BytesToInt32_L(bytes []byte) int32 {
	return int32(BytesToUInt32_L(bytes))
}

func BytesToInt64_B(bytes []byte) int64 {
	return int64(binary.BigEndian.Uint64(bytes))
}
func BytesToInt64_L(bytes []byte) int64 {
	return int64(binary.LittleEndian.Uint64(bytes))
}
func BytesToUint64_L(bytes []byte) uint64 {
	return binary.LittleEndian.Uint64(bytes)
}

func BytesToFloat32_B(bytes []byte) float32 {
	return math.Float32frombits(BytesToUInt32_B(bytes))
}
func BytesToFloat32_L(bytes []byte) float32 {
	return math.Float32frombits(BytesToUInt32_L(bytes))
}
func BytesToFloat64_L(bytes []byte) float64 {
	return math.Float64frombits(BytesToUint64_L(bytes))
}

func BytesToBase64(bytes []byte) string {
	str := base64.StdEncoding.EncodeToString(bytes)
	return str
}
func Base64ToBytes(b64 string) []byte {
	bytes, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		fmt.Println(err)
	}
	return bytes
}

func BytesToBase64URL(bytes []byte) string {
	str := base64.URLEncoding.EncodeToString(bytes)
	return str
}
func Base64URLToBytes(b64 string) []byte {
	bytes, err := base64.URLEncoding.DecodeString(b64)
	if err != nil {
		fmt.Println(err)
	}
	return bytes
}

func Int64ToBytes(in int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(in))
	return b
}

func Int32ToBytes(in int32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(in))
	return b
}

func Int16ToBytes(in int16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(in))
	return b
}

func Float32ToBytes(in float32) []byte {

	var b bytes.Buffer
	if err := binary.Write(&b, binary.LittleEndian, in); err != nil {
		TraceErr(err)
	}
	return b.Bytes()
}

func Float64ToBytes(in float64) []byte {

	var b bytes.Buffer
	if err := binary.Write(&b, binary.LittleEndian, in); err != nil {
		TraceErr(err)
	}
	return b.Bytes()
}

// func StrBytesToString(b []byte) (out string) {
// 	for i := range b {
// 		if b[i] != 255 {
// 			return string(b[i:])
// 		}
// 	}
// 	return
// }

func StrBytesToString(b []byte) (out string) {
	for i := range b {
		if b[i] == 32 {
			return string(b[:i])
		}
	}
	return
}

/* STRING INPUT */
func StringToNBytes(str string, size int) []byte {

	bin := []byte(str)
	l := len(bin)

	if l == size {
		/* bin ALREADY THE RIGHT SIZE, SHIP IT */
		return bin
	}
	if l > size {
		/* bin TOO BIG, RETURN THE LAST 'size' BYTES
		WE COULD RETURN THE FIRST 'size' BYTES...
		*/
		return bin[l-size:]
	}

	/* bin TOO SMALL*/

	/* FILL BUFFER WITH 'size' SPACES */
	out := bytes.Repeat([]byte{0x20}, size)

	/* WRITE 'bin TO THE START OF THE BUFFER */
	copy(out[:l], bin)

	// fmt.Printf("\n%s ( %d ) : %x\n",str , len(out), out)
	return out
}
func ValidateStringLength(str string, size int) (out string) {

	if len(str) > size {
		/* str TOO BIG, RETURN THE  FIRST 'size' CHARS... */
		return str[:size]
	}
	/* str ALREADY THE RIGHT SIZE, SHIP IT */
	return str
}

// func Float32ToHex(f float32)

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
	return TimeSeriesData{TSDPoints(v), min, max}
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

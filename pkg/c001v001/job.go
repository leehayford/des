package c001v001

import (
	"fmt"
	"strings"

	"github.com/leehayford/des/pkg"
)

type Job struct {
	Admins              []Admin  `json:"admins"`
	Headers             []Header `json:"headers"`
	Configs             []Config `json:"configs"`
	Events              []Event  `json:"events"`
	Samples             []Sample `json:"samples"`
	XYPoints            XYPoints `json:"xypoints"`
	pkg.DESRegistration `json:"reg"`
	pkg.DBClient        `json:"-"`
}

func (job *Job) JDBX() {
	job.DBClient = pkg.DBClient{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(job.DESJobName))}

}
func (job *Job) JDB() *pkg.DBClient {
	return &pkg.DBClient{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(job.DESJobName))}
}

type XYPoint struct {
	X int64   `json:"x"`
	Y float32 `json:"y"`
}

type XYPoints struct {
	CH4     []XYPoint `json:"ch4"`
	HiFlow  []XYPoint `json:"hi_flow"`
	LoFlow  []XYPoint `json:"lo_flow"`
	Press   []XYPoint `json:"press"`
	BatAmp  []XYPoint `json:"bat_amp"`
	BatVolt []XYPoint `json:"bat_volt"`
	MotVolt []XYPoint `json:"mot_volt"`
	VlvTgt  []XYPoint `json:"vlv_tgt"`
	VlvPos  []XYPoint `json:"vlv_pos"`
}

func (xys *XYPoints) AppendXYSample(smp Sample) {
	xys.CH4 = append(xys.CH4, XYPoint{X: smp.SmpTime, Y: smp.SmpCH4})
	xys.HiFlow = append(xys.HiFlow, XYPoint{X: smp.SmpTime, Y: smp.SmpHiFlow})
	xys.LoFlow = append(xys.LoFlow, XYPoint{X: smp.SmpTime, Y: smp.SmpLoFlow})
	xys.Press = append(xys.Press, XYPoint{X: smp.SmpTime, Y: smp.SmpPress})
	xys.BatAmp = append(xys.BatAmp, XYPoint{X: smp.SmpTime, Y: smp.SmpBatAmp})
	xys.BatVolt = append(xys.BatVolt, XYPoint{X: smp.SmpTime, Y: smp.SmpBatVolt})
	xys.MotVolt = append(xys.MotVolt, XYPoint{X: smp.SmpTime, Y: smp.SmpMotVolt})
	xys.VlvTgt = append(xys.VlvTgt, XYPoint{X: smp.SmpTime, Y: float32(smp.SmpVlvTgt)})
	xys.VlvPos = append(xys.VlvPos, XYPoint{X: smp.SmpTime, Y: float32(smp.SmpVlvPos)})
}

package c001v001

import (
	// "encoding/json"

	"github.com/leehayford/des/pkg"
)

/*
SAMPLE - AS WRITTEN TO JOB DATABASE
*/
type Sample struct {
	SmpID int64 `gorm:"unique; primaryKey" json:"-"`

	SmpTime    int64   `gorm:"not null" json:"smp_time"`
	SmpCH4     float32 `json:"smp_ch4"`
	SmpHiFlow  float32 `json:"smp_hi_flow"`
	SmpLoFlow  float32 `json:"smp_lo_flow"`
	SmpPress   float32 `json:"smp_press"`
	SmpBatAmp  float32 `json:"smp_bat_amp"`
	SmpBatVolt float32 `json:"smp_bat_volt"`
	SmpMotVolt float32 `json:"smp_mot_volt"`
	SmpVlvTgt  uint32  `json:"smp_vlv_tgt"`
	SmpVlvPos  uint32  `json:"smp_vlv_pos"`
	SmpJobName string  `json:"smp_job_name"`
}

func WriteSMP(smp Sample, dbc *pkg.DBClient) (err error) {

	/* WHEN Write IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING
	WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	dbc.WG.Add(1)
	smp.SmpID = 0
	res := dbc.Create(&smp)
	dbc.WG.Done()

	return res.Error
}

/*
SAMPLE - AS STORED IN DEVICE FLASH
*/
func (smp *Sample) SampleToBytes() (out []byte) {

	out = append(out, pkg.Int64ToBytes(smp.SmpTime)...)
	out = append(out, pkg.Float32ToBytes(smp.SmpCH4)...)
	out = append(out, pkg.Float32ToBytes(smp.SmpHiFlow)...)
	out = append(out, pkg.Float32ToBytes(smp.SmpLoFlow)...)
	out = append(out, pkg.Float32ToBytes(smp.SmpPress)...)
	out = append(out, pkg.Float32ToBytes(smp.SmpBatAmp)...)
	out = append(out, pkg.Float32ToBytes(smp.SmpBatVolt)...)
	out = append(out, pkg.Float32ToBytes(smp.SmpMotVolt)...)
	out = append(out, pkg.Int16ToBytes(int16(smp.SmpVlvTgt))...)
	out = append(out, pkg.Int16ToBytes(int16(smp.SmpVlvPos))...)

	return
}
func (smp *Sample) SampleFromBytes(bytes []byte) {

	smp = &Sample{
		SmpTime:    pkg.BytesToInt64_L(bytes[0:8]),
		SmpCH4:     pkg.BytesToFloat32_L(bytes[8:12]),
		SmpHiFlow:  pkg.BytesToFloat32_L(bytes[12:16]),
		SmpLoFlow:  pkg.BytesToFloat32_L(bytes[16:20]),
		SmpPress:   pkg.BytesToFloat32_L(bytes[20:24]),
		SmpBatAmp:  pkg.BytesToFloat32_L(bytes[24:28]),
		SmpBatVolt: pkg.BytesToFloat32_L(bytes[28:32]),
		SmpMotVolt: pkg.BytesToFloat32_B(bytes[32:36]),
		SmpVlvTgt:  pkg.BytesToUInt32_L(bytes[36:38]),
		SmpVlvPos:  pkg.BytesToUInt32_L(bytes[38:40]),
	}
	// pkg.Json("(smp *Sample) SampleFromBytes(...) ->  smp:", smp)
	return
}

/*
SAMPLE - MQTT MESSAGE STRUCTURE
*/
type MQTT_Sample struct {
	DesJobName string   `json:"des_job_name"`
	Data       string `json:"data"`
}

// func (job *Job) WriteMQTTSample(msg []byte, smp Sample) (err error) {

// 	// Decode the payload into an MQTTSampleMessage
// 	mqtts := &MQTT_Sample{}
// 	if err = json.Unmarshal(msg, &mqtts); err != nil {
// 		return pkg.TraceErr(err)
// 	} // pkg.Json("DecodeMQTTSampleMessage(...) ->  msg :", msg)

// 	for _, b64 := range mqtts.Data {

// 		// Decode base64 string
// 		smp.SmpJobName = mqtts.DesJobName
// 		if err = smp.DecodeMQTTSample(b64); err != nil {
// 			return err
// 		}

// 		// Write the Sample to the job database
// 		if err = WriteSMP(smp, &job.DBClient); err != nil {
// 			return err
// 		}

// 	}

// 	return err
// }

func (smp *Sample) DecodeMQTTSample(b64 string) (err error) {

	// bytes := pkg.Base64ToBytes(b64)
	bytes := pkg.Base64URLToBytes(b64)

	smp.SmpTime = pkg.BytesToInt64_L(bytes[0:8])
	smp.SmpCH4 = pkg.BytesToFloat32_L(bytes[8:12])
	smp.SmpHiFlow = pkg.BytesToFloat32_L(bytes[12:16])
	smp.SmpLoFlow = pkg.BytesToFloat32_L(bytes[16:20])
	smp.SmpPress = pkg.BytesToFloat32_L(bytes[20:24])
	smp.SmpBatAmp = pkg.BytesToFloat32_L(bytes[24:28])
	smp.SmpBatVolt = pkg.BytesToFloat32_L(bytes[28:32])
	smp.SmpMotVolt = pkg.BytesToFloat32_L(bytes[32:36])
	smp.SmpVlvTgt = pkg.BytesToUInt32_L(bytes[36:38])
	smp.SmpVlvPos = pkg.BytesToUInt32_L(bytes[38:40])

	// pkg.Json("DecodeMQTTSampleData(...) ->  smp:", smp)

	return err
}

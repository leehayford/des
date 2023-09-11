package c001v001

import (
	"encoding/json"

	"github.com/leehayford/des/pkg"
)

/*
SAMPLE - AS WRITTEN TO JOB DATABASE
*/
type Sample struct {
	SmpID      int64   `gorm:"unique; primaryKey" json:"smp_id"`
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

/*
SAMPLE - AS STORED IN DEVICE FLASH
*/
func (smp *Sample) FilterSmpBytes() (out []byte) {

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

/*
SAMPLE - MQTT MESSAGE STRUCTURE
*/
type MQTT_Sample struct {
	DesJobName string   `json:"des_job_name"`
	Data       []string `json:"data"`
}

func (job *Job) WriteMQTTSample(msg []byte, smp *Sample) (err error) {

	// Decode the payload into an MQTTSampleMessage
	mqtts := &MQTT_Sample{}
	if err = json.Unmarshal(msg, &mqtts); err != nil {
		return pkg.TraceErr(err)
	} // pkg.Json("DecodeMQTTSampleMessage(...) ->  msg :", msg)

	for _, b64 := range mqtts.Data {

		// Decode base64 string
		smp.SmpJobName = mqtts.DesJobName
		if err = job.DecodeMQTTSample(b64, smp); err != nil {
			return err
		}

		// Write the Sample to the job database
		smp.SmpID = 0
		if err = job.Write(smp); err != nil {
			return err
		}
		
	}

	return err
}

func (job *Job) DecodeMQTTSample(b64 string, smp *Sample) (err error) {

	bytes := pkg.Base64ToBytes(b64)

	smp.SmpTime = pkg.BytesToInt64(bytes[0:8])
	smp.SmpCH4 = pkg.BytesToFloat32(bytes[8:12])
	smp.SmpHiFlow = pkg.BytesToFloat32(bytes[12:16])
	smp.SmpLoFlow = pkg.BytesToFloat32(bytes[16:20])
	smp.SmpPress = pkg.BytesToFloat32(bytes[20:24])
	smp.SmpBatAmp = pkg.BytesToFloat32(bytes[24:28])
	smp.SmpBatVolt = pkg.BytesToFloat32(bytes[28:32])
	smp.SmpMotVolt = pkg.BytesToFloat32(bytes[32:36])
	smp.SmpVlvTgt = pkg.BytesToUInt32(bytes[36:38])
	smp.SmpVlvPos = pkg.BytesToUInt32(bytes[38:40])

	// pkg.Json("DecodeMQTTSampleData(...) ->  smp:", smp)

	return err
}

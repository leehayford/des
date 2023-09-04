package c001v001

import (
	"fmt"

	"github.com/leehayford/des/pkg"
)

const DEVICE_CLASS = "001"
const DEVICE_VERSION = "001"

const MODE_BUILD int32 = 0
const MODE_VENT int32 = 2
const MODE_HI_FLOW int32 = 4
const MODE_LO_FLOW int32 = 6


type Device struct {
	ADM     Admin `json:"adm"`   // Last known Admin value
	HDR     Header `json:"hdr"`  // Last known Header value
	CFG     Config `json:"cfg"`  // Last known Config value
	EVT     Event `json:"evt"`   // Last known Event value
	SMP     Sample `json:"smp"`   // Last known Sample value
	Job `json:"job"` // The active job for this device ( last job if it has ended )
	pkg.DESRegistration `json:"reg"`
	pkg.DESMQTTClient `json:"-"`
}

type DevicesMap map[string]Device
var Devices = make(DevicesMap)

func GetDeviceList() (devices []pkg.DESRegistration, err error) {

	/*
		WHERE MORE THAN ONE JOB IS ACTIVE ( des_job_end = 0 )
		WE WANT THE LATEST
	*/
	subQryLatestJob := pkg.DES.DB.
	Table("des_jobs").
	Select("des_job_dev_id, MAX(des_job_reg_time) AS max_time").
	Where("des_job_end = 0").
	Group("des_job_dev_id")

	qry := pkg.DES.DB.
	Table("des_jobs").
	Select("des_devs.*, des_jobs.*").
	Joins(`JOIN ( ? ) j ON des_jobs.des_job_dev_id = j.des_job_dev_id AND des_job_reg_time = j.max_time`, subQryLatestJob).
	Joins("JOIN des_devs ON des_devs.des_dev_id = j.des_job_dev_id").
	Order("des_devs.des_dev_serial DESC")

	res := qry.Scan(&devices)
	err = res.Error
	return 
}

func (device *Device) StartJob( ) {
	// zero := device.GetZeroJob()
	// db := zero.JDB()
	// db.Connect()
	// defer db.Close()
	
	// db.Last(&device.ADM)
	device.ADM.AdmID = 0
	// db.Last(&device.HDR)
	device.HDR.HdrID = 0
	// db.Last(&device.CFG)
	device.CFG.CfgID = 0
	// db.Last(&device.EVT)
	device.EVT.EvtID = 0
	// db.Last(&device.SMP)
	device.SMP.SmpID = 0
	pkg.Json("(device *Device) StartJob( ) -> First Sample ", device.SMP)

	// db.Close()

	device.Job = Job{
		DESRegistration: pkg.DESRegistration{
			DESDev: device.DESDev,
			DESJob: pkg.DESJob{
				DESJobRegTime: device.HDR.HdrTime,
				DESJobRegAddr: device.HDR.HdrAddr,
				DESJobRegUserID: device.HDR.HdrUserID,
				DESJobRegApp: device.HDR.HdrApp,

				DESJobName: device.HDR.HdrJobName,
				DESJobStart: device.HDR.HdrJobStart,
				DESJobEnd: 0,
				DESJobLng: device.HDR.HdrGeoLng,
				DESJobLat: device.HDR.HdrGeoLat,
				DESJobDevID: device.DESDevID,
			},
		},
		Admins: []Admin{device.ADM},
		Headers: []Header{device.HDR},
		Configs: []Config{device.CFG},
		Events: []Event{device.EVT},
		Samples: []Sample{device.SMP},
	}
	if err := device.Job.RegisterJob(); err != nil {
		pkg.Trace(err)
	}



}
func (device *Device) EndJob( ) {

	zero := device.GetZeroJob()
	db := zero.JDB()
	db.Connect()
	defer db.Close()
	
	db.Last(&device.EVT)

	db.Close()

	// device.Job = device.GetZeroJob()
	device.DESJobRegTime = device.EVT.EvtTime
	device.DESJobRegAddr = device.EVT.EvtAddr
	device.DESDevRegUserID = device.EVT.EvtUserID
	device.DESDevRegApp = device.EVT.EvtApp
	pkg.DES.DB.Save(device.DESJob)

	device.HDR = device.Job.RegisterJob_Default_JobHeader()
	device.HDR.HdrID = 0
	device.Job.Write(&device.HDR)
	
	device.EVT.EvtID = 0
	device.Job.Write(&device.EVT)

	device.GetDeviceStatus()
}



/* Create a device client for all registered devices */
func MQTTDeviceClient_CreateAndConnectAll() (err error) {

	drs, err := GetDeviceList() 
	if err != nil {
		return pkg.Trace(err)
	} // pkg.Json("GetDeviceList():", drs)

	for _, dr := range drs {
		device := Device{
			DESRegistration:     dr,
			Job:           Job{DESRegistration: dr},
			DESMQTTClient: pkg.DESMQTTClient{},
		}
		if err = device.MQTTDeviceClient_Connect(); err != nil {
			return pkg.Trace(err)
		}
		device.GetDeviceStatus()
		Devices[device.DESDevSerial] = device
		// d := Devices[device.DESDevSerial]
		// fmt.Printf("\nCached Device %s, current event code: %d\n", d.DESDevSerial, d.EVT.EvtCode)
	}

	return err
}

/* MQTT TOPICS - SIGNAL */
func (device *Device) MQTTTopic_SIGAdmin() (topic string) {
	return fmt.Sprintf("%s/%s/%s/sig/admin",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_SIGHeader() (topic string) {
	return fmt.Sprintf("%s/%s/%s/sig/header",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_SIGConfig() (topic string) {
	return fmt.Sprintf("%s/%s/%s/sig/config",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_SIGEvent() (topic string) {
	return fmt.Sprintf("%s/%s/%s/sig/event",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_SIGSample() (topic string) {
	return fmt.Sprintf("%s/%s/%s/sig/sample",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_SIGDiagSample() (topic string) {
	return fmt.Sprintf("%s/%s/%s/sig/diag_sample",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}

/* MQTT TOPICS - COMMAND */
func (device *Device) MQTTTopic_CMDAdmin() (topic string) {
	return fmt.Sprintf("%s/%s/%s/cmd/admin",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_CMDHeader() (topic string) {
	return fmt.Sprintf("%s/%s/%s/cmd/header",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_CMDConfig() (topic string) {
	return fmt.Sprintf("%s/%s/%s/cmd/config",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_CMDEvent() (topic string) {
	return fmt.Sprintf("%s/%s/%s/cmd/event",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_CMDSample() (topic string) {
	return fmt.Sprintf("%s/%s/%s/cmd/sample",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_CMDDiagSample() (topic string) {
	return fmt.Sprintf("%s/%s/%s/cmd/diag_sample",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
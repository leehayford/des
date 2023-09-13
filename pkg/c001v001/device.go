package c001v001

import (
	"fmt"
	"strings"
	"time"
	"github.com/leehayford/des/pkg"
)

const DEVICE_CLASS = "001"
const DEVICE_VERSION = "001"

const MODE_BUILD int32 = 0
const MODE_VENT int32 = 2
const MODE_HI_FLOW int32 = 4
const MODE_LO_FLOW int32 = 6

type Device struct {
	ADM                 Admin        `json:"adm"` // Last known Admin value
	HDR                 Header       `json:"hdr"` // Last known Header value
	CFG                 Config       `json:"cfg"` // Last known Config value
	EVT                 Event        `json:"evt"` // Last known Event value
	SMP                 Sample       `json:"smp"` // Last known Sample value
	Job                 `json:"job"` // The active job for this device ( last job if it has ended )
	pkg.DESRegistration `json:"reg"`
	pkg.DESMQTTClient   `json:"-"`
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

func (device *Device) ZeroJobName() string {
	return fmt.Sprintf("%s_0000000000000", device.DESDevSerial)
}
func (device *Device) ZeroJobDB() *pkg.DBI {
	return &pkg.DBI{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(device.ZeroJobName()))}
}
func (device *Device) ZeroJob() Job {
	return Job{ DESRegistration: pkg.DESRegistration{ DESJob: pkg.DESJob{ DESJobName: device.ZeroJobName() }} }
}

func (device *Device) GetZeroJob() (zero Job) {
	// fmt.Printf("\n(device) GetZeroJob() for: %s\n", device.DESDevSerial)
	qry := pkg.DES.DB.
		Table("des_devs AS d").
		Select("d.*, j.*").
		Joins("JOIN des_jobs AS j ON d.des_dev_id = j.des_job_dev_id").
		Where("d.des_dev_serial = ? AND j.des_job_name LIKE ?",
			device.DESDevSerial, fmt.Sprintf("%s_0000000000000", device.DESDevSerial))

	res := qry.Scan(&zero.DESRegistration)
	if res.Error != nil {
		pkg.TraceErr(res.Error)
	}
	// pkg.Json("(device *Device) GetZeroJob( )", zero)
	return
}
func (device *Device) GetCurrentJob() {
	// fmt.Printf("\n(device) GetCurrentJob() for: %s\n", device.DESDevSerial)

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
		Where("des_devs.des_dev_serial = ? ", device.DESDevSerial)

	res := qry.Scan(&device.Job.DESRegistration)
	if res.Error != nil {
		pkg.TraceErr(res.Error)
	}
	// pkg.Json("(device *Device) GetCurrentJob( )", device.Job)
	return
}
func (device *Device) GetDeviceStatus() (err error) {
	device.GetCurrentJob()
	db := device.Job.JDB()
	db.Connect()
	defer db.Close()

	db.Last(&device.ADM)
	db.Last(&device.HDR)
	db.Last(&device.CFG)
	db.Last(&device.EVT)
	db.Last(&device.SMP)

	db.Close()
	
	d := Devices[device.DESDevSerial]
	device.DESMQTTClient = d.DESMQTTClient
	Devices[device.DESDevSerial] = *device

return
}

func (device *Device) StartJob() {

	device.ADM.AdmID = 0
	device.HDR.HdrID = 0
	device.CFG.CfgID = 0
	device.EVT.EvtID = 0
	device.SMP.SmpID = 0 // pkg.Json("(device *Device) StartJob( ) -> First Sample ", device.SMP)

	device.Job = Job{
		DESRegistration: pkg.DESRegistration{
			DESDev: device.DESDev,
			DESJob: pkg.DESJob{
				DESJobRegTime:   device.HDR.HdrTime,
				DESJobRegAddr:   device.HDR.HdrAddr,
				DESJobRegUserID: device.HDR.HdrUserID,
				DESJobRegApp:    device.HDR.HdrApp,

				DESJobName:  device.HDR.HdrJobName,
				DESJobStart: device.HDR.HdrJobStart,
				DESJobEnd:   0,
				DESJobLng:   device.HDR.HdrGeoLng,
				DESJobLat:   device.HDR.HdrGeoLat,
				DESJobDevID: device.DESDevID,
			},
		},
		Admins:  []Admin{device.ADM},
		Headers: []Header{device.HDR},
		Configs: []Config{device.CFG},
		Events:  []Event{device.EVT},
		Samples: []Sample{device.SMP},
	}
	if err := device.Job.RegisterJob(); err != nil {
		pkg.TraceErr(err)
	}
	
	d := Devices[device.DESDevSerial] // pkg.Json("(device *Device) StartJob(): -> Devices[device.DESDevSerial] BEFORE UPDATE", d)
	device.DESMQTTClient = d.DESMQTTClient
	Devices[device.DESDevSerial] = *device // d = Devices[device.DESDevSerial] // pkg.Json("(device *Device) StartJob(): -> Devices[device.DESDevSerial] AFTER UPDATE", d)

	fmt.Printf("\n(device *Device) StartJob( ): %s\n", device.HDR.HdrJobName)
}
func (device *Device) EndJob() {
	
	d := Devices[device.DESDevSerial] // pkg.Json("(device *Device) EndJob(): -> Devices[device.DESDevSerial] BEFORE UPDATE", d)
	d.EVT = device.EVT
	d.EVT.EvtID = 0
	d.Job.Write(&d.EVT)

	/* CLOSE DES JOB X */
	d.Job.DESJob.DESJobRegTime = d.EVT.EvtTime
	d.Job.DESJob.DESJobRegAddr = d.EVT.EvtAddr
	d.Job.DESJob.DESJobRegUserID = d.EVT.EvtUserID
	d.Job.DESJob.DESJobRegApp = d.EVT.EvtApp
	d.Job.DESJob.DESJobEnd = d.EVT.EvtTime
	// pkg.Json("(device *Device) EndJob() -> CLOSE DES JOB X ->", d.Job.DESJob)
	pkg.DES.DB.Save(d.Job.DESJob)
	fmt.Printf("\n(device *Device) EndJob( ) ENDING: %s\n", d.HDR.HdrJobName)

	/* UPDATE DES JOB 0 */
	zero := d.GetZeroJob()
	zero.DESJobRegTime = time.Now().UTC().UnixMilli() // WE WANT THIS TO BE THE LATEST
	zero.DESJobRegAddr = d.EVT.EvtAddr
	zero.DESJobRegUserID = d.EVT.EvtUserID
	zero.DESJobRegApp = d.EVT.EvtApp
	// pkg.Json("(device *Device) EndJob() -> UPDATE DES JOB 0  ->", zero.DESJob)
	pkg.DES.DB.Save(zero.DESJob)

	/* SET DEVICE JOB TO JOB 0 - > AVEC DEFAULT HEADER */
	d.Job = zero // ONLY THE DESRegistration is hydrated

	d.ADM = d.Job.RegisterJob_Default_JobAdmin()
	d.ADM.AdmID = 0
	d.MQTTPublication_DeviceClient_CMDAdmin(d.ADM)

	d.HDR = d.Job.RegisterJob_Default_JobHeader()
	d.HDR.HdrID = 0
	d.MQTTPublication_DeviceClient_CMDHeader(d.HDR)

	d.CFG = d.Job.RegisterJob_Default_JobConfig()
	d.CFG.CfgID = 0
	d.MQTTPublication_DeviceClient_CMDConfig(d.CFG)

	Devices[device.DESDevSerial] = d

	// device.EVT.EvtID = 0
	// device.Job.Write(&device.EVT)

	// /* CLOSE DES JOB X */
	// device.Job.DESJob.DESJobRegTime = device.EVT.EvtTime
	// device.Job.DESJob.DESJobRegAddr = device.EVT.EvtAddr
	// device.Job.DESJob.DESJobRegUserID = device.EVT.EvtUserID
	// device.Job.DESJob.DESJobRegApp = device.EVT.EvtApp
	// device.Job.DESJob.DESJobEnd = device.EVT.EvtTime
	// // pkg.Json("(device *Device) EndJob() -> CLOSE DES JOB X ->", device.Job.DESJob)
	// pkg.DES.DB.Save(device.Job.DESJob)
	// fmt.Printf("\n(device *Device) EndJob( ) ENDING: %s\n", device.HDR.HdrJobName)

	// /* UPDATE DES JOB 0 */
	// zero := device.GetZeroJob()
	// zero.DESJobRegTime = time.Now().UTC().UnixMilli() // WE WANT THIS TO BE THE LATEST
	// zero.DESJobRegAddr = device.EVT.EvtAddr
	// zero.DESJobRegUserID = device.EVT.EvtUserID
	// zero.DESJobRegApp = device.EVT.EvtApp
	// // pkg.Json("(device *Device) EndJob() -> UPDATE DES JOB 0  ->", zero.DESJob)
	// pkg.DES.DB.Save(zero.DESJob)

	// /* SET DEVICE JOB TO JOB 0 - > AVEC DEFAULT HEADER */
	// device.Job = zero // ONLY THE DESRegistration is hydrated

	// device.ADM = device.Job.RegisterJob_Default_JobAdmin()
	// device.ADM.AdmID = 0
	// device.MQTTPublication_DeviceClient_CMDAdmin(device.ADM)

	// device.HDR = device.Job.RegisterJob_Default_JobHeader()
	// device.HDR.HdrID = 0
	// device.MQTTPublication_DeviceClient_CMDHeader(device.HDR)

	// device.CFG = device.Job.RegisterJob_Default_JobConfig()
	// device.CFG.CfgID = 0
	// device.MQTTPublication_DeviceClient_CMDConfig(device.CFG)

	// d := Devices[device.DESDevSerial] // pkg.Json("(device *Device) EndJob(): -> Devices[device.DESDevSerial] BEFORE UPDATE", d)
	// device.DESMQTTClient = d.DESMQTTClient
	// Devices[device.DESDevSerial] = *device // d = Devices[device.DESDevSerial] // pkg.Json("(device *Device) EndJob(): -> Devices[device.DESDevSerial] AFTER UPDATE", d)

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

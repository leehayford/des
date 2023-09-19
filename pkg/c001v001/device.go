package c001v001

import (
	"fmt"
	"strings"
	"time"

	"github.com/leehayford/des/pkg"
)

const DEVICE_CLASS = "001"
const DEVICE_VERSION = "001"

const STATUS_JOB_END_REQ int32 = 2
const STATUS_JOB_ENDED int32 = 3

const STATUS_JOB_START_REQ int32 = 4
const STATUS_JOB_STARTED int32 = 5

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
	pkg.DESMQTTClient   `json:"-"`   // MQTT client handling all subscriptions and publications for this device
	JobDBC              pkg.DBClient `json:"-"` // Database Client for the current job
	ZeroDBC             pkg.DBClient `json:"-"` // Database Client for the zero job
}

type DevicesMap map[string]Device

var Devices = make(DevicesMap)

/* GET THE CURRENT DES REGISTRATION INFO FOR ALL DEVICES */
func GetDeviceList() (devices []pkg.DESRegistration, err error) {

	/* WHERE MORE THAN ONE JOB IS ACTIVE ( des_job_end = 0 ) WE WANT THE LATEST */
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
func DeviceClient_ConnectAll() {

	ds, err := GetDeviceList()
	if err != nil {
		pkg.TraceErr(err)
	}

	for _, r := range ds {
		(&Device{}).DeviceClient_Connect(r)
	}
}
func DeviceClient_DisconnectAll() {
	/* TODO: TEST WHEN IMPLEMENTING GRACEFUL SHUTDOWN */
	for _, d := range Devices {
		d.DeviceClient_Disconnect()
	}

	return
}

/*
	CONNECT DEVICE DATABASE AND MQTT CLIENTS

ADD CONNECTED DEVICE TO DevicesMap
*/
func (device *Device) DeviceClient_Connect(reg pkg.DESRegistration) {
	device.DESRegistration = reg
	device.Job = Job{DESRegistration: reg}
	device.DESMQTTClient = pkg.DESMQTTClient{}

	if err := device.ConnectZeroDBC(); err != nil {
		pkg.TraceErr(err)
	}
	if err := device.ConnectJobDBC(); err != nil {
		pkg.TraceErr(err)
	}
	if err := device.MQTTDeviceClient_Connect(); err != nil {
		pkg.TraceErr(err)
	}

	Devices[device.DESDevSerial] = *device
}

/*
	DISCONNECT DEVICE DATABASE AND MQTT CLIENTS

REMOVE CONNECTED DEVICE FROM DevicesMap
*/
func (device *Device) DeviceClient_Disconnect() {
	/* TODO: TEST WHEN IMPLEMENTING
	UNREGISTER DEVICE
	GRACEFUL SHUTDOWN
	*/
	if err := device.DisconnectZeroDBC(); err != nil {
		pkg.TraceErr(err)
	}
	if err := device.DisconnectJobDBC(); err != nil {
		pkg.TraceErr(err)
	}
	if err := device.MQTTDeviceClient_Disconnect(); err != nil {
		pkg.TraceErr(err)
	}
	delete(Devices, device.DESDevSerial)
}

/* HYDRATES THE DEVICE'S DB & MQTT CLIENT OBJECTS OF THE DEVICE FROM DevicesMap */
func (device *Device) GetMappedClients() {

	/* GET THE DEVICE CLIENT DATA FROM THE DEVICES CLIENT MAP */
	d := Devices[device.DESDevSerial]
	device.ZeroDBC = d.ZeroDBC
	device.JobDBC = d.JobDBC
	device.DESMQTTClient = d.DESMQTTClient
}

/* HYDRATES THE DEVICE'S Admin STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedADM() {
	d := Devices[device.DESDevSerial]
	device.ADM = d.ADM
}

/* HYDRATES THE DEVICE'S HEader STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedHDR() {
	d := Devices[device.DESDevSerial]
	device.HDR = d.HDR
}

/* HYDRATES THE DEVICE'S Config STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedCFG() {
	d := Devices[device.DESDevSerial]
	device.CFG = d.CFG
}

/* HYDRATES THE DEVICE'S Event STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedEVT() {
	d := Devices[device.DESDevSerial]
	device.EVT = d.EVT
}

/* HYDRATES THE DEVICE'S Sample STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedSMP() {
	d := Devices[device.DESDevSerial]
	device.SMP = d.SMP
}

/* RETURNS THE CMDArchive NAME  */
func (device Device) ZeroJobName() string {
	return fmt.Sprintf("%s_0000000000000", device.DESDevSerial)
	// return fmt.Sprintf("%s_CMDARCHIVE_DB", device.DESDevSerial)
}

/* RETURNS THE ZERO JOB DESRegistration FROM THE DES DATABASE */
func (device Device) GetZeroJob() (zero Job) {
	// fmt.Printf("\n(device) GetZeroJob() for: %s\n", device.DESDevSerial)
	qry := pkg.DES.DB.
		Table("des_devs AS d").
		Select("d.*, j.*").
		Joins("JOIN des_jobs AS j ON d.des_dev_id = j.des_job_dev_id").
		Where("d.des_dev_serial = ? AND j.des_job_name LIKE ?",
			device.DESDevSerial, device.ZeroJobName())

	res := qry.Scan(&zero.DESRegistration)
	if res.Error != nil {
		pkg.TraceErr(res.Error)
	}
	// pkg.Json("(device *Device) GetZeroJob( )", zero)
	return
}

/* RETURNS THE ZERO JOB DESRegistration NAME ONLY ***NOT*** FROM DATABASE */
func (device Device) ZeroJob() Job {
	return Job{DESRegistration: pkg.DESRegistration{DESJob: pkg.DESJob{DESJobName: device.ZeroJobName()}}}
}

/* CONNECTS THE ZERO JOB DBClient TO THE ZERO JOB DATABASE */
func (device *Device) ConnectZeroDBC() (err error) {
	device.ZeroDBC = pkg.DBClient{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(device.ZeroJobName()))}
	return device.ZeroDBC.Connect()
}

/* CLOSES ZERO JOB DATABASE CONNECTION */
func (device *Device) DisconnectZeroDBC() (err error) {
	err = device.ZeroDBC.Close()
	device.ZeroDBC.DB = nil
	return err
}

/* CONNECTS THE ACTIVE JOB DBClient TO THE ACTIVE JOB DATABASE */
func (device *Device) ConnectJobDBC() (err error) {
	device.JobDBC = pkg.DBClient{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(device.Job.DESJobName))}
	return device.JobDBC.Connect()
}

/* CLOSES ACTIVE JOB DATABASE CONNECTION */
func (device *Device) DisconnectJobDBC() (err error) {
	err = device.JobDBC.Close()
	device.JobDBC.DB = nil
	return err
}

/* RETURNS THE DESRegistration FOR THE DEVICE AND ITS ACTIVE JOB FROM THE DES DATABASE */
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

	res := qry.Scan(&device.DESRegistration)
	if res.Error != nil {
		pkg.TraceErr(res.Error)
		return
	}
	// pkg.Json("(device *Device) GetCurrentJob( )", device.Job)
	return
}

/* HYDRATES THE DEVICE'S DevicesMap WITH THE MOST RECENT RECORDS FROM THE ACTIVE JOB DATABASE  */
func (device *Device) GetDeviceStatus() (err error) {

	device.GetCurrentJob()
	// fmt.Printf("\n(device *Device) GetDeviceStatus() -> device.DESJobName: %v\n", device.DESJobName)

	d := Devices[device.DESDevSerial]
	// fmt.Printf("\n(device *Device) GetDeviceStatus() -> d.DESJobName: %v\n", d.DESJobName)

	if d.ZeroDBC.DB == nil {
		d.ConnectZeroDBC() // fmt.Printf("\n(device *Device) GetDeviceStatus() -> d.ZeroDBC.DB: %v\n", d.ZeroDBC.DB)
	}

	if d.JobDBC.DB == nil {
		d.ConnectJobDBC() // fmt.Printf("\n(device *Device) GetDeviceStatus() -> d.JobDBC.DB: %v\n", d.JobDBC.DB)
	}

	d.JobDBC.Last(&d.ADM)
	d.JobDBC.Last(&d.HDR)
	d.JobDBC.Last(&d.CFG)
	d.JobDBC.Last(&d.EVT)
	d.JobDBC.Last(&d.SMP)
	Devices[device.DESDevSerial] = d

	return
}

/* CALLED WHEN THE DEVICE MQTT CLIENT REVIEVES A 'JOB STARTED' EVENT FROM THE DEVICE */
func (device *Device) StartJob() {

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedClients()

	/* CLEAR THE ACTIVE JOB DATABASE CONNECTION */
	device.DisconnectJobDBC()

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

	/* CREATE A JOB RECORD IN THE DES DATABASE */
	if res := pkg.DES.DB.Create(&device.Job.DESJob); res.Error != nil {
		pkg.TraceErr(res.Error)
	}

	dbName := strings.ToLower(device.Job.DESJobName)
	/* WE AVOID WRITING IF THE DATABASE WAS PRE-EXISTING */
	if !pkg.ADB.CheckDatabaseExists(dbName) {

		/* CREATE NEW JOB DATABASE */
		pkg.ADB.CreateDatabase(dbName)

		/* CONNECT THE NEW ACTIVE JOB DATABASE */
		device.ConnectJobDBC()
		fmt.Printf("\n(device *Device) StartJob(): CONNECTED TO DATABASE: %s\n", device.HDR.HdrJobName)

		/* CREATE JOB DB TABLES */
		if err := device.JobDBC.Migrator().CreateTable(
			&Admin{},
			&Header{},
			&Config{},
			&EventTyp{},
			&Event{},
			&Sample{},
		); err != nil {
			pkg.TraceErr(err)
		}

		/* WRITE INITIAL JOB RECORDS */
		for _, typ := range EVENT_TYPES {
			device.JobDBC.Create(&typ)
		}
		if res := device.JobDBC.Create(&device.Job.Admins[0]); res.Error != nil {
			pkg.TraceErr(res.Error)
		}
		if res := device.JobDBC.Create(&device.Job.Headers[0]); res.Error != nil {
			pkg.TraceErr(res.Error)
		}
		if res := device.JobDBC.Create(&device.Job.Configs[0]); res.Error != nil {
			pkg.TraceErr(res.Error)
		}
		if res := device.JobDBC.Create(&device.Job.Events[0]); res.Error != nil {
			pkg.TraceErr(res.Error)
		}
		if res := device.JobDBC.Create(&device.Job.Samples[0]); res.Error != nil {
			pkg.TraceErr(res.Error)
		}

	}

	if device.JobDBC.DB == nil {
		device.JobDBC = device.ZeroDBC
		fmt.Printf("\n(device *Device) StartJob( ) FAILED! *** LOGGING TO: %s\n", device.HDR.HdrJobName)
	}

	/* UPDATE THE DEVICES CLIENT MAP */
	Devices[device.DESDevSerial] = *device

	fmt.Printf("\n(device *Device) StartJob( ) COMPLETE: %s\n", device.HDR.HdrJobName)
}

/* CALLED WHEN THE DEVICE MQTT CLIENT REVIEVES A 'JOB ENDED' EVENT FROM THE DEVICE */
func (device *Device) EndJob() {

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedClients()

	/* WRITE END JOB REQUEST EVENT AS RECEIVED TO JOB X */
	device.JobDBC.Write(&device.EVT)

	/* CLEAR THE ACTIVE JOB DATABASE CONNECTION */
	device.DisconnectJobDBC()

	/* CLOSE DES JOB X */
	device.Job.DESJob.DESJobRegTime = device.EVT.EvtTime
	device.Job.DESJob.DESJobRegAddr = device.EVT.EvtAddr
	device.Job.DESJob.DESJobRegUserID = device.EVT.EvtUserID
	device.Job.DESJob.DESJobRegApp = device.EVT.EvtApp
	device.Job.DESJob.DESJobEnd = device.EVT.EvtTime
	fmt.Printf("\n(device *Device) EndJob( ) ENDING: %s\n", device.HDR.HdrJobName)
	pkg.DES.DB.Save(device.Job.DESJob)

	/* UPDATE DES JOB 0 */
	zero := device.GetZeroJob()
	zero.DESJobRegTime = time.Now().UTC().UnixMilli() // WE WANT THIS TO BE THE LATEST
	zero.DESJobRegAddr = device.EVT.EvtAddr
	zero.DESJobRegUserID = device.EVT.EvtUserID
	zero.DESJobRegApp = device.EVT.EvtApp
	pkg.DES.DB.Save(zero.DESJob)

	/* SET DEVICE JOB TO JOB 0 - > AVEC DEFAULT ADMIN, HEADER, & CONFIG */
	device.Job = zero
	device.ConnectJobDBC() // ENSURE WE CATCH STRAY SAMPLES IN THE ZERO JOB
	fmt.Printf("\n(device *Device) EndJob( ) ENDING -> JOB ZERO ACTIVE: %s\n", device.JobDBC.ConnStr)

	/* RETURN DEVICE CLIENT DATA TO DEFAULT STATE */
	device.ADM = device.Job.RegisterJob_Default_JobAdmin()
	device.ADM.AdmTime = zero.DESJobRegTime
	device.ADM.AdmAddr = device.EVT.EvtAddr
	device.ADM.AdmUserID = device.EVT.EvtUserID
	device.ADM.AdmApp = device.EVT.EvtApp

	device.HDR = device.Job.RegisterJob_Default_JobHeader()
	device.HDR.HdrTime = zero.DESJobRegTime
	device.HDR.HdrAddr = device.EVT.EvtAddr
	device.HDR.HdrUserID = device.EVT.EvtUserID
	device.HDR.HdrApp = device.EVT.EvtApp

	device.CFG = device.Job.RegisterJob_Default_JobConfig()
	device.CFG.CfgTime = zero.DESJobRegTime
	device.CFG.CfgAddr = device.EVT.EvtAddr
	device.CFG.CfgUserID = device.EVT.EvtUserID
	device.CFG.CfgApp = device.EVT.EvtApp

	/* RETURN DEVICE (PHYSICAL) DATA TO DEFAULT STATE */
	device.MQTTPublication_DeviceClient_CMDAdmin(device.ADM)
	device.MQTTPublication_DeviceClient_CMDHeader(device.HDR)
	device.MQTTPublication_DeviceClient_CMDConfig(device.CFG)
	// pkg.Json("(device *Device) EndJob(): -> Devices[device.DESDevSerial] AFTER UPDATE", device)

	Devices[device.DESDevSerial] = *device

	fmt.Printf("\n(device *Device) EndJob( ) COMPLETE: %s\n", device.HDR.HdrJobName)
}

/* MQTT TOPICS ************************************************************************
THESE ARE HERE BECAUSE THEY ARE USED BY MORE THAN ONE TYPE OF CLIENT */

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

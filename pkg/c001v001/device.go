package c001v001

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/leehayford/des/pkg"
)

const DEVICE_CLASS = "001"
const DEVICE_VERSION = "001"

/********************************************************************************************************/
/* STATUS ( Event.EvtCode ) ****************************************************************************/

const STATUS_DES_REG_REQ int32 = 0 // USER REQUEST -> CHANGE DEVICE'S OPERATIONAL DATA EXCHANGE SERVER
const STATUS_DES_REGISTERED int32 = 1 // DEVICE RESPONSE -> SENT TO NEW DATA EXCHANGE SERVER
const STATUS_JOB_ENDED int32 = 2 // DEVICE RESPONSE -> WAITING TO START A NEW JOB
const STATUS_JOB_START_REQ int32 = 3 // USER REQUEST -> START JOB

/* STATUS > JOB_START_REQ MEANS WE ARE LOGGING TO AN ACTIVE JOB */
const STATUS_JOB_STARTED int32 = 4 // DEVICE RESPONSE -> JOB HAS BEGUN
const STATUS_JOB_END_REQ int32 = 5 // USER REQUEST -> END JOB

/* STATUS ( Event.EvtCode ) ****************************************************************************/
/*********************************************************************************************************/

/* VALVE POSITIONS */
const MODE_BUILD int32 = 0
const MODE_VENT int32 = 2
const MODE_HI_FLOW int32 = 4
const MODE_LO_FLOW int32 = 6

/*
	FOR EACH REGISTERED DEVICE, THE DES MAINTAINS:

	THE MOST RECENT REGISTRATION DATA FOR THE DEVICE ITSELF, AND THE ACTIVE JOB

	THE MOST RECENT MESSAGE DATA FROM THE DEVICE, ONE OF EACH DATA MODEL PRESENT IN A JOB DATABASE

	SEVERAL DEDICATED CONNECTIONS:

		A DEVICE-SPECIFIC CMDARCHIVE DATABASE ( FOR LIFE )
		A DEVICE-SPECIFIC ACTIVE JOB DATABASE ( CHANGES WITH EACH JOB START )
		DEVICE-SPECIFIC MQTT CLIENT ( FOR LIFE )
*/
type Device struct {
	pkg.DESRegistration `json:"reg"` // Contains registration data for both the device and active job
	ADM                 Admin        `json:"adm"` // Last known Admin value
	HDR                 Header       `json:"hdr"` // Last known Header value
	CFG                 Config       `json:"cfg"` // Last known Config value
	EVT                 Event        `json:"evt"` // Last known Event value
	SMP                 Sample       `json:"smp"` // Last known Sample value

	/****************************************************************************************************/
	/* TODO: REMOVE Job STRUCT FROM DEVICE SO Job CAN BE DEDICATED TO REPORTING */
	Job                 `json:"job"` // The active job for this device ( CMDARCHIVE when between jobs )
	/****************************************************************************************************/

	CmdDBC              pkg.DBClient `json:"-"` // Database Client for the CMDARCHIVE
	JobDBC              pkg.DBClient `json:"-"` // Database Client for the active job
	pkg.DESMQTTClient   `json:"-"`   // MQTT client handling all subscriptions and publications for this device
}

type DevicesMap map[string]Device

var Devices = make(DevicesMap)
var DevicesRWMutex = sync.RWMutex{}

/* GET THE CURRENT DESRegistration FOR ALL DEVICES ON THIS DES */
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

/* CALLED ON SERVER STARTUP */
func DeviceClient_ConnectAll() {

	regs, err := GetDeviceList()
	if err != nil {
		pkg.TraceErr(err)
	}

	for _, reg := range regs {
		device := Device{}
		device.DESRegistration = reg
		device.Job = Job{DESRegistration: reg}
		device.DESMQTTClient = pkg.DESMQTTClient{}
		device.DeviceClient_Connect()
	}
}

/* CALLED ON SERVER SHUT DOWN */
func DeviceClient_DisconnectAll() {
	/* TODO: TEST WHEN IMPLEMENTING
	- UNREGISTER DEVICE
	- GRACEFUL SHUTDOWN
	*/
	fmt.Printf("\nDeviceClient_DisconnectAll()\n")
	for _, d := range Devices {
		d.DeviceClient_Disconnect()
	}
}

/* CONNECT DEVICE DATABASE AND MQTT CLIENTS ADD CONNECTED DEVICE TO DevicesMap */
func (device *Device) DeviceClient_Connect() {

	fmt.Printf("\n\n(device *Device) DeviceClient_Connect() -> %s -> connecting... \n", device.DESDevSerial)

	fmt.Printf("\n(device *Device) DeviceClient_Connect() -> %s -> connecting CMDARCHIVE... \n", device.DESDevSerial)
	if err := device.ConnectCmdDBC(); err != nil {
		pkg.TraceErr(err)
	}

	fmt.Printf("\n(device *Device) DeviceClient_Connect() -> %s -> connecting ACTIVE JOB... \n", device.DESDevSerial)
	if err := device.ConnectJobDBC(); err != nil {
		pkg.TraceErr(err)
	}

	device.JobDBC.Last(&device.ADM)
	device.JobDBC.Last(&device.HDR)
	device.JobDBC.Last(&device.CFG)
	device.JobDBC.Last(&device.SMP)
	device.JobDBC.Last(&device.EVT)

	if err := device.MQTTDeviceClient_Connect(); err != nil {
		pkg.TraceErr(err)
	}

	/* ADD TO Devices MAP */
	UpdateDevicesMap(device.DESDevSerial,  *device)
	// Devices[device.DESDevSerial] = *device
	fmt.Printf("\n(device *Device) DeviceClient_Connect() -> %s -> connected... \n\n", device.DESDevSerial)
}

/* DISCONNECT DEVICE DATABASE AND MQTT CLIENTS; REMOVE CONNECTED DEVICE FROM DevicesMap */
func (device *Device) DeviceClient_Disconnect() {
	/* TODO: TEST WHEN IMPLEMENTING
	- UNREGISTER DEVICE
	- GRACEFUL SHUTDOWN
	*/
	fmt.Printf("\n\n(device *Device) DeviceClient_Disconnect() -> %s -> disconnecting... \n", device.DESDevSerial)

	if err := device.CmdDBC.Disconnect(); err != nil {
		pkg.TraceErr(err)
	}
	if err := device.JobDBC.Disconnect(); err != nil {
		pkg.TraceErr(err)
	}
	if err := device.MQTTDeviceClient_Disconnect(); err != nil {
		pkg.TraceErr(err)
	}
	delete(Devices, device.DESDevSerial)
}

/* UPDATE THE DevicesMap USING RWMutex TO PREVENT CONCURRENT MAP WRITES */
func  (device *Device) ReadDevicesMap(serial string) (d Device) {
	
	DevicesRWMutex.Lock()
	d = Devices[serial]
	DevicesRWMutex.Unlock()
	return
}

/* HYDRATES THE DEVICE'S DB & MQTT CLIENT OBJECTS OF THE DEVICE FROM DevicesMap */
func (device *Device) GetMappedClients() {

	/* GET THE DEVICE CLIENT DATA FROM THE DEVICES CLIENT MAP */
	d :=  device.ReadDevicesMap(device.DESDevSerial)
	
	/* WAIT TO PREVENT RACE CONDITION - DON"T READ WHEN DBC IS BUSY */
	d.CmdDBC.WG.Wait()
	if device.CmdDBC.DB != nil {
		device.CmdDBC.WG.Wait()
	}
	device.CmdDBC = d.CmdDBC

	/* WAIT TO PREVENT RACE CONDITION - DON"T READ WHEN DBC IS BUSY */
	d.JobDBC.WG.Wait()
	if device.JobDBC.DB != nil {
		device.JobDBC.WG.Wait()
	}
	device.JobDBC = d.JobDBC

	/* WAIT TO PREVENT RACE CONDITION - DON"T READ WHEN DESMQTTClient IS BUSY */
	d.DESMQTTClient.WG.Wait()
	device.DESMQTTClient.WG.Wait()
	device.DESMQTTClient = d.DESMQTTClient
}

/* HYDRATES THE DEVICE'S Admin STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedADM() {
	d :=  device.ReadDevicesMap(device.DESDevSerial)
	device.ADM = d.ADM
}

/* HYDRATES THE DEVICE'S HEader STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedHDR() {
	d :=  device.ReadDevicesMap(device.DESDevSerial)
	device.HDR = d.HDR
}

/* HYDRATES THE DEVICE'S Config STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedCFG() {
	d :=  device.ReadDevicesMap(device.DESDevSerial)
	device.CFG = d.CFG
}

/* ENSURE THE SAMPLE / LOG / TRANS RATES HAVE BEEN SET WITHIN ACCEPTABLE LIMITS */
func (device *Device) ValidateCFG() {
	/* TODO: SET ACCEPTABLE LIMITS FOR THE REST OF THE CONFIG SETTINGS */
	smpPeriodLimit := int32(200)
	if device.CFG.CfgOpSample < smpPeriodLimit { device.CFG.CfgOpSample = smpPeriodLimit }
	if device.CFG.CfgOpLog < device.CFG.CfgOpSample { device.CFG.CfgOpLog = device.CFG.CfgOpSample }
	if device.CFG.CfgOpTrans < device.CFG.CfgOpSample { device.CFG.CfgOpTrans = device.CFG.CfgOpSample }
	if device.CFG.CfgDiagSample < smpPeriodLimit { device.CFG.CfgDiagSample = smpPeriodLimit }
	if device.CFG.CfgDiagLog < device.CFG.CfgDiagSample { device.CFG.CfgDiagLog = device.CFG.CfgDiagSample }
	if device.CFG.CfgDiagTrans < device.CFG.CfgDiagSample { device.CFG.CfgDiagTrans = device.CFG.CfgDiagSample }
}

/* HYDRATES THE DEVICE'S Event STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedEVT() {
	d :=  device.ReadDevicesMap(device.DESDevSerial)
	device.EVT = d.EVT
}

/* HYDRATES THE DEVICE'S Sample STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedSMP() {
	d :=  device.ReadDevicesMap(device.DESDevSerial)
	device.SMP = d.SMP
}

/* UPDATE THE DevicesMap USING RWMutex TO PREVENT CONCURRENT MAP WRITES */
func UpdateDevicesMap(serial string, d Device) {
	DevicesRWMutex.Lock()
	Devices[serial] = d
	DevicesRWMutex.Unlock()
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Admin */
func (device *Device) UpdateMappedADM() {
	d :=  device.ReadDevicesMap(device.DESDevSerial)
	d.ADM = device.ADM
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Header */
func (device *Device) UpdateMappedHDR() {
	d :=  device.ReadDevicesMap(device.DESDevSerial)
	d.HDR = device.HDR
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Config */
func (device *Device) UpdateMappedCFG() {
	d :=  device.ReadDevicesMap(device.DESDevSerial)
	d.CFG = device.CFG
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Event */
func (device *Device) UpdateMappedEVT() {
	d :=  device.ReadDevicesMap(device.DESDevSerial)
	d.EVT = device.EVT
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Sample */
func (device *Device) UpdateMappedSMP() {
	d :=  device.ReadDevicesMap(device.DESDevSerial)
	d.SMP = device.SMP
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* RETURNS THE CMDARCHIVE NAME  */
func (device Device) CmdArchiveName() string {
	return fmt.Sprintf("%s_CMDARCHIVE", device.DESDevSerial)
}

/* RETURNS THE CMDARCHIVE DESRegistration FROM THE DES DATABASE */
func (device Device) GetCmdArchiveDESRegistration() (cmd Job) {
	// fmt.Printf("\n(device) GetCmdArchiveDESRegistration() for: %s\n", device.DESDevSerial)
	qry := pkg.DES.DB.
		Table("des_devs AS d").
		Select("d.*, j.*").
		Joins("JOIN des_jobs AS j ON d.des_dev_id = j.des_job_dev_id").
		Where("d.des_dev_serial = ? AND j.des_job_name LIKE ?",
			device.DESDevSerial, device.CmdArchiveName())

	res := qry.Scan(&cmd.DESRegistration)
	if res.Error != nil {
		pkg.TraceErr(res.Error)
	}
	// pkg.Json("(device *Device) GetCmdArchiveDESRegistration( )", cmd)
	return
}

/* CONNECTS THE CMDARCHIVE DBClient TO THE CMDARCHIVE DATABASE */
func (device *Device) ConnectCmdDBC() (err error) {
	device.CmdDBC = pkg.DBClient{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(device.CmdArchiveName()))}
	return device.CmdDBC.Connect()
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

/* CONNECTS THE ACTIVE JOB DBClient TO THE ACTIVE JOB DATABASE */
func (device *Device) ConnectJobDBC() (err error) {
	device.JobDBC = pkg.DBClient{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(device.Job.DESJobName))}
	return device.JobDBC.Connect()
}

/* CALLED WHEN THE DEVICE MQTT CLIENT REVIEVES A 'JOB STARTED' EVENT FROM THE DEVICE */
func (device *Device) StartJob(evt Event) {
	fmt.Printf("\ndevice *Device) StartJob()\n")

	/* WAIT FOR PENDING MQTT MESSAGE TO COMPLETE */
	device.DESMQTTClient.WG.Wait()

	/* CLEAR THE ACTIVE JOB DATABASE CONNECTION */
	device.JobDBC.Disconnect()

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
	}

	fmt.Printf("\ndevice *Device) StartJob() -> CREATE A JOB RECORD IN THE DES DATABASE\n")
	/* CREATE A JOB RECORD IN THE DES DATABASE */
	if res := pkg.DES.DB.Create(&device.Job.DESJob); res.Error != nil {
		pkg.TraceErr(res.Error)
	}

	/* CREATE DESJobSearch RECORD */
	device.HDR.Create_DESJobSearch(device.Job.DESRegistration)

	/* WE AVOID CREATING IF THE DATABASE WAS PRE-EXISTING, LOG TO CMDARCHIVE  */
	if pkg.ADB.CheckDatabaseExists(device.Job.DESJobName) {
		device.JobDBC = device.CmdDBC
		fmt.Printf("\n(device *Device) StartJob( ): DATABASE ALREADY EXISTS! *** LOGGING TO: %s\n", device.JobDBC.GetDBName())
	} else {
		/* CREATE NEW JOB DATABASE */
		pkg.ADB.CreateDatabase(device.Job.DESJobName)

		/* CONNECT TO THE NEW ACTIVE JOB DATABASE, ON FAILURE, LOG TO CMDARCHIVE */
		if err := device.ConnectJobDBC(); err != nil {
			device.JobDBC = device.CmdDBC
			fmt.Printf("\n(device *Device) StartJob( ): CONNECTION FAILED! *** LOGGING TO: %s\n", device.JobDBC.GetDBName())

		} else {
			fmt.Printf("\n(device *Device) StartJob( ): CONNECTED TO: %s\n", device.JobDBC.GetDBName())

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

			for _, typ := range EVENT_TYPES {
				device.JobDBC.Create(&typ)
			}
		}
	}

	/* WRITE INITIAL JOB RECORDS */
	if err := device.JobDBC.Write(&device.ADM); err != nil {
		pkg.TraceErr(err)
	}
	if err := device.JobDBC.Write(&device.HDR); err != nil {
		pkg.TraceErr(err)
	}
	if err := device.JobDBC.Write(&device.CFG); err != nil {
		pkg.TraceErr(err)
	}
	if err := device.JobDBC.Write(&evt); err != nil {
		pkg.TraceErr(err)
	}

	/* WAIT FOR PENDING MQTT MESSAGES TO COMPLETE */
	device.DESMQTTClient.WG.Wait()

	/* UPDATE THE DEVICE EVENT CODE, ENABLING MQTT MESSAGE WRITES TO ACTIVE JOB DB
	AFTER WE HAVE WRITTEN THE INITIAL JOB RECORDS
	*/
	device.EVT = evt

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, *device)

	fmt.Printf("\n(device *Device) StartJob( ): COMPLETE: %s\n", device.JobDBC.GetDBName())
}

/* CALLED WHEN THE DEVICE MQTT CLIENT REVIEVES A 'JOB ENDED' EVENT FROM THE DEVICE */
func (device *Device) EndJob(evt Event) {

	/* WAIT FOR PENDING MQTT MESSAGE TO COMPLETE */
	device.DESMQTTClient.WG.Wait()

	// /* WAIT FOR FINAL HEADER TO BE RECEIVED */
	// fmt.Printf("\n(device *Device) EndJob( ) -> Waiting for final Header... H: %d : E: %d", device.HDR.HdrTime, evt.EvtTime)
	// for device.HDR.HdrTime < evt.EvtTime {
	// 	/* THIS IS A SHITE SOLUTION AND LIKELY UNNECESSARY...

	// 		MQTT MESSAGES COMING FROM THE SIMULATION ARRIVE MUCH QUICKER THAN THEY WILL IN REALITY.
	// 		THESE MESSAGES ARE PROCESSED IN GO ROUTINES SO A SHORT MESSAGE (JOB ENDED EVENT) CAN END UP
	// 		BEING PROCESSED BEFORE A LARGER MESSAGE (HEADER), EVEN IF THE SHORT MESSAGE ARRIVED LAST.  
			
	// 		WE'LL DO SOME TESTING ONCE THE REAL DEVICES ARE UP AND RUNNING
	// 	*/
	// } 
	// fmt.Printf("\n(device *Device) EndJob( ) -> Final Header received. H: %d : E: %d", device.HDR.HdrTime, evt.EvtTime)

	/* WRITE END JOB REQUEST EVENT AS RECEIVED TO JOB */
	device.JobDBC.Write(evt)

	/* UPDATE THE DEVICE EVENT CODE, DISABLING MQTT MESSAGE WRITES TO ACTIVE JOB DB	*/
	device.EVT = evt

	/* CLEAR THE ACTIVE JOB DATABASE CONNECTION */
	device.JobDBC.Disconnect()

	/* CLOSE DES JOB */
	device.Job.DESJob.DESJobRegTime = evt.EvtTime
	device.Job.DESJob.DESJobRegAddr = evt.EvtAddr
	device.Job.DESJob.DESJobRegUserID = evt.EvtUserID
	device.Job.DESJob.DESJobRegApp = evt.EvtApp
	device.Job.DESJob.DESJobEnd = evt.EvtTime
	fmt.Printf("\n(device *Device) EndJob( ) ENDING: %s\n", device.HDR.HdrJobName)
	pkg.DES.DB.Save(device.Job.DESJob)

	/* UPDATE DES CMDARCHIVE */
	cmd := device.GetCmdArchiveDESRegistration()
	cmd.DESJobRegTime = time.Now().UTC().UnixMilli() // WE WANT THIS TO BE THE LATEST
	cmd.DESJobRegAddr = evt.EvtAddr
	cmd.DESJobRegUserID = evt.EvtUserID
	cmd.DESJobRegApp = evt.EvtApp
	pkg.DES.DB.Save(cmd.DESJob)

	/* ENSURE WE CATCH STRAY SIGNALS IN THE CMDARCHIVE */
	device.Job = cmd
	device.ConnectJobDBC()

	/* RETURN DEVICE CLIENT DATA TO DEFAULT STATE */
	device.ADM.DefaultSettings_Admin(cmd.DESRegistration)
	device.HDR.DefaultSettings_Header(cmd.DESRegistration)
	device.CFG.DefaultSettings_Config(cmd.DESRegistration)

	/* RETURN DEVICE (PHYSICAL) DATA TO DEFAULT STATE */
	device.MQTTPublication_DeviceClient_CMDAdmin(device.ADM)
	device.MQTTPublication_DeviceClient_CMDHeader(device.HDR)
	device.MQTTPublication_DeviceClient_CMDConfig(device.CFG)
	device.SMP = Sample{ SmpTime: cmd.DESJobRegTime, SmpJobName: cmd.DESJobName }
	// pkg.Json("(device *Device) EndJob(): -> Devices[device.DESDevSerial] AFTER UPDATE", device)

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, *device)

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

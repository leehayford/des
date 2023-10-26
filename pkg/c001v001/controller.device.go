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

/* STATUS ( Event.EvtCode ) ****************************************************************************/
const STATUS_DES_REG_REQ int32 = 0    // USER REQUEST -> CHANGE DEVICE'S OPERATIONAL DATA EXCHANGE SERVER
const STATUS_DES_REGISTERED int32 = 1 // DEVICE RESPONSE -> SENT TO NEW DATA EXCHANGE SERVER
const STATUS_JOB_ENDED int32 = 2      // DEVICE RESPONSE -> JOB ENDED
const STATUS_JOB_START_REQ int32 = 3  // USER REQUEST -> START JOB

/* STATUS > JOB_START_REQ MEANS WE ARE LOGGING TO AN ACTIVE JOB */
const STATUS_JOB_STARTED int32 = 4 // DEVICE RESPONSE -> JOB STARTED
const STATUS_JOB_END_REQ int32 = 5 // USER REQUEST -> END JOB

/* TODO: TEST EVENT DRIVEN STATUS VS .../cmd/topic/report DRIVEN STATUS */
// const STATUS_ADM_REQ int32 = 6 // USER REQUEST -> GET CURRENT ADM
// const STATUS_HWID_REQ int32 = 7 // USER REQUEST -> GET CURRENT HWID
// const STATUS_HDR_REQ int32 = 8 // USER REQUEST -> GET CURRENT HDR
// const STATUS_CFG_REQ int32 = 9 // USER REQUEST -> GET CURRENT CFG

/* END STATUS ( Event.EvtCode ) ***********************************************************************/

/* VALVE POSITIONS ***********************************************************************************/
const MODE_BUILD int32 = 0
const MODE_VENT int32 = 2
const MODE_HI_FLOW int32 = 4
const MODE_LO_FLOW int32 = 6

/* END VALVE POSITIONS ******************************************************************************/

const MIN_SAMPLE_PERIOD int32 = 200

/*
FOR EACH REGISTERED DEVICE, THE DES MAINTAINS:

# THE MOST RECENT REGISTRATION DATA FOR THE DEVICE ITSELF, AND THE ACTIVE JOB

# THE MOST RECENT MESSAGE DATA FROM THE DEVICE, ONE OF EACH DATA MODEL PRESENT IN A JOB DATABASE

SEVERAL DEDICATED CONNECTIONS:

	A DEVICE-SPECIFIC CMDARCHIVE DATABASE ( FOR LIFE )
	A DEVICE-SPECIFIC ACTIVE JOB DATABASE ( CHANGES WITH EACH JOB START )
	DEVICE-SPECIFIC MQTT CLIENT ( FOR LIFE )
*/
type Device struct {
	pkg.DESRegistration `json:"reg"` // Contains registration data for both the device and active job
	ADM                 Admin        `json:"adm"` // Last known Admin value
	HW                  HwID         `json:"hw"`  // Last known HwID value
	HDR                 Header       `json:"hdr"` // Last known Header value
	CFG                 Config       `json:"cfg"` // Last known Config value
	EVT                 Event        `json:"evt"` // Last known Event value
	SMP                 Sample       `json:"smp"` // Last known Sample value

	/****************************************************************************************************/
	/* TODO: REMOVE Job STRUCT FROM DEVICE SO Job CAN BE DEDICATED TO REPORTING */
	Job `json:"job"` // The active job for this device ( CMDARCHIVE when between jobs )
	/****************************************************************************************************/

	CmdDBC            pkg.DBClient `json:"-"` // Database Client for the CMDARCHIVE
	JobDBC            pkg.DBClient `json:"-"` // Database Client for the active job
	pkg.DESMQTTClient `json:"-"`   // MQTT client handling all subscriptions and publications for this device
	
}

type DevicesMap map[string]Device

var Devices = make(DevicesMap)
var DevicesRWMutex = sync.RWMutex{}

/* TODO: TEST WaitGroup vs. RWMutex ON SERVER TO PREVENT CONCURRENT MAP WRITES
--> var DevicesMapWG = sync.WaitGroup{}
FAILING THAT, LOOK INTO:
	1. Channel-based updates
	2. Log-based, Change Data Capture ( CDC ) updates
*/

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
	// pkg.Json("GetDeviceList(): DESRegistrations", res)
	err = res.Error
	return
}

/* GET THE MAPPED DATA FOR ALL DEVICES IN THE LIST OF DESRegistrations */
func GetDevices(regs []pkg.DESRegistration) (devices []Device) {
	for _, reg := range regs {
		// pkg.Json("GetDevices( ) -> reg", reg)
		device := (&Device{}).ReadDevicesMap(reg.DESDevSerial)
		device.DESRegistration = reg
		devices = append(devices, device)
	}
	// pkg.Json("GetDevices(): Devices", devices)
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
	UpdateDevicesMap(device.DESDevSerial, *device)
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

/*
	READ THE DevicesMap

TODO: TEST WaitGroup vs. RWMutex ON SERVER TO PREVENT CONCURRENT MAP WRITES
*/
func (device *Device) ReadDevicesMap(serial string) (d Device) {

	DevicesRWMutex.Lock()
	d = Devices[serial]
	DevicesRWMutex.Unlock()

	return
}

/* HYDRATES THE DEVICE'S DB & MQTT CLIENT OBJECTS OF THE DEVICE FROM DevicesMap */
func (device *Device) GetMappedClients() {

	/* GET THE DEVICE CLIENT DATA FROM THE DEVICES CLIENT MAP */
	d := device.ReadDevicesMap(device.DESDevSerial)

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
	d := device.ReadDevicesMap(device.DESDevSerial)
	device.ADM = d.ADM
}

/* HYDRATES THE DEVICE'S HwID STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedHW() {
	d := device.ReadDevicesMap(device.DESDevSerial)
	device.HW = d.HW
}

/* HYDRATES THE DEVICE'S HEader STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedHDR() {
	d := device.ReadDevicesMap(device.DESDevSerial)
	device.HDR = d.HDR
}

/* HYDRATES THE DEVICE'S Config STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedCFG() {
	d := device.ReadDevicesMap(device.DESDevSerial)
	device.CFG = d.CFG
}

/* HYDRATES THE DEVICE'S Event STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedEVT() {
	d := device.ReadDevicesMap(device.DESDevSerial)
	device.EVT = d.EVT
}

/* HYDRATES THE DEVICE'S Sample STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedSMP() {
	d := device.ReadDevicesMap(device.DESDevSerial)
	device.SMP = d.SMP
}

/*
	UPDATE THE DevicesMap

TODO: TEST WaitGroup vs. RWMutex ON SERVER TO PREVENT CONCURRENT MAP WRITES
*/
func UpdateDevicesMap(serial string, d Device) {

	DevicesRWMutex.Lock()
	Devices[serial] = d
	DevicesRWMutex.Unlock()
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Admin */
func (device *Device) UpdateMappedADM() {
	d := device.ReadDevicesMap(device.DESDevSerial)
	d.ADM = device.ADM
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT HwID */
func (device *Device) UpdateMappedHW() {
	d := device.ReadDevicesMap(device.DESDevSerial)
	d.HW = device.HW
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Header */
// func (device *Device) UpdateMappedHDR(hdr Header) {
func (device *Device) UpdateMappedHDR() {
	d := device.ReadDevicesMap(device.DESDevSerial)
	// d.HDR = hdr
	d.HDR = device.HDR
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Config */
func (device *Device) UpdateMappedCFG() {
	d := device.ReadDevicesMap(device.DESDevSerial)
	d.CFG = device.CFG
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Event */
func (device *Device) UpdateMappedEVT() {
	d := device.ReadDevicesMap(device.DESDevSerial)
	d.EVT = device.EVT
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Sample */
func (device *Device) UpdateMappedSMP() {
	d := device.ReadDevicesMap(device.DESDevSerial)
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

/* START JOB **********************************************************************************************/

/* PREPARE, LOG, AND SEND A START JOB REQUEST */
func (device *Device) StartJobRequest(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DESMQTTClient.WG = &sync.WaitGroup{}
	device.GetMappedClients()

	/* START NEW JOB
	MAKE ADM, HDR, CFG, EVT ( START JOB )
	ENSURE ADM, HDR, CFG, & EVT HAVE THE SAME TIME STAMP / SIGNATURE
	*/
	startTime := time.Now().UTC().UnixMilli()

	device.DESRegistration.DESJobRegTime = startTime
	// device.Job.DESRegistration = device.DESRegistration

	device.ADM.AdmTime = startTime
	device.ADM.AdmAddr = src
	device.ADM.AdmUserID = device.DESJobRegUserID
	device.ADM.AdmApp = device.DESJobRegApp
	device.ADM.AdmDefHost = pkg.MQTT_HOST
	device.ADM.AdmDefPort = pkg.MQTT_PORT
	device.ADM.AdmOpHost = pkg.MQTT_HOST
	device.ADM.AdmOpPort = pkg.MQTT_PORT
	device.ADM.Validate()
	// pkg.Json("HandleStartJob(): -> device.ADM", device.ADM)

	device.HW.HwTime = startTime
	device.HW.HwAddr = src
	device.HW.HwUserID = device.DESJobRegUserID
	device.HW.HwApp = device.DESJobRegApp
	device.HW.HwSerial = device.DESDevSerial
	device.HW.HwVersion = DEVICE_VERSION
	device.HW.HwClass = DEVICE_CLASS
	device.HW.Validate()
	// pkg.Json("HandleStartJob(): -> device.HW", device.HW)

	device.HDR.HdrTime = startTime
	device.HDR.HdrAddr = src
	device.HDR.HdrUserID = device.DESJobRegUserID
	device.HDR.HdrApp = device.DESJobRegApp
	device.HDR.HdrJobName = device.CmdArchiveName()
	device.HDR.HdrJobStart = startTime // This is displays the time/date of the request while pending
	device.HDR.HdrJobEnd = -1          // This means there is a pending request for the device to start a new job
	device.HDR.HdrGeoLng = -180
	device.HDR.HdrGeoLat = 90
	device.HDR.Validate()
	// pkg.Json("HandleStartJob(): -> device.HDR", device.HDR)

	device.CFG.CfgTime = startTime
	device.CFG.CfgAddr = src
	device.CFG.CfgUserID = device.DESJobRegUserID
	device.CFG.CfgApp = device.DESJobRegApp
	device.CFG.Validate()
	// pkg.Json("HandleStartJob(): -> device.CFG", device.CFG)

	device.EVT = Event{
		EvtTime:   startTime,
		EvtAddr:   src,
		EvtUserID: device.DESJobRegUserID,
		EvtApp:    device.DESJobRegApp,
		EvtCode:   STATUS_JOB_START_REQ,
		EvtTitle:  "START JOB REQUEST",
		EvtMsg:    "",
	}

	/* LOG START JOB REQUEST TO CMDARCHIVE */
	device.CmdDBC.Create(&device.ADM)
	device.CmdDBC.Create(&device.HW)
	device.CmdDBC.Create(&device.HDR)
	device.CmdDBC.Create(&device.CFG)
	device.CmdDBC.Create(&device.EVT)

	// /* MQTT PUB CMD: ADM, HDR, CFG, EVT */
	fmt.Printf("\nHandleStartJob( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDAdmin(device.ADM)
	device.MQTTPublication_DeviceClient_CMDHwID(device.HW)
	device.MQTTPublication_DeviceClient_CMDHeader(device.HDR)
	device.MQTTPublication_DeviceClient_CMDConfig(device.CFG)
	device.MQTTPublication_DeviceClient_CMDEvent(device.EVT)

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, *device)

	return
}

/* CALLED WHEN THE DEVICE MQTT CLIENT REVIEVES A 'JOB STARTED' EVENT FROM THE DEVICE */
func (device *Device) StartJob(evt Event) {
	fmt.Printf("\n(device *Device) StartJob()\n")

	// /* WAIT FOR PENDING MQTT MESSAGE TO COMPLETE */
	// device.DESMQTTClient.WG.Wait()

	/* CLEAR THE ACTIVE JOB DATABASE CONNECTION */
	device.JobDBC.Disconnect()
	hdr := device.HDR
	
	device.Job = Job{
		DESRegistration: pkg.DESRegistration{
			DESDev: device.DESDev,
			DESJob: pkg.DESJob{
				DESJobRegTime:   hdr.HdrTime,
				DESJobRegAddr:   hdr.HdrAddr,
				DESJobRegUserID: hdr.HdrUserID,
				DESJobRegApp:    hdr.HdrApp,

				DESJobName:  hdr.HdrJobName,
				DESJobStart: hdr.HdrJobStart,
				DESJobEnd:   0,
				DESJobLng:   hdr.HdrGeoLng,
				DESJobLat:   hdr.HdrGeoLat,
				DESJobDevID: device.DESDevID,
			},
		},
	}

	fmt.Printf("\n(device *Device) StartJob() -> CREATE A JOB RECORD IN THE DES DATABASE\n")
	/* CREATE A JOB RECORD IN THE DES DATABASE */
	if res := pkg.DES.DB.Create(&device.Job.DESJob); res.Error != nil {
		pkg.TraceErr(res.Error)
	}

	/* CREATE DESJobSearch RECORD */
	hdr.Create_DESJobSearch(device.Job.DESRegistration)
	// device.HDR.Create_DESJobSearch(device.Job.DESRegistration)

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
				&HwID{},
				&Header{},
				&Config{},
				&EventTyp{},
				&Event{},
				&Sample{},
			); err != nil {
				pkg.TraceErr(err)
			}

			for _, typ := range EVENT_TYPES {
				WriteETYP(typ, &device.JobDBC)
			}
		}
	}

	/* WRITE INITIAL JOB RECORDS */
	if err := WriteADM(device.ADM, &device.JobDBC); err != nil {
		pkg.TraceErr(err)
	}
	if err := WriteHW(device.HW, &device.JobDBC); err != nil {
		pkg.TraceErr(err)
	}
	if err := WriteHDR(hdr, &device.JobDBC); err != nil {
	// if err := WriteHDR(device.HDR, &device.JobDBC); err != nil {
		pkg.TraceErr(err)
	}
	if err := WriteCFG(device.CFG, &device.JobDBC); err != nil {
		pkg.TraceErr(err)
	}
	if err := WriteEVT(evt, &device.JobDBC); err != nil {
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

/* END JOB ************************************************************************************************/

/* PREPARE, LOG, AND SEND AN END JOB REQUEST */
func (device *Device) EndJobRequest(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.GetMappedHW()
	device.GetMappedHDR()
	device.GetMappedCFG()
	device.GetMappedSMP()
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DESMQTTClient.WG = &sync.WaitGroup{}
	device.GetMappedClients()

	device.EVT = Event{
		EvtTime:   time.Now().UTC().UnixMilli(),
		EvtAddr:   src,
		EvtUserID: device.DESJobRegUserID,
		EvtApp:    device.DESJobRegApp,
		EvtCode:   STATUS_JOB_END_REQ,
		EvtTitle:  "END JOB REQUEST",
		EvtMsg:    "",
	}

	/* LOG END JOB REQUEST TO CMDARCHIVE */ // fmt.Printf("\nHandleEndJob( ) -> Write to %s \n", device.CmdArchiveName())
	device.CmdDBC.Create(&device.EVT)

	/* LOG END JOB REQUEST TO ACTIVE JOB */ // fmt.Printf("\nHandleEndJob( ) -> Write to %s \n", device.DESJobName)
	device.EVT.EvtID = 0
	device.JobDBC.Create(&device.EVT)

	/* MQTT PUB CMD: EVT */
	fmt.Printf("\nHandleEndJob( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDEvent(device.EVT)

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, *device)
	return
}

/* CALLED WHEN THE DEVICE MQTT CLIENT REVIEVES A 'JOB ENDED' EVENT FROM THE DEVICE */
func (device *Device) EndJob(evt Event) {

	// /* WAIT FOR PENDING MQTT MESSAGE TO COMPLETE */
	// device.DESMQTTClient.WG.Wait()

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
	WriteEVT(evt, &device.JobDBC)
	// device.JobDBC.Write(evt)

	/* UPDATE THE DEVICE EVENT CODE, DISABLING MQTT MESSAGE WRITES TO ACTIVE JOB DB	*/
	device.EVT = evt

	/* CLEAR THE ACTIVE JOB DATABASE CONNECTION */
	device.JobDBC.Disconnect()

	jobName := device.Job.DESJobName
	/* CLOSE DES JOB */
	device.Job.DESJob.DESJobRegTime = evt.EvtTime
	device.Job.DESJob.DESJobRegAddr = evt.EvtAddr
	device.Job.DESJob.DESJobRegUserID = evt.EvtUserID
	device.Job.DESJob.DESJobRegApp = evt.EvtApp
	device.Job.DESJob.DESJobEnd = evt.EvtTime
	fmt.Printf("\n(device *Device) EndJob( ) ENDING: %s\n", jobName)
	pkg.DES.DB.Save(device.Job.DESJob)

	/* UPDATE DES CMDARCHIVE */
	cmd := device.GetCmdArchiveDESRegistration()
	cmd.DESJobRegTime = time.Now().UTC().UnixMilli() // WE WANT THIS TO BE THE LATEST
	cmd.DESJobRegAddr = evt.EvtAddr
	cmd.DESJobRegUserID = evt.EvtUserID
	cmd.DESJobRegApp = evt.EvtApp
	cmd.DESJob.DESJobEnd = 0 // ENSURE THE DEVICE IS DISCOVERABLE
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
	device.SMP = Sample{SmpTime: cmd.DESJobRegTime, SmpJobName: cmd.DESJobName}
	// pkg.Json("(device *Device) EndJob(): -> Devices[device.DESDevSerial] AFTER UPDATE", device)

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, *device)

	fmt.Printf("\n(device *Device) EndJob( ) COMPLETE: %s\n", jobName)
}

/* SET / GET JOB PARAMS *********************************************************************************/

/* PREPARE, LOG, AND SEND A SET ADMIN REQUEST TO THE DEVICE */
func (device *Device) SetAdminRequest(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
	device.ADM.AdmTime = time.Now().UTC().UnixMilli()
	device.ADM.AdmAddr = src
	device.ADM.Validate() // fmt.Printf("\nHandleSetAdmin( ) -> ADM Validated")
	device.GetMappedHW()  // fmt.Printf("\nHandleSetAdmin( ) -> Mapped HW gotten")
	device.GetMappedHDR() // fmt.Printf("\nHandleSetAdmin( ) -> Mapped HDR gotten")
	device.GetMappedCFG() // fmt.Printf("\nHandleSetAdmin( ) -> Mapped CFG gotten")
	device.GetMappedEVT() // fmt.Printf("\nHandleSetAdmin( ) -> Mapped EVT gotten")
	device.GetMappedSMP() // fmt.Printf("\nHandleSetAdmin( ) -> Mapped SMP gotten")
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DESMQTTClient.WG = &sync.WaitGroup{}
	device.GetMappedClients() // fmt.Printf("\nHandleSetAdmin( ) -> Mapped Clients gotten")

	/* LOG ADM CHANGE REQUEST TO  CMDARCHIVE */
	adm := device.ADM
	device.CmdDBC.Create(&adm)

	/* MQTT PUB CMD: ADM */
	fmt.Printf("\nHandleSetAdmin( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDAdmin(adm)

	/* UPDATE DevicesMap */
	UpdateDevicesMap(device.DESDevSerial, *device)

	return
}

/*
	TODO: DO NOT USE

TEST EVENT DRIVEN STATUS VS .../cmd/topic/report DRIVEN STATUS
REQUEST THE CURRENT ADMIN FROM THE DEVICE
*/
func (device *Device) GetAdminRequest(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
	device.ADM.AdmTime = time.Now().UTC().UnixMilli()
	device.ADM.AdmAddr = src
	device.ADM.Validate()
	device.GetMappedHW()
	device.GetMappedHDR()
	device.GetMappedCFG()
	device.GetMappedEVT()
	device.GetMappedSMP()
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DESMQTTClient.WG = &sync.WaitGroup{}
	device.GetMappedClients()

	/* MQTT PUB CMD: ADM */
	fmt.Printf("\nGetAdminRequest( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDAdminReport(device.ADM)

	/* TODO: DO NOT USE
	TEST EVENT DRIVEN STATUS VS .../cmd/topic/report DRIVEN STATUS
	*/
	// device.EVT = Event{
	// 	EvtTime:   time.Now().UTC().UnixMilli(),
	// 	EvtAddr:   src,
	// 	EvtUserID: device.DESJobRegUserID,
	// 	EvtApp:    device.DESJobRegApp,
	// 	// EvtCode:   STATUS_ADM_REQ,
	// 	EvtTitle:  "GET ADM REQUEST",
	// 	EvtMsg:    "",
	// }

	// device.DESMQTTClient = pkg.DESMQTTClient{}
	// device.DESMQTTClient.WG = &sync.WaitGroup{}
	// device.GetMappedClients()

	// /* MQTT PUB CMD: ADM */
	// fmt.Printf("\nGetAdminRequest( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	// device.MQTTPublication_DeviceClient_CMDEvent(device.EVT)

	return
}

/* REQUEST THE CURRENT HARDWARE ID FROM THE DEVICE */
func (device *Device) GetHwIDRequest(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.GetMappedHDR()
	device.HW.HwTime = time.Now().UTC().UnixMilli()
	device.HW.HwAddr = src
	device.HW.Validate()
	device.GetMappedCFG()
	device.GetMappedEVT()
	device.GetMappedSMP()
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DESMQTTClient.WG = &sync.WaitGroup{}
	device.GetMappedClients()

	/* MQTT PUB CMD: HARDWARE ID */
	fmt.Printf("\nGetHwIDRequest( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDHwID(device.HW)

	/* TODO: DO NOT USE
	TEST EVENT DRIVEN STATUS VS .../cmd/topic/report DRIVEN STATUS
	*/
	// device.EVT = Event{
	// 	EvtTime:   time.Now().UTC().UnixMilli(),
	// 	EvtAddr:   src,
	// 	EvtUserID: device.DESJobRegUserID,
	// 	EvtApp:    device.DESJobRegApp,
	// 	// EvtCode:   STATUS_HW_REQ,
	// 	EvtTitle:  "GET HW REQUEST",
	// 	EvtMsg:    "",
	// }

	// /* MQTT PUB CMD: HARDWARE ID */
	// fmt.Printf("\nGetHwIDRequest( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	// device.MQTTPublication_DeviceClient_CMDEvent(device.EVT)

	return
}

/* PREPARE, LOG, AND SEND A SET HEADER REQUEST TO THE DEVICE */
func (device *Device) SetHeaderRequest(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.HDR.HdrTime = time.Now().UTC().UnixMilli()
	device.HDR.HdrAddr = src
	device.HDR.Validate()
	device.GetMappedHW()
	device.GetMappedCFG()
	device.GetMappedEVT()
	device.GetMappedSMP()
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DESMQTTClient.WG = &sync.WaitGroup{}
	device.GetMappedClients()

	/* LOG HDR CHANGE REQUEST TO CMDARCHIVE */
	hdr := device.HDR
	device.CmdDBC.Create(&hdr)

	/* MQTT PUB CMD: HDR */
	fmt.Printf("\nHandleSetHeader( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDHeader(hdr)

	/* UPDATE DevicesMap */
	UpdateDevicesMap(device.DESDevSerial, *device)

	return
}

/* REQUEST THE CURRENT HEADER FROM THE DEVICE */

/* PREPARE, LOG, AND SEND A SET CONFIG REQUEST TO THE DEVICE */
func (device *Device) SetConfigRequest(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.GetMappedHW()
	device.GetMappedHDR()
	device.CFG.CfgTime = time.Now().UTC().UnixMilli()
	device.CFG.CfgAddr = src
	device.CFG.Validate()
	device.GetMappedEVT()
	device.GetMappedSMP()
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DESMQTTClient.WG = &sync.WaitGroup{}
	device.GetMappedClients()

	/* LOG CFG CHANGE REQUEST TO CMDARCHIVE */
	cfg := device.CFG
	device.CmdDBC.Create(&cfg)

	/* MQTT PUB CMD: CFG */
	fmt.Printf("\nHandleSetConfig( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDConfig(cfg)

	/* UPDATE DevicesMap */
	UpdateDevicesMap(device.DESDevSerial, *device)

	return
}

/* REQUEST THE CURRENT CONFIG FROM THE DEVICE */

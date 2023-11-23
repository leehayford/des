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

const DEFAULT_GEO_LNG = -180 // TODO: TEST -999.25
const DEFAULT_GEO_LAT = 90   // TODO: TEST -999.25

/* OPERATION CODES ( Event.EvtCode 0 : 999 ) *******************************************************/
const OP_CODE_DES_REG_REQ int32 = 0    // USER REQUEST -> CHANGE DEVICE'S OPERATIONAL DATA EXCHANGE SERVER
const OP_CODE_DES_REGISTERED int32 = 1 // DEVICE RESPONSE -> SENT TO NEW DATA EXCHANGE SERVER
const OP_CODE_JOB_ENDED int32 = 2      // DEVICE RESPONSE -> JOB ENDED
const OP_CODE_JOB_START_REQ int32 = 3  // USER REQUEST -> START JOB
const OP_CODE_JOB_STARTED int32 = 4    // DEVICE RESPONSE -> JOB STARTED
const OP_CODE_JOB_END_REQ int32 = 5    // USER REQUEST -> END JOB
/* END OPERATION CODES  ( Event.EvtCode ) *********************************************************/

/* STATUS CODES ( Event.EvtCode 1000 : 1999 ) *******************************************************/
const STATUS_BAT_HIGH_AMP int32 = 1000
const STATUS_BAT_LOW_VOLT int32 = 1001
const STATUS_MOT_HIGH_AMP int32 = 1002
const STATUS_MAX_PRESSURE int32 = 1003
const STATUS_HFS_MAX_FLOW int32 = 1004
const STATUS_HFS_MAX_PRESS int32 = 1005
const STATUS_HFS_MAX_DIFF int32 = 1006
const STATUS_LFS_MAX_FLOW int32 = 1007
const STATUS_LFS_MAX_PRESS int32 = 1008
const STATUS_LFS_MAX_DIFF int32 = 1009

/* END STATUS CODES ( Event.EvtCode ) **************************************************************/

/* MODE ( VALVE POSITIONS ) *************************************************************************/
const MODE_BUILD int32 = 0
const MODE_VENT int32 = 2
const MODE_HI_FLOW int32 = 4
const MODE_LO_FLOW int32 = 6

/* END VALVE POSITIONS ******************************************************************************/

const MIN_SAMPLE_PERIOD int32 = 1000

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
	pkg.DESRegistration `json:"reg"`  // Contains registration data for both the device and active job
	ADM                 Admin         `json:"adm"`      // Last known Admin value
	STA                 State         `json:"sta"`      // Last known State value
	HDR                 Header        `json:"hdr"`      // Last known Header value
	CFG                 Config        `json:"cfg"`      // Last known Config value
	EVT                 Event         `json:"evt"`      // Last known Event value
	SMP                 Sample        `json:"smp"`      // Last known Sample value
	DBG                 Debug         `json:"dbg"`      // Settings used while debugging
	PING                pkg.Ping      `json:"ping"`     // Last Ping received from device
	DESPING             pkg.Ping      `json:"des_ping"` // Last Ping sent from this DES device client
	DESPingStop         chan struct{} `json:"-"`        // Send DESPingStop when DeviceClients are disconnected
	CmdDBC              pkg.DBClient  `json:"-"`        // Database Client for the CMDARCHIVE
	JobDBC              pkg.DBClient  `json:"-"`        // Database Client for the active job
	pkg.DESMQTTClient   `json:"-"`    // MQTT client handling all subscriptions and publications for this device

}

type Debug struct {
	MQTTDelay int32 `json:"mqtt_delay"`
}

type DevicesMap map[string]Device

var Devices = make(DevicesMap)
var DevicesRWMutex = sync.RWMutex{}

const DEVICE_PING_TIMEOUT = 30000
const DEVICE_PING_LIMIT = DEVICE_PING_TIMEOUT + 1000

const DES_PING_TIMEOUT = 10000
const DES_PING_LIMIT = DEVICE_PING_TIMEOUT + 1000

type DevicePingsMap map[string]pkg.Ping

var DeviceClientPings = make(DevicePingsMap)

func (device *Device) UpdateDeviceClientPing(ping pkg.Ping) {

	/* UPDATE device.PING AND DevicePings MAP */
	DeviceClientPings[device.DESDevSerial] = ping
	device.DESPING = ping

	/* CALL IN GO ROUTINE  *** DES TOPIC *** - ALERT USER CLIENTS */
	go device.MQTTPublication_DeviceClient_DESDeviceClientPing(ping)
}

var DevicePings = make(DevicePingsMap)

func (device *Device) UpdateDevicePing(ping pkg.Ping) {

	if !ping.OK || ping.Time == 0 {
		ping.Time = DevicePings[device.DESDevSerial].Time
		ping.OK = false
		// fmt.Printf("\n%s -> UpdateDevicePing( ) -> Timeout.", device.DESDevSerial )
	}

	/* UPDATE device.PING AND DevicePings MAP */
	DevicePings[device.DESDevSerial] = ping
	device.PING = ping
}

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

	// qry := pkg.DES.DB.
	// 	Table("des_jobs").
	// 	Select("des_devs.*, des_jobs.*, des_job_searches.*").
	// 	Joins(`JOIN ( ? ) j ON des_jobs.des_job_dev_id = j.des_job_dev_id AND des_job_reg_time = j.max_time`, subQryLatestJob).
	// 	Joins("JOIN des_devs ON des_devs.des_dev_id = j.des_job_dev_id").
	// 	Joins("JOIN des_job_searches ON des_jobs.des_job_id = des_job_searches.des_job_key").
	// 	Order("des_devs.des_dev_serial DESC")

	res := qry.Scan(&devices)
	// pkg.Json("GetDeviceList(): DESRegistrations", res)
	err = res.Error
	return
}

/* GET THE MAPPED DATA FOR ALL DEVICES IN THE LIST OF DESRegistrations */
func GetDevices(regs []pkg.DESRegistration) (devices []Device) {
	for _, reg := range regs {
		// pkg.Json("GetDevices( ) -> reg", reg)
		device := ReadDevicesMap(reg.DESDevSerial)
		device.DESRegistration = reg
		device.PING = DevicePings[device.DESDevSerial]
		device.DESPING = DeviceClientPings[device.DESDevSerial]
		devices = append(devices, device)
	}
	// pkg.Json("GetDevices(): Devices", devices)
	return
}

func (device *Device) GetDeviceDESRegistration(serial string) (err error) {

	res := pkg.DES.DB.
		Order("des_dev_reg_time desc").
		First(&device.DESRegistration.DESDev, "des_dev_serial =?", serial)
	err = res.Error
	return
}

/* RETURNS ALL EVENTS ASSOCIATED WITH THE ACTIVE JOB */
func (device *Device) GetActiveJobEvents() (evts *[]Event, err error) {

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedClients()

	res := device.JobDBC.Select("*").Table("events").Where("evt_addr = ?", device.DESDevSerial).Order("evt_time DESC").Scan(&evts)
	err = res.Error
	return
}

/* CALLED ON SERVER STARTUP */
func DeviceClient_ConnectAll() {

	regs, err := GetDeviceList()
	if err != nil {
		pkg.LogErr(err)
	}

	for _, reg := range regs {
		device := Device{}
		device.DESRegistration = reg
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
func (device *Device) DeviceClient_Connect() (err error) {

	fmt.Printf("\n\n(device *Device) DeviceClient_Connect() -> %s -> connecting... \n", device.DESDevSerial)

	fmt.Printf("\n(device *Device) DeviceClient_Connect() -> %s -> connecting CMDARCHIVE... \n", device.DESDevSerial)
	if err := device.ConnectCmdDBC(); err != nil {
		return pkg.LogErr(err)
	}

	fmt.Printf("\n(device *Device) DeviceClient_Connect() -> %s -> connecting ACTIVE JOB... \n", device.DESDevSerial)
	if err := device.ConnectJobDBC(); err != nil {
		return pkg.LogErr(err)
	}

	if res := device.JobDBC.Last(&device.ADM); res.Error != nil {
		return pkg.LogErr(res.Error)
	}
	if res := device.JobDBC.Last(&device.STA); res.Error != nil {
		return pkg.LogErr(res.Error)
	}
	if res := device.JobDBC.Last(&device.HDR); res.Error != nil {
		return pkg.LogErr(res.Error)
	}
	if res := device.JobDBC.Last(&device.CFG); res.Error != nil {
		return pkg.LogErr(res.Error)
	}
	if res := device.JobDBC.Last(&device.SMP); res.Error != nil {
		return pkg.LogErr(res.Error)
	}
	if res := device.JobDBC.Last(&device.EVT); res.Error != nil {
		return pkg.LogErr(res.Error)
	}

	if err := device.MQTTDeviceClient_Connect(); err != nil {
		return pkg.LogErr(err)
	}

	/* UPDATE DevicePings MAP. START THE KEEP-ALIVE */
	device.UpdateDevicePing(device.PING)

	/* UPDATE DeviceClientPings MAP. START THE KEEP-ALIVE */
	device.UpdateDeviceClientPing(device.DESPING)

	/* START DES DEVICE CLIENT PING */

	device.DESPingStop = make(chan struct{})
	live := true
	go func() {
		for live {
			select {

			case <-device.DESPingStop:
				live = false

			default:
				time.Sleep(time.Millisecond * DES_PING_TIMEOUT)
				device.UpdateDeviceClientPing(pkg.Ping{
					Time: time.Now().UTC().UnixMilli(),
					OK:   true,
				})
				// fmt.Printf("\n(device *Device) DeviceClient_Connect() -> %s -> DES DEVICE CLIENT PING... \n\n", device.DESDevSerial)
			}
		}
		device.DESPING = pkg.Ping{}
		delete(DeviceClientPings, device.DESDevSerial)
		fmt.Printf("\n(device *Device) DeviceClient_Connect() -> %s -> DES DEVICE CLIENT PING STOPPED. \n\n", device.DESDevSerial)
	}()

	/* ADD TO Devices MAP */
	UpdateDevicesMap(device.DESDevSerial, *device)

	fmt.Printf("\n(device *Device) DeviceClient_Connect() -> %s -> connected... \n\n", device.DESDevSerial)
	return
}

/* DISCONNECT DEVICE DATABASE AND MQTT CLIENTS; REMOVE CONNECTED DEVICE FROM DevicesMap */
func (device *Device) DeviceClient_Disconnect() (err error) {
	/* TODO: TEST WHEN IMPLEMENTING
	- UNREGISTER DEVICE
	- GRACEFUL SHUTDOWN
	*/
	fmt.Printf("\n\n(device *Device) DeviceClient_Disconnect() -> %s -> disconnecting... \n", device.DESDevSerial)

	/* KILL DES DEVICE CLIENT PING REMOVE FROM DeviceClientPings MAP */
	device.DESPingStop <- struct{}{}

	if err := device.CmdDBC.Disconnect(); err != nil {
		return pkg.LogErr(err)
	}
	if err := device.JobDBC.Disconnect(); err != nil {
		return pkg.LogErr(err)
	}
	if err := device.MQTTDeviceClient_Disconnect(); err != nil {
		return pkg.LogErr(err)
	}

	/* REMOVE FROM Devices MAP */
	delete(Devices, device.DESDevSerial)

	return
}

/*
	READ THE DevicesMap

TODO: TEST WaitGroup vs. RWMutex ON SERVER TO PREVENT CONCURRENT MAP WRITES
*/
func ReadDevicesMap(serial string) (device Device) {

	DevicesRWMutex.Lock()
	device = Devices[serial]
	DevicesRWMutex.Unlock()

	return
}

/* HYDRATES THE DEVICE'S DB & MQTT CLIENT OBJECTS OF THE DEVICE FROM DevicesMap */
func (device *Device) GetMappedClients() {

	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DESMQTTClient.WG = &sync.WaitGroup{}

	/* GET THE DEVICE CLIENT DATA FROM THE DEVICES CLIENT MAP */
	d := ReadDevicesMap(device.DESDevSerial)

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

// /* HYDRATES THE DEVICE'S Ping STRUCT FROM THE DevicesMap */
// func (device *Device) GetMappedPING() {
// 	d := device.ReadDevicesMap(device.DESDevSerial)
// 	device.PING = d.PING
// }

/* HYDRATES THE DEVICE'S Admin STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedADM() {
	d := ReadDevicesMap(device.DESDevSerial)
	device.ADM = d.ADM
}

/* HYDRATES THE DEVICE'S State STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedSTA() {
	d := ReadDevicesMap(device.DESDevSerial)
	device.STA = d.STA
}

/* HYDRATES THE DEVICE'S Header STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedHDR() {
	d := ReadDevicesMap(device.DESDevSerial)
	device.HDR = d.HDR
}

/* HYDRATES THE DEVICE'S Config STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedCFG() {
	d := ReadDevicesMap(device.DESDevSerial)
	device.CFG = d.CFG
}

/* HYDRATES THE DEVICE'S Event STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedEVT() {
	d := ReadDevicesMap(device.DESDevSerial)
	device.EVT = d.EVT
}

/* HYDRATES THE DEVICE'S Sample STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedSMP() {
	d := ReadDevicesMap(device.DESDevSerial)
	device.SMP = d.SMP
}

/* HYDRATES THE DEVICE'S Debug STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedDBG() {
	d := ReadDevicesMap(device.DESDevSerial)
	device.DBG = d.DBG
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

// /* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Ping */
// func (device *Device) UpdateMappedPING() {
// 	d := device.ReadDevicesMap(device.DESDevSerial)
// 	d.PING = device.PING
// 	UpdateDevicesMap(device.DESDevSerial, d)
// }

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Admin */
func (device *Device) UpdateMappedADM() {
	d := ReadDevicesMap(device.DESDevSerial)
	d.ADM = device.ADM
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT State */
func (device *Device) UpdateMappedSTA() {
	d := ReadDevicesMap(device.DESDevSerial)
	d.STA = device.STA
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Header */
// func (device *Device) UpdateMappedHDR(hdr Header) {
func (device *Device) UpdateMappedHDR() {
	d := ReadDevicesMap(device.DESDevSerial)
	// d.HDR = hdr
	d.HDR = device.HDR
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Config */
func (device *Device) UpdateMappedCFG() {
	d := ReadDevicesMap(device.DESDevSerial)
	d.CFG = device.CFG
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Event */
func (device *Device) UpdateMappedEVT() {
	d := ReadDevicesMap(device.DESDevSerial)
	d.EVT = device.EVT
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Sample */
func (device *Device) UpdateMappedSMP() {
	d := ReadDevicesMap(device.DESDevSerial)
	d.SMP = device.SMP
	UpdateDevicesMap(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Debug */
func (device *Device) UpdateMappedDBG(sync bool) {
	d := ReadDevicesMap(device.DESDevSerial)
	d.DBG = device.DBG
	UpdateDevicesMap(device.DESDevSerial, d)
	if sync {
		device = &d
	}
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
		pkg.LogErr(res.Error)
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
		pkg.LogErr(res.Error)
		return
	}
	// pkg.Json("(device *Device) GetCurrentJob( )", device.Job)
	return
}

/* CONNECTS THE ACTIVE JOB DBClient TO THE ACTIVE JOB DATABASE */
func (device *Device) ConnectJobDBC() (err error) {
	device.JobDBC = pkg.DBClient{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(device.DESJobName))}
	return device.JobDBC.Connect()
}

/* START JOB **********************************************************************************************/

/* PREPARE, LOG, AND SEND A START JOB REQUEST */
func (device *Device) StartJobRequestX(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
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

	device.STA.StaTime = startTime
	device.STA.StaAddr = src
	device.STA.StaUserID = device.DESJobRegUserID
	device.STA.StaApp = device.DESJobRegApp
	device.STA.StaSerial = device.DESDevSerial
	device.STA.StaVersion = DEVICE_VERSION
	device.STA.StaClass = DEVICE_CLASS
	device.STA.StaLogging = OP_CODE_JOB_START_REQ // This means there is a pending request for the device to start a new job
	device.STA.StaJobName = device.CmdArchiveName()
	device.STA.Validate()
	// pkg.Json("HandleStartJob(): -> device.STA", device.STA)

	device.HDR.HdrTime = startTime
	device.HDR.HdrAddr = src
	device.HDR.HdrUserID = device.DESJobRegUserID
	device.HDR.HdrApp = device.DESJobRegApp
	device.HDR.HdrJobStart = startTime // This is displays the time/date of the request while pending
	device.HDR.HdrJobEnd = 0
	device.HDR.HdrGeoLng = DEFAULT_GEO_LNG
	device.HDR.HdrGeoLat = DEFAULT_GEO_LAT
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
		EvtCode:   OP_CODE_JOB_START_REQ,
		EvtTitle:  "START JOB REQUEST",
		EvtMsg:    "",
	}

	/* LOG START JOB REQUEST TO CMDARCHIVE */
	device.CmdDBC.Create(&device.ADM) /* TODO: USE WriteADM... */
	device.CmdDBC.Create(&device.STA) /* TODO: USE WriteSTA... */
	device.CmdDBC.Create(&device.HDR) /* TODO: USE WriteHDR... */
	device.CmdDBC.Create(&device.CFG) /* TODO: USE WriteCFG... */
	device.CmdDBC.Create(&device.EVT) /* TODO: USE WriteEVT... */

	/* MQTT PUB CMD: ADM, HDR, CFG, EVT */
	fmt.Printf("\nHandleStartJob( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)

	device.MQTTPublication_DeviceClient_CMDStartJob()

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, *device)

	return
}

/* PREPARE, LOG, AND SEND A START JOB REQUEST */
func (device *Device) StartJobRequest(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
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

	device.STA.StaTime = startTime
	device.STA.StaAddr = src
	device.STA.StaUserID = device.DESJobRegUserID
	device.STA.StaApp = device.DESJobRegApp
	device.STA.StaSerial = device.DESDevSerial
	device.STA.StaVersion = DEVICE_VERSION
	device.STA.StaClass = DEVICE_CLASS
	device.STA.StaLogging = OP_CODE_JOB_START_REQ // This means there is a pending request for the device to start a new job
	device.STA.StaJobName = device.CmdArchiveName()
	device.STA.Validate()
	// pkg.Json("HandleStartJob(): -> device.STA", device.STA)

	device.HDR.HdrTime = startTime
	device.HDR.HdrAddr = src
	device.HDR.HdrUserID = device.DESJobRegUserID
	device.HDR.HdrApp = device.DESJobRegApp
	device.HDR.HdrJobStart = startTime // This is displays the time/date of the request while pending
	device.HDR.HdrJobEnd = 0
	device.HDR.HdrGeoLng = DEFAULT_GEO_LNG
	device.HDR.HdrGeoLat = DEFAULT_GEO_LAT
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
		EvtCode:   OP_CODE_JOB_START_REQ,
		EvtTitle:  "START JOB REQUEST",
		EvtMsg:    "",
	}

	/* LOG START JOB REQUEST TO CMDARCHIVE */
	device.CmdDBC.Create(&device.ADM) /* TODO: USE WriteADM... */
	device.CmdDBC.Create(&device.STA) /* TODO: USE WriteSTA... */
	device.CmdDBC.Create(&device.HDR) /* TODO: USE WriteHDR... */
	device.CmdDBC.Create(&device.CFG) /* TODO: USE WriteCFG... */
	device.CmdDBC.Create(&device.EVT) /* TODO: USE WriteEVT... */

	/* MQTT PUB CMD: ADM, HDR, CFG, EVT */
	fmt.Printf("\nHandleStartJob( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)

	/* DEBUG: ENABLE MQTT MESSAGE DELAY */
	device.GetMappedDBG()

	device.MQTTPublication_DeviceClient_CMDAdmin(device.ADM)
	time.Sleep(time.Second * time.Duration(device.DBG.MQTTDelay))

	device.MQTTPublication_DeviceClient_CMDState(device.STA)
	time.Sleep(time.Second * time.Duration(device.DBG.MQTTDelay))

	device.MQTTPublication_DeviceClient_CMDHeader(device.HDR)
	time.Sleep(time.Second * time.Duration(device.DBG.MQTTDelay))

	device.MQTTPublication_DeviceClient_CMDConfig(device.CFG)
	time.Sleep(time.Second * time.Duration(device.DBG.MQTTDelay))

	device.MQTTPublication_DeviceClient_CMDEvent(device.EVT)

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, *device)

	return
}

func (device *Device) CancelStartJobRequest(src string) (err error) {
	fmt.Printf("\nCancelStartJobRequest( )...")

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.GetMappedSTA()
	device.STA.StaTime = time.Now().UTC().UnixMilli()
	device.STA.StaAddr = src
	device.STA.StaLogging = OP_CODE_JOB_END_REQ
	device.GetMappedHDR()
	device.GetMappedCFG()
	device.GetMappedSMP()
	device.GetMappedClients()

	device.EVT = Event{
		EvtTime:   time.Now().UTC().UnixMilli(),
		EvtAddr:   src,
		EvtUserID: device.DESJobRegUserID,
		EvtApp:    device.DESJobRegApp,
		EvtCode:   OP_CODE_JOB_END_REQ,
		EvtTitle:  "CANCEL START JOB REQUEST",
		EvtMsg:    "",
	}

	/* LOG CANCEL START JOB REQUEST TO CMDARCHIVE */
	// device.CmdDBC.Create(&device.STA)
	/* TODO: USE WriteSTA... */
	WriteSTA(device.STA, &device.CmdDBC)

	// device.CmdDBC.Create(&device.EVT)
	/* TODO: USE WriteEVT... */
	WriteEVT(device.EVT, &device.CmdDBC)

	/* MQTT PUB CMD: EVT */
	fmt.Printf("\nCancelStartJobRequest( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	// device.MQTTPublication_DeviceClient_CMDState(device.STA)
	device.MQTTPublication_DeviceClient_CMDEvent(device.EVT)

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, *device)
	return err
}
/* CALLED WHEN THE DEVICE MQTT CLIENT REVIEVES A 'JOB STARTED' EVENT FROM THE DEVICE */
func (device *Device) StartJobX(start StartJob) {
	fmt.Printf("\n(device *Device) StartJob() -> start:\n%v\n", start)

	// /* WAIT FOR PENDING MQTT MESSAGE TO COMPLETE */
	// device.DESMQTTClient.WG.Wait()

	/* TODO: CHECK device.STA.StaLogging
	IF WE ARE ALREADY LOGGING, THE ACTIVE JOB MUST BE ENDED BEFORE A NEW ONE IST STARTED
	*/

	pkg.Json("(device *Device) StartJobX(start StartJob): ", start)

	/* CALL DB WRITE IN GOROUTINE */
	WriteADM(start.ADM, &device.CmdDBC)
	WriteSTA(start.STA, &device.CmdDBC)
	WriteHDR(start.HDR, &device.CmdDBC)
	WriteCFG(start.CFG, &device.CmdDBC)
	WriteEVT(start.EVT, &device.CmdDBC)

	/* CLEAR THE ACTIVE JOB DATABASE CONNECTION */
	device.JobDBC.Disconnect()

	device.DESJobRegTime = start.STA.StaTime
	device.DESJobRegAddr = start.STA.StaAddr
	device.DESJobRegUserID = start.STA.StaUserID
	device.DESJobRegApp = start.STA.StaApp

	device.DESJobName = start.STA.StaJobName
	device.DESJobStart = start.STA.StaTime
	device.DESJobEnd = 0
	device.DESJobDevID = device.DESDevID

	/* GET LOCATION DATA */
	if start.HDR.HdrGeoLng < DEFAULT_GEO_LNG  {
		/* Header WAS NOT RECEIVED */
		fmt.Printf("\n(device *Device) StartJob() -> INVALID VALID LOCATION\n")
		device.DESJobLng = DEFAULT_GEO_LNG
		device.DESJobLat = DEFAULT_GEO_LAT
		/*
			TODO: SEND LOCATION REQUEST
		*/
	} else {
		device.DESJobLng = start.HDR.HdrGeoLng
		device.DESJobLat = start.HDR.HdrGeoLat
	}

	fmt.Printf("\n(device *Device) StartJob() Check Well Name -> %s\n", start.HDR.HdrWellName)
	if start.HDR.HdrWellName == "" || start.HDR.HdrWellName == device.CmdArchiveName() {
		start.HDR.HdrWellName = start.STA.StaJobName
	}

	fmt.Printf("\n(device *Device) StartJob() -> CREATE A JOB RECORD IN THE DES DATABASE\n%v\n", device.DESJob)

	/* CREATE A JOB RECORD IN THE DES DATABASE */
	if err := pkg.WriteDESJob(&device.DESJob); err != nil {
		pkg.LogErr(err)
	}

	/* CREATE DESJobSearch RECORD */
	start.HDR.Create_DESJobSearch(device.DESRegistration)

	/* WE AVOID CREATING IF THE DATABASE WAS PRE-EXISTING, LOG TO CMDARCHIVE  */
	if pkg.ADB.CheckDatabaseExists(device.DESJobName) {
		device.JobDBC = device.CmdDBC
		fmt.Printf("\n(device *Device) StartJob( ): DATABASE ALREADY EXISTS! *** LOGGING TO: %s\n", device.JobDBC.GetDBName())
	} else {
		/* CREATE NEW JOB DATABASE */
		pkg.ADB.CreateDatabase(device.DESJobName)

		/* CONNECT TO THE NEW ACTIVE JOB DATABASE, ON FAILURE, LOG TO CMDARCHIVE */
		if err := device.ConnectJobDBC(); err != nil {
			device.JobDBC = device.CmdDBC
			fmt.Printf("\n(device *Device) StartJob( ): CONNECTION FAILED! *** LOGGING TO: %s\n", device.JobDBC.GetDBName())

		} else {
			fmt.Printf("\n(device *Device) StartJob( ): CONNECTED TO: %s\n", device.JobDBC.GetDBName())

			/* CREATE JOB DB TABLES */
			if err := device.JobDBC.Migrator().CreateTable(
				&Admin{},
				&State{},
				&Header{},
				&Config{},
				&Sample{},
				&EventTyp{},
				&Event{},
				&Report{},
				&RepSection{},
				&SecDataset{},
				&SecAnnotation{},
			); err != nil {
				pkg.LogErr(err)
			}

			for _, typ := range EVENT_TYPES {
				WriteETYP(typ, &device.JobDBC)
			}
		}
	}

	/* WRITE INITIAL JOB RECORDS */
	if err := WriteADM(start.ADM, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}
	if err := WriteSTA(start.STA, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}
	if err := WriteHDR(start.HDR, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}
	if err := WriteCFG(start.CFG, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}
	if err := WriteEVT(start.EVT, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}

	/* WAIT FOR PENDING MQTT MESSAGES TO COMPLETE */
	device.DESMQTTClient.WG.Wait()

	/* UPDATE THE DEVICE STATE, ENABLING MQTT MESSAGE WRITES TO ACTIVE JOB DB
	AFTER WE HAVE WRITTEN THE INITIAL JOB RECORDS
	*/
	device.ADM = start.ADM
	device.HDR = start.HDR
	device.CFG = start.CFG
	device.EVT = start.EVT
	device.STA = start.STA

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, *device)

	pkg.LogChk(fmt.Sprintf("COMPLETE: %s\n", device.JobDBC.GetDBName()))
}

/* CALLED WHEN THE DEVICE MQTT CLIENT REVIEVES A 'JOB STARTED' EVENT FROM THE DEVICE */
func (device *Device) StartJob(sta State) {
	fmt.Printf("\n(device *Device) StartJob() -> sta:\n%v\n", sta)

	// /* WAIT FOR PENDING MQTT MESSAGE TO COMPLETE */
	// device.DESMQTTClient.WG.Wait()

	/* TODO: CHECK device.STA.StaLogging
	IF WE ARE ALREADY LOGGING, THE ACTIVE JOB MUST BE ENDED BEFORE A NEW ONE IST STARTED
	*/

	/* CLEAR THE ACTIVE JOB DATABASE CONNECTION */
	device.JobDBC.Disconnect()
	hdr := device.HDR

	device.DESJobRegTime = sta.StaTime
	device.DESJobRegAddr = sta.StaAddr
	device.DESJobRegUserID = sta.StaUserID
	device.DESJobRegApp = sta.StaApp

	device.DESJobName = sta.StaJobName
	device.DESJobStart = sta.StaTime
	device.DESJobEnd = 0
	device.DESJobDevID = device.DESDevID

	/* GET LOCATION DATA */
	if hdr.HdrTime == sta.StaTime {
		device.DESJobLng = device.HDR.HdrGeoLng
		device.DESJobLat = device.HDR.HdrGeoLat
	} else {
		/* Header WAS NOT RECEIVED */
		fmt.Printf("\n(device *Device) StartJob() -> Header WAS NOT RECEIVED\n")
		device.DESJobLng = DEFAULT_GEO_LNG
		device.DESJobLat = DEFAULT_GEO_LAT
		/*
			TODO: SEND LOCATION REQUEST
		*/
	}

	fmt.Printf("\n(device *Device) StartJob() -> CREATE A JOB RECORD IN THE DES DATABASE\n%v\n", device.DESJob)

	/* CREATE A JOB RECORD IN THE DES DATABASE */
	if err := pkg.WriteDESJob(&device.DESJob); err != nil {
		pkg.LogErr(err)
	}

	/* CREATE DESJobSearch RECORD */
	hdr.Create_DESJobSearch(device.DESRegistration)

	/* WE AVOID CREATING IF THE DATABASE WAS PRE-EXISTING, LOG TO CMDARCHIVE  */
	if pkg.ADB.CheckDatabaseExists(device.DESJobName) {
		device.JobDBC = device.CmdDBC
		fmt.Printf("\n(device *Device) StartJob( ): DATABASE ALREADY EXISTS! *** LOGGING TO: %s\n", device.JobDBC.GetDBName())
	} else {
		/* CREATE NEW JOB DATABASE */
		pkg.ADB.CreateDatabase(device.DESJobName)

		/* CONNECT TO THE NEW ACTIVE JOB DATABASE, ON FAILURE, LOG TO CMDARCHIVE */
		if err := device.ConnectJobDBC(); err != nil {
			device.JobDBC = device.CmdDBC
			fmt.Printf("\n(device *Device) StartJob( ): CONNECTION FAILED! *** LOGGING TO: %s\n", device.JobDBC.GetDBName())

		} else {
			fmt.Printf("\n(device *Device) StartJob( ): CONNECTED TO: %s\n", device.JobDBC.GetDBName())

			/* CREATE JOB DB TABLES */
			if err := device.JobDBC.Migrator().CreateTable(
				&Admin{},
				&State{},
				&Header{},
				&Config{},
				&Sample{},
				&EventTyp{},
				&Event{},
				&Report{},
				&RepSection{},
				&SecDataset{},
				&SecAnnotation{},
			); err != nil {
				pkg.LogErr(err)
			}

			for _, typ := range EVENT_TYPES {
				WriteETYP(typ, &device.JobDBC)
			}
		}
	}

	/* WRITE INITIAL JOB RECORDS */
	if err := WriteADM(device.ADM, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}
	if err := WriteSTA(sta, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}
	if err := WriteHDR(hdr, &device.JobDBC); err != nil {
		// if err := WriteHDR(device.HDR, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}
	if err := WriteCFG(device.CFG, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}
	if err := WriteEVT(device.EVT, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}

	/* WAIT FOR PENDING MQTT MESSAGES TO COMPLETE */
	device.DESMQTTClient.WG.Wait()

	/* UPDATE THE DEVICE STATE, ENABLING MQTT MESSAGE WRITES TO ACTIVE JOB DB
	AFTER WE HAVE WRITTEN THE INITIAL JOB RECORDS
	*/
	device.STA = sta

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, *device)

	pkg.LogChk(fmt.Sprintf("COMPLETE: %s\n", device.JobDBC.GetDBName()))
	// fmt.Printf("\n(device *Device) StartJob( ): COMPLETE: %s\n", device.JobDBC.GetDBName())
}

/* CALLED WHEN DES RECEIVES SAMPLES WITH AN UNKNOWN JOB NAME ( DATABASE DOES NOT EXIST ) */
func (device *Device) RegisterJob() {

	/* ENSURE PREVIOUS JOB HAS ENDED
	MAKE EVT -> JOB ENDED
		- SOURCE DES ( TIME, ADDR, DES UID, APP  )
		- MSG JOB ENDED OFFLINE BY DEVICE
	*/

	/* MAKE EVT -> REGISTER JOB REQ
	- LOG TO CMDARCHIVE
	- PUBLISH TO DEVICE
	- DEVICE WILL SEND ADM, HDR, CFG, EVT( REGISTER JOB )
	- AND FINALLY STA WHICH WILL RESULT IN A CALL TO StartJob( )
	*/

}

/* END JOB ************************************************************************************************/

/* PREPARE, LOG, AND SEND AN END JOB REQUEST */
func (device *Device) EndJobRequestX(src string) (err error) {

	endTime := time.Now().UTC().UnixMilli()

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.STA.StaTime = endTime
	device.STA.StaAddr = src
	device.STA.StaUserID = device.DESJobRegUserID
	device.STA.StaApp = device.DESJobRegApp
	device.STA.StaSerial = device.DESDevSerial
	device.STA.StaVersion = DEVICE_VERSION
	device.STA.StaClass = DEVICE_CLASS
	device.STA.StaLogging = OP_CODE_JOB_END_REQ // This means there is a pending request for the device to start a new job
	device.STA.StaJobName = device.CmdArchiveName()
	device.STA.Validate()
	// pkg.Json("HandleStartJob(): -> device.STA", device.STA)
	device.GetMappedHDR()
	device.GetMappedCFG()
	device.GetMappedSMP()
	device.GetMappedClients()

	device.EVT = Event{
		EvtTime:   endTime,
		EvtAddr:   src,
		EvtUserID: device.DESJobRegUserID,
		EvtApp:    device.DESJobRegApp,
		EvtCode:   OP_CODE_JOB_END_REQ,
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
	device.MQTTPublication_DeviceClient_CMDEndJob(device.EVT)

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, *device)
	return err
}

/* PREPARE, LOG, AND SEND AN END JOB REQUEST */
func (device *Device) EndJobRequest(src string) (err error) {

	endTime := time.Now().UTC().UnixMilli()

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.STA.StaTime = endTime
	device.STA.StaAddr = src
	device.STA.StaUserID = device.DESJobRegUserID
	device.STA.StaApp = device.DESJobRegApp
	device.STA.StaSerial = device.DESDevSerial
	device.STA.StaVersion = DEVICE_VERSION
	device.STA.StaClass = DEVICE_CLASS
	device.STA.StaLogging = OP_CODE_JOB_END_REQ // This means there is a pending request for the device to start a new job
	device.STA.StaJobName = device.CmdArchiveName()
	device.STA.Validate()
	// pkg.Json("HandleStartJob(): -> device.STA", device.STA)
	device.GetMappedHDR()
	device.GetMappedCFG()
	device.GetMappedSMP()
	device.GetMappedClients()

	device.EVT = Event{
		EvtTime:   endTime,
		EvtAddr:   src,
		EvtUserID: device.DESJobRegUserID,
		EvtApp:    device.DESJobRegApp,
		EvtCode:   OP_CODE_JOB_END_REQ,
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
	return err
}

/* CALLED WHEN THE DEVICE MQTT CLIENT REVIEVES A 'JOB ENDED' EVENT FROM THE DEVICE */
func (device *Device) EndJob(sta State) {

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

	/* WRITE END JOB STATE AS RECEIVED TO JOB */
	WriteSTA(sta, &device.JobDBC)

	/* UPDATE THE DEVICE EVENT CODE, DISABLING MQTT MESSAGE WRITES TO ACTIVE JOB DB	*/
	device.STA = sta

	/* CLEAR THE ACTIVE JOB DATABASE CONNECTION */
	device.JobDBC.Disconnect()

	jobName := device.DESJobName
	/* CLOSE DES JOB */
	device.DESJobRegTime = sta.StaTime
	device.DESJobRegAddr = sta.StaAddr
	device.DESJobRegUserID = sta.StaUserID
	device.DESJobRegApp = sta.StaApp
	device.DESJobEnd = sta.StaTime
	fmt.Printf("\n(device *Device) EndJob( ) ENDING: %s\nDESJobID: %d\n", jobName, device.DESJobID)
	pkg.DES.DB.Save(device.DESJob)
	fmt.Printf("\n(device *Device) EndJob( ) %s ENDED\n", jobName)

	device.Update_DESJobSearch(device.DESRegistration)

	/* UPDATE DES CMDARCHIVE */
	cmd := device.GetCmdArchiveDESRegistration()
	cmd.DESJobRegTime = time.Now().UTC().UnixMilli() // WE WANT THIS TO BE THE LATEST
	cmd.DESJobRegAddr = sta.StaAddr
	cmd.DESJobRegUserID = sta.StaUserID
	cmd.DESJobRegApp = sta.StaApp
	cmd.DESJob.DESJobEnd = 0 // ENSURE THE DEVICE IS DISCOVERABLE
	fmt.Printf("\n(device *Device) EndJob( ) UPDATING CMDARCHIVE\ncmd.DESJobID: %d\v", cmd.DESJobID)
	pkg.DES.DB.Save(cmd.DESJob)
	fmt.Printf("\n(device *Device) EndJob( ) CMDARCHIVE UPDATED\n")

	/* ENSURE WE CATCH STRAY SAMPLES IN THE CMDARCHIVE */
	device.DESJob = cmd.DESJob
	device.ConnectJobDBC()

	// /* RETURN DEVICE CLIENT DATA TO DEFAULT STATE */
	// device.ADM.DefaultSettings_Admin(cmd.DESRegistration)
	// device.HDR.DefaultSettings_Header(cmd.DESRegistration)
	// device.CFG.DefaultSettings_Config(cmd.DESRegistration)

	// /* RETURN DEVICE (PHYSICAL) DATA TO DEFAULT STATE */
	// device.MQTTPublication_DeviceClient_CMDAdmin(device.ADM)
	// device.MQTTPublication_DeviceClient_CMDHeader(device.HDR)
	// device.MQTTPublication_DeviceClient_CMDConfig(device.CFG)
	device.SMP = Sample{SmpTime: cmd.DESJobRegTime, SmpJobName: cmd.DESJobName}
	// pkg.Json("(device *Device) EndJob(): -> Devices[device.DESDevSerial] AFTER UPDATE", device)

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, *device)

	fmt.Printf("\n(device *Device) EndJob( ) COMPLETE: %s\n", jobName)
}

/* HEADER - UPDATE DESJobSearch */
func (device *Device) Update_DESJobSearch(reg pkg.DESRegistration) {
	s := pkg.DESJobSearch{}

	res := pkg.DES.DB.Where("des_job_key = ?", reg.DESJobID).First(&s)
	if res.Error != nil {
		pkg.LogErr(res.Error)
	}

	s.DESJobJson = pkg.ModelToJSONString(device)
	if res := pkg.DES.DB.Save(&s); res.Error != nil {
		pkg.LogErr(res.Error)
	}
}

/* SET / GET JOB PARAMS *********************************************************************************/

/* PREPARE, LOG, AND SEND A SET ADMIN REQUEST TO THE DEVICE */
func (device *Device) SetAdminRequest(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
	device.ADM.AdmTime = time.Now().UTC().UnixMilli()
	device.ADM.AdmAddr = src
	device.ADM.Validate() // fmt.Printf("\nHandleSetAdmin( ) -> ADM Validated")
	device.GetMappedSTA() // fmt.Printf("\nHandleSetAdmin( ) -> Mapped STA gotten")
	device.GetMappedHDR() // fmt.Printf("\nHandleSetAdmin( ) -> Mapped HDR gotten")
	device.GetMappedCFG() // fmt.Printf("\nHandleSetAdmin( ) -> Mapped CFG gotten")
	device.GetMappedEVT() // fmt.Printf("\nHandleSetAdmin( ) -> Mapped EVT gotten")
	device.GetMappedSMP() // fmt.Printf("\nHandleSetAdmin( ) -> Mapped SMP gotten")
	device.GetMappedClients() // fmt.Printf("\nHandleSetAdmin( ) -> Mapped Clients gotten")

	/* LOG ADM CHANGE REQUEST TO  CMDARCHIVE */
	adm := device.ADM
	WriteADM(adm, &device.CmdDBC)

	/* CHECK TO SEE IF WE SHOULD LOG TO ACTIVE JOB */
	if device.DESJobName != device.CmdArchiveName() {
		WriteADM(adm, &device.JobDBC)
	}

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
	device.GetMappedSTA()
	device.GetMappedHDR()
	device.GetMappedCFG()
	device.GetMappedEVT()
	device.GetMappedSMP()
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

/*
USED TO SET THE STATE VALUES FOR A GIVEN DEVICE
***NOTE***

	THE STATE IS A READ ONLY STRUCTURE AT THIS TIME
	FUTURE VERSIONS WILL ALLOW DEVICE ADMINISTRATORS TO ALTER SOME STATE VALUES REMOTELY
	CURRENTLY THIS HANDLER IS USED ONLY TO REQUEST THE CURRENT DEVICE STATE
*/
func (device *Device) SetStateRequest(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.GetMappedHDR()
	device.STA.StaTime = time.Now().UTC().UnixMilli()
	device.STA.StaAddr = src
	device.STA.Validate()
	device.GetMappedCFG()
	device.GetMappedEVT()
	device.GetMappedSMP()
	device.GetMappedClients()

	/* LOG STA CHANGE REQUEST TO CMDARCHIVE */
	sta := device.STA
	WriteSTA(sta, &device.CmdDBC)

	/* CHECK TO SEE IF WE SHOULD LOG TO ACTIVE JOB */
	if device.DESJobName != device.CmdArchiveName() {
		WriteSTA(sta, &device.JobDBC)
	}

	/* MQTT PUB CMD: STATE */
	fmt.Printf("\nSetStateRequest( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDState(sta)

	return
}

/* PREPARE, LOG, AND SEND A SET HEADER REQUEST TO THE DEVICE */
func (device *Device) SetHeaderRequest(src string) (err error) {

	// hdr := device.HDR
	// hdr.HdrTime = time.Now().UTC().UnixMilli()
	// hdr.HdrAddr = src
	// hdr.Validate()
	// d := ReadDevicesMap(device.DESDevSerial)
	// d.HDR = hdr
	// device = &d

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.HDR.HdrTime = time.Now().UTC().UnixMilli()
	device.HDR.HdrAddr = src
	device.HDR.Validate()
	device.GetMappedSTA()
	device.GetMappedCFG()
	device.GetMappedEVT()
	device.GetMappedSMP()
	device.GetMappedClients()

	/* LOG HDR CHANGE REQUEST TO CMDARCHIVE */
	hdr := device.HDR
	WriteHDR(hdr, &device.CmdDBC)

	/* CHECK TO SEE IF WE SHOULD LOG TO ACTIVE JOB */
	if device.DESJobName != device.CmdArchiveName() {
		WriteHDR(hdr, &device.JobDBC)
	}

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
	device.GetMappedSTA()
	device.GetMappedHDR()
	device.CFG.CfgTime = time.Now().UTC().UnixMilli()
	device.CFG.CfgAddr = src
	device.CFG.Validate()
	device.GetMappedEVT()
	device.GetMappedSMP()
	device.GetMappedClients()

	/* LOG CFG CHANGE REQUEST TO CMDARCHIVE */
	cfg := device.CFG
	WriteCFG(cfg, &device.CmdDBC)

	/* CHECK TO SEE IF WE SHOULD LOG TO ACTIVE JOB */
	if device.DESJobName != device.CmdArchiveName() {
		WriteCFG(cfg, &device.JobDBC)
	}

	/* MQTT PUB CMD: CFG */
	fmt.Printf("\nHandleSetConfig( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDConfig(cfg)

	/* UPDATE DevicesMap */
	UpdateDevicesMap(device.DESDevSerial, *device)

	return
}

/* REQUEST THE CURRENT CONFIG FROM THE DEVICE */

/* PREPARE, LOG, AND SEND A SET EVENT REQUEST TO THE DEVICE */
func (device *Device) CreateEventRequest(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.GetMappedSTA() // fmt.Printf("\nCreateEventRequest( ) -> Mapped STA gotten")
	device.GetMappedHDR() // fmt.Printf("\nCreateEventRequest( ) -> Mapped HDR gotten")
	device.GetMappedCFG() // fmt.Printf("\nCreateEventRequest( ) -> Mapped CFG gotten")
	device.GetMappedSMP() // fmt.Printf("\nCreateEventRequest( ) -> Mapped SMP gotten")
	device.GetMappedClients() // fmt.Printf("\nCreateEventRequest( ) -> Mapped Clients gotten")

	/* LOG EVT CHANGE REQUEST TO  CMDARCHIVE */
	device.EVT.EvtTime = time.Now().UTC().UnixMilli()
	device.EVT.EvtAddr = src
	device.EVT.Validate() // fmt.Printf("\nCreateEventRequest( ) -> EVT Validated")
	evt := device.EVT
	WriteEVT(evt, &device.CmdDBC)

	/* CHECK TO SEE IF WE SHOULD LOG TO ACTIVE JOB */
	if device.DESJobName != device.CmdArchiveName() {
		WriteEVT(evt, &device.JobDBC)
	}

	/* MQTT PUB CMD: EVT */
	fmt.Printf("\nHandleSetEvent( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDEvent(evt)

	/* UPDATE DevicesMap */
	UpdateDevicesMap(device.DESDevSerial, *device)

	return
}

/*
UPDATE THE MAPPED DES DEVICE WITH NEW Debug SETTINGS
***NOTE***

	DEBUG SETINGS ARE NOT LOGGED TO ANY DATABASE
	NOR ARE THEY TRANSMITTED TO THE PHYSICAL DEVICE
*/
func (device *Device) SetDebug() (err error) {

	device.UpdateMappedDBG(true)
	/* TODO: ERROR CHECKING */
	return
}

func (device *Device) TestMsgLimit() (size int, err error) {

	/* 1468 Byte Kafka*/
	msg := MsgLimit{ 
		Kafka: `One morning, when Gregor Samsa woke from troubled dreams, he found himself transformed in his bed into a horrible vermin. He lay on his armour-like back, and if he lifted his head a little he could see his brown belly, slightly domed and divided by arches into stiff sections. The bedding was hardly able to cover it and seemed ready to slide off any moment. His many legs, pitifully thin compared with the size of the rest of him, waved about helplessly as he looked. "What's happened to me?" he thought. It wasn't a dream. His room, a proper human room although a little too small, lay peacefully between its four familiar walls. A collection of textile samples lay spread out on the table - Samsa was a travelling salesman - and above it there hung a picture that he had recently cut out of an illustrated magazine and housed in a nice, gilded frame. It showed a lady fitted out with a fur hat and fur boa who sat upright, raising a heavy fur muff that covered the whole of her lower arm towards the viewer. Gregor then turned to look out the window at the dull weather. Drops of rain could be heard hitting the pane, which made him feel quite sad. "How about if I sleep a little bit longer and forget all this nonsense", he thought, but that was something he was unable to do because he was used to sleeping on his right, and in his present state couldn't get into that position. However hard he threw himself onto his right, he always rolled back to where he was.`,
	}
	out := pkg.ModelToJSONString(msg)
	size = len(out)
	fmt.Printf("\nTestMsgLimit( ) -> length: %d\n", size)

	// fmt.Printf("\nTestMsgLimit( ) -> getting mapped clients...\n")
	device.GetMappedClients()
	
	/* MQTT PUB CMD: EVT */
	// fmt.Printf("\nTestMsgLimit( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDMsgLimit(msg)

	return
}

/*
	REGISTER A C001V001 DEVICE ON THIS DES

- CREATE DES DB RECORDS
  - A DEVICE RECORD FOR THIS DEVICE IN des_devs
  - A JOB RECORD FOR THIS DEVICE'S CMDARCHIVE IN des_jobs
  - A JOB SEARCH RECORDS FOR THIS DEVICE'S CMDARCHIVE IN des_job_searches

- CREATE A CMDARCHIVE DATABASE FOR THIS DEVICE
  - POPULATE DEFAULT ADM, STA, HDR, CFG, EVT

-  CONNECT DEVICE ( DeviceClient_Connect() )
*/
func (device *Device) RegisterDevice(src string, reg pkg.DESRegistration) (err error) {
	fmt.Printf("\n(device *Device)RegisterDevice( )...\n")

	t := time.Now().UTC().UnixMilli()

	/* CREATE A DES DEVICE RECORD */
	device.DESDev = reg.DESDev
	device.DESDevRegTime = t
	device.DESDevRegAddr = src
	/* TODO: VALIDATE SERIAL # */
	device.DESDevVersion = "001"
	device.DESDevClass = "001"
	if res := pkg.DES.DB.Create(&device.DESDev); res.Error != nil {
		return res.Error
	}

	/* CREATE A DES JOB RECORD ( CMDARCHIVE )*/
	device.DESJobRegTime = t
	device.DESJobRegAddr = src
	device.DESJobRegUserID = device.DESDevRegUserID
	device.DESJobRegApp = device.DESDevRegApp
	device.DESJobName = fmt.Sprintf("%s_CMDARCHIVE", device.DESDevSerial)
	device.DESJobStart = 0
	device.DESJobEnd = 0
	device.DESJobLng = DEFAULT_GEO_LNG
	device.DESJobLat = DEFAULT_GEO_LAT
	device.DESJobDevID = device.DESDevID

	pkg.Json("RegisterDevice( ) -> pkg.DES.DB.Create(&device.DESJob) -> device.DESJob", device.DESJob)
	if res := pkg.DES.DB.Create(&device.DESJob); res.Error != nil {
		return res.Error
	}

	/*  CREATE A CMDARCHIVE DATABASE FOR THIS DEVICE */
	pkg.ADB.CreateDatabase(strings.ToLower(device.DESJobName))

	/*  TEMPORARILY CONNECT TO CMDARCHIVE DATABASE FOR THIS DEVICE */
	if err = device.ConnectJobDBC(); err != nil {
		return err
	}

	/* CREATE JOB DB TABLES */
	if err := device.JobDBC.Migrator().CreateTable(
		&Admin{},
		&State{},
		&Header{},
		&Config{},
		&EventTyp{},
		&Event{},
		&Sample{},
	); err != nil {
		pkg.LogErr(err)
	}

	/* WRITE DEFAULT ADM, STA, HDR, CFG, EVT TO CMDARCHIVE */
	device.ADM.DefaultSettings_Admin(device.DESRegistration)
	device.STA.DefaultSettings_State(device.DESRegistration)
	device.STA.StaLogging = OP_CODE_DES_REGISTERED
	device.HDR.DefaultSettings_Header(device.DESRegistration)
	device.CFG.DefaultSettings_Config(device.DESRegistration)
	device.SMP = Sample{SmpTime: t, SmpJobName: device.DESJobName}
	device.EVT = Event{
		EvtTime:   t,
		EvtAddr:   src,
		EvtUserID: device.DESDevRegUserID,
		EvtApp:    device.DESDevRegApp,
		EvtCode:   OP_CODE_DES_REGISTERED,
		EvtTitle:  "DEVICE REGISTRATION",
		EvtMsg:    "DEVICE REGISTERED",
	}

	for _, typ := range EVENT_TYPES {
		WriteETYP(typ, &device.JobDBC)
	}
	if err := WriteADM(device.ADM, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}
	if err := WriteSTA(device.STA, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}
	if err := WriteHDR(device.HDR, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}
	if err := WriteCFG(device.CFG, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}
	if err := WriteEVT(device.EVT, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}
	if err := WriteSMP(device.SMP, &device.JobDBC); err != nil {
		pkg.LogErr(err)
	}

	/* CLOSE TEMPORARY CONNECTION TO  CMDARCHIVE DB */
	device.JobDBC.Disconnect()

	/* CREATE PERMANENT DES DEVICE CLIENT CONNECTIONS */
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DeviceClient_Connect()

	return
}

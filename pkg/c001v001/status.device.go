package c001v001

import (
	"fmt"
	"sync"

	"github.com/leehayford/des/pkg"
)

const DEVICE_CLASS = "001"
const DEVICE_VERSION = "001"

const DEFAULT_GEO_LNG = -180 // TODO: TEST -999.25
const DEFAULT_GEO_LAT = 90   // TODO: TEST -999.25

/* OPERATION CODES ( Event.EvtCode 0 : 999 ) *******************************************************/
const OP_CODE_DES_REG_REQ int32 = 0       // USER REQUEST -> CHANGE DEVICE'S OPERATIONAL DATA EXCHANGE SERVER
const OP_CODE_DES_REGISTERED int32 = 1    // DEVICE RESPONSE -> SENT TO NEW DATA EXCHANGE SERVER
const OP_CODE_JOB_ENDED int32 = 2         // DEVICE RESPONSE -> JOB ENDED
const OP_CODE_JOB_START_REQ int32 = 3     // USER REQUEST -> START JOB
const OP_CODE_JOB_STARTED int32 = 4       // DEVICE RESPONSE -> JOB STARTED
const OP_CODE_JOB_END_REQ int32 = 5       // USER REQUEST -> END JOB
const OP_CODE_JOB_OFFLINE_START int32 = 6 // JOB WAS STARTED OFFLINE BY OPERATOR ON SITE
const OP_CODE_JOB_OFFLINE_END int32 = 7   // JOB WAS ENDED OFFLINE BY OPERATOR ON SITE
const OP_CODE_GPS_ACQ int32 = 8  // DEVICE NOTIFICATION -> LTE DISABLED FOR GPS AQUISITION
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

const MIN_SAMPLE_PERIOD int32 = 100

type DevicesMap map[string]Device

var Devices = make(DevicesMap)
var DevicesRWMutex = sync.RWMutex{}

/* GET THE CURRENT DESRegistration FOR ALL DEVICES ON THIS DES */
func GetDeviceList() (devices []pkg.DESRegistration, err error) {

	/* WHERE MORE THAN ONE JOB IS ACTIVE ( des_job_end = 0 ) WE WANT THE LATEST */
	subQryLatestJob := pkg.DES.DB.
		Table("des_jobs").Select("des_job_dev_id, MAX(des_job_reg_time) AS max_time").
		Where("des_job_end = 0").
		Group("des_job_dev_id")

	qry := pkg.DES.DB.
		Table("des_devs").Select("des_devs.*, des_jobs.*").
		Joins("JOIN des_jobs ON des_jobs.des_job_dev_id = des_devs.des_dev_id").
		Joins(`JOIN ( ? ) j ON des_jobs.des_job_dev_id = j.des_job_dev_id AND des_jobs.des_job_reg_time = j.max_time`, subQryLatestJob).
		Order("j.max_time DESC")

	res := qry.Scan(&devices)
	// pkg.Json("GetDeviceList(): DESRegistrations", res)
	err = res.Error
	return
}

/* GET THE MAPPED DATA FOR ALL DEVICES IN THE LIST OF DESRegistrations */
func GetDevices(regs []pkg.DESRegistration) (devices []Device) {
	for _, reg := range regs {
		// pkg.Json("GetDevices( ) -> reg", reg)
		device := DevicesMapRead(reg.DESDevSerial)
		device.DESRegistration = reg
		devices = append(devices, device)
	}
	// pkg.Json("GetDevices(): Devices", devices)
	return
}

/* CONNECT DB AND MQTT CLIENTS FOR ALL DEVICES; CALLED ON SERVER STARTUP */
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

/* DISCONNECT DB AND MQTT CLIENTS FOR ALL DEVICES; CALLED ON SERVER SHUT DOWN */
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

/* WRITE TO THE DevicesMap

WRITE LOCK IS USED TO PREVENT DEVICE MAP READS DURING WRITE OPERATIONS
  - WHERE THE MAP IS ALREADY LOCKED, THIS WRITE OPERATION IS BLOCKED UNTIL THE READ IS COMPLETE
  - ONCE THIS WRITE OPERATION ESTABLISHES A LOCK, ALL READ & WRITE  OPERATIONS ARE BLOCKED UNTIL THIS WRITE IS COMPLETE
*/
func DevicesMapWrite(serial string, d Device) {
	DevicesRWMutex.Lock()
	Devices[serial] = d
	DevicesRWMutex.Unlock()
}

/* READ THE DevicesMap

WRITE LOCK IS USED TO PREVENT MAP READS DURING WRITE OPERATIONS
  - WHERE THE MAP IS ALREADY LOCKED, THIS READ OPERATION IS BLOCKED UNTIL THE WRITE IS COMPLETE
  - ONCE THIS READ OPERATION ESTABLISHES A LOCK, ALL READ & WRITE OPERATIONS ARE BLOCKED UNTIL THIS READ IS COMPLETE
*/
func DevicesMapRead(serial string) (device Device) {
	DevicesRWMutex.Lock()
	device = Devices[serial]
	DevicesRWMutex.Unlock()
	return
}

/* REMOVE DEVICE FROM DevicesMap MAP */
func FromDevicesMapRemove(serial string) {
	DevicesRWMutex.Lock()
	delete(Devices, serial)
	DevicesRWMutex.Unlock()
	// fmt.Printf("\n\nFromDevicesMapRemove( %s ) Removed... \n", serial)
}

/* HYDRATES THE DEVICE'S DB & MQTT CLIENT OBJECTS OF THE DEVICE FROM DevicesMap */
func (device *Device) GetMappedClients() {

	/* GET THE DEVICE CLIENT DATA FROM THE DEVICES CLIENT MAP */
	d := DevicesMapRead(device.DESDevSerial) // fmt.Printf("\n%v", d)

	/* WAIT TO PREVENT RACE CONDITION - DON"T READ WHEN DBC IS BUSY */
	if d.CmdDBC.DB != nil {
		d.CmdDBC.WG.Wait()
	}
	if device.CmdDBC.DB != nil {
		device.CmdDBC.WG.Wait()
	}
	device.CmdDBC = d.CmdDBC

	/* WAIT TO PREVENT RACE CONDITION - DON"T READ WHEN DBC IS BUSY */
	if d.JobDBC.DB != nil {
		d.JobDBC.WG.Wait()
	}
	if device.JobDBC.DB != nil {
		device.JobDBC.WG.Wait()
	}
	device.JobDBC = d.JobDBC

	if device.DESMQTTClient.Client == nil {
		device.DESMQTTClient = pkg.DESMQTTClient{}
	}
	device.DESMQTTClient = d.DESMQTTClient
}

/* HYDRATES THE DEVICE'S Admin STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedADM() {
	d := DevicesMapRead(device.DESDevSerial)
	device.ADM = d.ADM
}

/* HYDRATES THE DEVICE'S State STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedSTA() {
	d := DevicesMapRead(device.DESDevSerial)
	device.STA = d.STA
}

/* HYDRATES THE DEVICE'S Header STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedHDR() {
	d := DevicesMapRead(device.DESDevSerial)
	device.HDR = d.HDR
}

/* HYDRATES THE DEVICE'S Config STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedCFG() {
	d := DevicesMapRead(device.DESDevSerial)
	device.CFG = d.CFG
}

/* HYDRATES THE DEVICE'S Event STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedEVT() {
	d := DevicesMapRead(device.DESDevSerial)
	device.EVT = d.EVT
}

/* HYDRATES THE DEVICE'S Sample STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedSMP() {
	d := DevicesMapRead(device.DESDevSerial)
	device.SMP = d.SMP
}

/* HYDRATES THE DEVICE'S pkg.UserResponse STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedDESU() {
	d := DevicesMapRead(device.DESDevSerial)
	device.DESU = d.DESU
}

/* HYDRATES THE DEVICE'S Debug STRUCT FROM THE DevicesMap */
func (device *Device) GetMappedDBG() {
	d := DevicesMapRead(device.DESDevSerial)
	device.DBG = d.DBG
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Admin */
func (device *Device) UpdateMappedADM() {
	d := DevicesMapRead(device.DESDevSerial)
	d.ADM = device.ADM
	DevicesMapWrite(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT State */
func (device *Device) UpdateMappedSTA() {
	d := DevicesMapRead(device.DESDevSerial)
	d.STA = device.STA
	DevicesMapWrite(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Header */
// func (device *Device) UpdateMappedHDR(hdr Header) {
func (device *Device) UpdateMappedHDR() {
	d := DevicesMapRead(device.DESDevSerial)
	// d.HDR = hdr
	d.HDR = device.HDR
	DevicesMapWrite(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Config */
func (device *Device) UpdateMappedCFG() {
	d := DevicesMapRead(device.DESDevSerial)
	d.CFG = device.CFG
	DevicesMapWrite(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Event */
func (device *Device) UpdateMappedEVT() {
	d := DevicesMapRead(device.DESDevSerial)
	d.EVT = device.EVT
	DevicesMapWrite(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Sample */
func (device *Device) UpdateMappedSMP() {
	d := DevicesMapRead(device.DESDevSerial)
	d.SMP = device.SMP
	DevicesMapWrite(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT pkg.UserResponse */
func (device *Device) UpdateMappedDESU() {
	d := DevicesMapRead(device.DESDevSerial)
	d.DESU = device.DESU
	DevicesMapWrite(device.DESDevSerial, d)
}

/* UPDATES THE DevicesMap WITH THE DEVICE'S CURRENT Debug */
func (device *Device) UpdateMappedDBG(sync bool) {
	d := DevicesMapRead(device.DESDevSerial)
	d.DBG = device.DBG
	DevicesMapWrite(device.DESDevSerial, d)
	if sync {
		device = &d
	}
}



/* DES DEVICE CLIENT KEEP ALIVE ********************************************************/
const DES_PING_TIMEOUT = 10000
const DES_PING_LIMIT = DEVICE_PING_TIMEOUT + 1000

var DESDeviceClientPings = make(pkg.PingsMap)
var DESDeviceClientPingsRWMutex = sync.RWMutex{}

/* WRITE TO THE DESDeviceClientPingsMap

WRITE LOCK IS USED TO PREVENT MAP READS DURING WRITE OPERATIONS
  - WHERE THE MAP IS ALREADY LOCKED, THIS WRITE OPERATION IS BLOCKED UNTIL THE READ IS COMPLETE
  - ONCE THIS WRITE OPERATION ESTABLISHES A LOCK, ALL READ & WRITE  OPERATIONS ARE BLOCKED UNTIL THIS WRITE IS COMPLETE
*/
func DESDeviceClientPingsMapWrite(serial string, ping pkg.Ping) {
	DESDeviceClientPingsRWMutex.Lock()
	DESDeviceClientPings[serial] = ping
	DESDeviceClientPingsRWMutex.Unlock()
}

/* READ FROM THE DESDeviceClientPingsMap; RETURS pkg.Ping

WRITE LOCK IS USED TO PREVENT MAP READS DURING WRITE OPERATIONS
  - WHERE THE MAP IS ALREADY LOCKED, THIS READ OPERATION IS BLOCKED UNTIL THE WRITE IS COMPLETE
  - ONCE THIS READ OPERATION ESTABLISHES A LOCK, ALL READ & WRITE OPERATIONS ARE BLOCKED UNTIL THIS READ IS COMPLETE
*/
func DESDeviceClientPingsMapRead(serial string) (ping pkg.Ping) {
	DESDeviceClientPingsRWMutex.Lock()
	ping = DESDeviceClientPings[serial]
	DESDeviceClientPingsRWMutex.Unlock()
	return
}

/* REMOVE DEVICE FROM DESDeviceClientPings MAP */
func DESDeviceClientPingsMapRemove(serial string) {
	DESDeviceClientPingsRWMutex.Lock()
	delete(DESDeviceClientPings, serial)
	DESDeviceClientPingsRWMutex.Unlock()
	// fmt.Printf("\n\nDESDeviceClientPingsMapRemove( %s ) Removed... \n", serial)
}

/* UPDATE DESDeviceClientPingsMap, AND Publish DESPING */
func (device *Device) UpdateDESDeviceClientPing(ping pkg.Ping) {

	/* UPDATE DESDeviceClientPings MAP */
	DESDeviceClientPingsMapWrite(device.DESDevSerial, ping)

	/* CALL IN GO ROUTINE  *** DES TOPIC *** - ALERT USER CLIENTS */
	go device.MQTTPublication_DeviceClient_DESDeviceClientPing(ping)
}



/* PHYSICAL DEVICE KEEP ALIVE ********************************************************/
const DEVICE_PING_TIMEOUT = 30000
const DEVICE_PING_LIMIT = DEVICE_PING_TIMEOUT + 1000

var DevicePings = make(pkg.PingsMap)
var DevicePingsRWMutex = sync.RWMutex{}

/* WRITE TO THE DevicePingsMap

WRITE LOCK IS USED TO PREVENT MAP READS DURING WRITE OPERATIONS
  - WHERE THE MAP IS ALREADY LOCKED, THIS WRITE OPERATION IS BLOCKED UNTIL THE READ IS COMPLETE
  - ONCE THIS WRITE OPERATION ESTABLISHES A LOCK, ALL READ & WRITE  OPERATIONS ARE BLOCKED UNTIL THIS WRITE IS COMPLETE
*/
func DevicePingsMapWrite(serial string, ping pkg.Ping) {
	DevicePingsRWMutex.Lock()
	DevicePings[serial] = ping
	DevicePingsRWMutex.Unlock()
}

/* READ FROM THE DevicePingsMap; RETURS pkg.Ping

WRITE LOCK IS USED TO PREVENT MAP READS DURING WRITE OPERATIONS
  - WHERE THE MAP IS ALREADY LOCKED, THIS READ OPERATION IS BLOCKED UNTIL THE WRITE IS COMPLETE
  - ONCE THIS READ OPERATION ESTABLISHES A LOCK, ALL READ & WRITE OPERATIONS ARE BLOCKED UNTIL THIS READ IS COMPLETE
*/
func DevicePingsMapRead(serial string) (ping pkg.Ping) {
	DevicesRWMutex.Lock()
	ping = DevicePings[serial]
	DevicesRWMutex.Unlock()
	return
}

/* REMOVE DEVICE FROM DevicePings MAP */
func DevicePingsMapRemove(serial string) {
	DevicePingsRWMutex.Lock()
	delete(DevicePings, serial)
	DevicePingsRWMutex.Unlock()
	// fmt.Printf("\n\nDevicePingsMapRemove( %s ) Removed... \n", serial)
}

/* QUALIFY RECEIVED PING THEN UPDATE DevicePingsMap, AND Publish PING */
func (device *Device) UpdateDevicePing(ping pkg.Ping) {

	/* TODO : CHECK LATENCEY BETWEEN DEVICE PING TIME AND SERVER TIME
	- IGNORE THE RECEIVED DEVICE TIME FOR NOW,
	- WE DON'T REALLY CARE FOR KEEP-ALIVE PURPOSES
	*/

	if !ping.OK || ping.Time == 0 {
		ping = DevicePingsMapRead(device.DESDevSerial)
		ping.OK = false
		// fmt.Printf("\n%s -> UpdateDevicePing( ) -> Timeout.", device.DESDevSerial )
	}

	/* UPDATE device.PING AND DevicePings MAP */
	DevicePingsMapWrite(device.DESDevSerial, ping)

	/* CALL IN GO ROUTINE  *** DES TOPIC *** - ALERT USER CLIENTS */
	go device.MQTTPublication_DeviceClient_DESDevicePing(ping)
}

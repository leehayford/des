package c001v001

import (
	"fmt"
	"strings"

	"time"

	"github.com/leehayford/des/pkg"
)

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
	pkg.DESRegistration `json:"reg"`     // Contains registration data for both the device and active job
	ADM                 Admin            `json:"adm"` // Last known Admin value
	STA                 State            `json:"sta"` // Last known State value
	HDR                 Header           `json:"hdr"` // Last known Header value
	CFG                 Config           `json:"cfg"` // Last known Config value
	EVT                 Event            `json:"evt"` // Last known Event value
	SMP                 Sample           `json:"smp"` // Last known Sample value
	DBG                 Debug            `json:"dbg"` // Settings used while debugging
	DESPingStop         chan struct{}    `json:"-"`   // Send DESPingStop when DeviceClients are disconnected
	CmdDBC              pkg.DBClient     `json:"-"`   // Database Client for the CMDARCHIVE
	JobDBC              pkg.DBClient     `json:"-"`   // Database Client for the active job
	pkg.DESMQTTClient   `json:"-"`       // MQTT client handling all subscriptions and publications for this device
	DESU                pkg.UserResponse `json:"-"` // User Account of this Device. Appears in Device / DES generated records / messages
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
	device.DESJobName = device.CmdArchiveName()
	device.DESJobStart = 0
	device.DESJobEnd = 0
	device.DESJobLng = DEFAULT_GEO_LNG
	device.DESJobLat = DEFAULT_GEO_LAT
	device.DESJobDevID = device.DESDevID

	pkg.Json("RegisterDevice( ) -> pkg.DES.DB.Create(&device.DESJob) -> device.DESJob", device.DESJob)
	if res := pkg.DES.DB.Create(&device.DESJob); res.Error != nil {
		return res.Error
	}

	/* CREATE A DES USER ACCOUNT FOR THIS DEVICE */
	_, err = pkg.CreateDESUserForDevice(device.DESDevSerial, device.CmdArchiveName())
	if err != nil {
		return err
	}

	/*  CREATE A CMDARCHIVE DATABASE FOR THIS DEVICE */
	pkg.ADB.CreateDatabase(strings.ToLower(device.CmdArchiveName()))

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
	// device.STA.StaLogging = OP_CODE_DES_REGISTERED
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

	/* CREATE DESJobSearch RECORD FOR CMDARCHIVE */
	device.Create_DESJobSearch(device.DESRegistration)

	/* CREATE PERMANENT DES DEVICE CLIENT CONNECTIONS */
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DeviceClient_Connect()

	return
}

/*  DEVICE CLEINT CONNECTIONS ********************************************************************/

/* GET THIS DEVICE'S REGISTRATION RECORD FROM DES DATABASE */
func (device *Device) GetDeviceDESRegistration(serial string) (err error) {

	res := pkg.DES.DB.
		Order("des_dev_reg_time desc").
		First(&device.DESRegistration.DESDev, "des_dev_serial =?", serial)
	err = res.Error
	return
}

/* CONNECT DEVICE DATABASE AND MQTT CLIENTS ADD CONNECTED DEVICE TO DevicesMap */
func (device *Device) DeviceClient_Connect() (err error) {

	fmt.Printf("\n\n(device *Device) DeviceClient_Connect() -> %s -> connecting... \n", device.DESDevSerial)

	/* DEVICE USER ID IS USED WHEN CREATING AUTOMATED / ALARM Event OR Config STRUCTS
	- WE DON'T WANT TO ATTRIBUTE THEM TO ANOTHER USER */
	device.GetDeviceDESU()

	fmt.Printf("\n(device *Device) DeviceClient_Connect() -> %s -> connecting CMDARCHIVE... \n", device.DESDevSerial)
	if err := device.ConnectCmdDBC(); err != nil {
		return pkg.LogErr(err)
	}

	fmt.Printf("\n(device *Device) DeviceClient_Connect() -> %s -> connecting ACTIVE JOB: %s\n... \n", device.DESDevSerial, device.DESJobName)
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

	/* START DES DEVICE CLIENT PING */
	device.DESPingStop = make(chan struct{})

	/* ADD TO Devices MAP */
	DevicesMapWrite(device.DESDevSerial, *device)

	/* ADD TO DeviceClientPings MAP */
	DESDeviceClientPingsMapWrite(device.DESDevSerial, pkg.Ping{
		Time: time.Now().UTC().UnixMilli(),
		OK:   true,
	})

	/* ADD TO DevicePings MAP */
	DevicePingsMapWrite(device.DESDevSerial, pkg.Ping{})

	live := true
	go func() {
		for live {
			select {

			case <-device.DESPingStop:
				live = false

			default:
				time.Sleep(time.Millisecond * DES_PING_TIMEOUT)
				device.UpdateDESDeviceClientPing(pkg.Ping{
					Time: time.Now().UTC().UnixMilli(),
					OK:   true,
				}) // fmt.Printf("\n(device *Device) DeviceClient_Connect() -> %s -> DES DEVICE CLIENT PING... \n\n", device.DESDevSerial)

			}
		}
		if device.DESPingStop != nil {
			close(device.DESPingStop)
			device.DESPingStop = nil
		}

		delete(DESDeviceClientPings, device.DESDevSerial)
		fmt.Printf("\n(device *Device) DeviceClient_Connect() -> %s -> DES DEVICE CLIENT PING STOPPED. \n\n", device.DESDevSerial)
	}()

	fmt.Printf("\n(device *Device) DeviceClient_Connect() -> %s -> connected... \n\n", device.DESDevSerial)
	return
}

/* DISCONNECT DEVICE DATABASE AND MQTT CLIENTS; REMOVE CONNECTED DEVICE FROM DevicesMap */
func (device *Device) DeviceClient_Disconnect() (err error) {
	/* TODO: TEST WHEN IMPLEMENTING
	- UNREGISTER DEVICE
	- GRACEFUL SHUTDOWN
	*/
	fmt.Printf("\n\n(*Device) DeviceClient_Disconnect() -> %s -> disconnecting... \n", device.DESDevSerial)

	/* KILL DES DEVICE CLIENT PING REMOVE FROM DeviceClientPings MAP */
	if device.DESPingStop != nil {
		device.DESPingStop <- struct{}{}
	}

	fmt.Printf("\n\n(*Device) DeviceClient_Disconnect() -> %s -> unsubscribing MQTT... \n", device.DESDevSerial)
	if err := device.MQTTDeviceClient_Disconnect(); err != nil {
		return pkg.LogErr(err)
	}

	fmt.Printf("\n\n(*Device) DeviceClient_Disconnect() -> %s -> disconnecting CmdDBC... \n", device.DESDevSerial)
	if err := device.CmdDBC.Disconnect(); err != nil {
		return pkg.LogErr(err)
	}

	fmt.Printf("\n\n(*Device) DeviceClient_Disconnect() -> %s -> disconnecting JobDBC... \n", device.DESDevSerial)
	if err := device.JobDBC.Disconnect(); err != nil {
		return pkg.LogErr(err)
	}

	/* REMOVE DEVICE FROM DevicesMap MAP */
	FromDevicesMapRemove(device.DESDevSerial)

	/* REMOVE DEVICE FROM DESDeviceClientPings MAP */
	DESDeviceClientPingsMapRemove(device.DESDevSerial)

	/* REMOVE DEVICE FROM DevicePings MAP */
	DevicePingsMapRemove(device.DESDevSerial)

	fmt.Printf("\n\n(*Device) DeviceClient_Disconnect() -> %s -> COMPLETE\n", device.DESDevSerial)
	return
}

func (device *Device) DeviceClient_RefreshConnections() (err error) {
	fmt.Printf("\n\n(*Device) DeviceClient_RefreshConnections() -> %s ... \n", device.DESDevSerial)

	/* CLOSE ANY EXISTING CONNECTIONS */
	if err = device.DeviceClient_Disconnect(); err != nil {
		return pkg.LogErr(err)
	}

	/* CONNECT THE DES DEVICE CLIENTS */
	if err = device.DeviceClient_Connect(); err != nil {
		return pkg.LogErr(err)
	}

	return
}

/* DEVICE CLIENT COMMAND ARCHIVE *************************************************************/

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
	} // pkg.Json("(device *Device) GetCmdArchiveDESRegistration( )", cmd)
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

/* DEVICE CLIENT ACTIVE JOB ***********************************************************************/

/* CONNECTS THE ACTIVE JOB DBClient TO THE ACTIVE JOB DATABASE */
func (device *Device) ConnectJobDBC() (err error) {
	device.JobDBC = pkg.DBClient{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(device.DESJobName))}
	return device.JobDBC.Connect()
}

/* HYDRATE THE Device.DESU UserResponse FROM DES.DB */
func (device *Device) GetDeviceDESU() (err error) {
	qry := pkg.DES.DB.Table("users").Select("*").Where("name = ?", device.DESDevSerial)

	u := pkg.User{}
	res := qry.Scan(&u)
	if res.Error != nil {
		pkg.LogErr(res.Error)
		err = res.Error
	} // pkg.Json("GetDeviceDESU( ): ", u)
	device.DESU = u.FilterUserRecord()
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

/* START JOB **********************************************************************************************/

/*
	HTTP REQUEST LOGIC - START JOB REQUEST

- PREPARE, LOG, AND SEND: StartJob STRUCT to MQTT .../cmd/start
*/
func (device *Device) StartJobRequest(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedClients()
	device.GetDeviceDESU()
	device.GetMappedDBG()

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
	device.UpdateMappedSTA()
	pkg.Json("HandleStartJob(): -> device.STA", device.STA)

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
	DevicesMapWrite(device.DESDevSerial, *device)

	return
}

/*
	MQTT REPONSE LOGIC - EXPECTED JOB STARTED RESPONSE

- CALLED WHEN THE DEVICE CLIENT RECIEVES A 'JOB STARTED' EVENT FROM THE DEVICE
*/
func (device *Device) StartJob(start StartJob) {
	// pkg.Json("(device *Device) StartJobX(start StartJob): ", start)

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
	if start.HDR.HdrGeoLng < DEFAULT_GEO_LNG {
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

	// fmt.Printf("\n(device *Device) StartJob() -> CREATE A JOB RECORD IN THE DES DATABASE\n%v\n", device.DESJob)

	/* CREATE A JOB RECORD IN THE DES DATABASE */
	if err := pkg.WriteDESJob(&device.DESJob); err != nil {
		pkg.LogErr(err)
	}

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

		}
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
	DevicesMapWrite(device.DESDevSerial, *device)

	/* CREATE DESJobSearch RECORD */
	device.Create_DESJobSearch(device.DESRegistration)

	pkg.LogChk(fmt.Sprintf("COMPLETE: %s\n", device.JobDBC.GetDBName()))
}

/*
	MQTT RESPONSE LOGIC - UNEXPECTED JOB STARTED RESPONSE

- CALLED WHEN A DEVICE HAS STARTED A JOB AND NO REGISTRATION OR DATABASE EXISTS
*/
func (device *Device) OfflineJobStart(smp Sample) {
	fmt.Printf("\n(*Device) OfflineJobStart( )... \n")

	/* AVOID REPEAT CALLS WHILE WE START A JOB */
	sta := device.STA
	sta.StaTime = smp.SmpTime
	sta.StaAddr = device.DESDevSerial
	sta.StaUserID = device.DESU.GetUUIDString()
	sta.StaApp = pkg.DES_APP
	sta.StaJobName = smp.SmpJobName
	sta.StaLogging = OP_CODE_JOB_OFFLINE_START
	device.STA = sta
	device.UpdateMappedSTA()
	fmt.Printf("\n(*Device) OfflineJobStart( ) -> device.UpdateMappedSTA(): OK \n")

	/* CREATE JOB START MODELS USING sta SOURCE VALUES */
	adm := Admin{}
	adm.DefaultSettings_Admin(device.DESRegistration)
	adm.AdmAddr = sta.StaAddr
	adm.AdmUserID = sta.StaUserID
	adm.AdmApp = sta.StaApp

	hdr := Header{}
	hdr.DefaultSettings_Header(device.DESRegistration)
	hdr.HdrAddr = sta.StaAddr
	hdr.HdrUserID = sta.StaUserID
	hdr.HdrApp = sta.StaApp
	hdr.HdrJobStart = sta.StaTime

	cfg := Config{}
	cfg.DefaultSettings_Config(device.DESRegistration)
	cfg.CfgAddr = sta.StaAddr
	cfg.CfgUserID = sta.StaUserID
	cfg.CfgApp = sta.StaApp

	/* CREATE EVT.EvtCode = OP_CODE_JOB_OFFLINE_START  */
	evt := Event{
		EvtTime:   sta.StaTime,
		EvtAddr:   sta.StaAddr,
		EvtUserID: sta.StaUserID,
		EvtApp:    sta.StaApp,

		EvtCode:  sta.StaLogging,
		EvtTitle: GetEventTypeByCode(sta.StaLogging),
		EvtMsg:   sta.StaJobName,
	}
	fmt.Printf("\n(*Device) OfflineJobStart( ) -> xxx.DefaultSettings_Xxxxx: OK \n")

	// /* ENSURE WE ARE CONNECTED TO THE DB AND MQTT CLIENTS */
	// device.GetMappedClients()
	// fmt.Printf("\n(*Device) OfflineJobStart( ) -> device.GetMappedClients(): OK \n")

	/* START A JOB */
	device.StartJob(StartJob{
		ADM: adm,
		STA: sta,
		HDR: hdr,
		CFG: cfg,
		EVT: evt,
	},
	)

	/* LOG smp TO JOB DATABASE */
	go WriteSMP(smp, &device.JobDBC)
	fmt.Printf("\n(*Device) OfflineJobStart( ) -> WriteSMP(): OK \n")

	/* AQUIRE THE LATES ADM, STA, HDR, CFG, EVT FROM THE DEVICE */
	go device.MQTTPublication_DeviceClient_CMDReport()

	fmt.Printf("\n(*Device) OfflineJobStart( ): COMPLETE. \n")
}

/* END JOB ************************************************************************************************/

/*
	HTTP REQUEST LOGIC - END JOB REQUEST

- PREPARE, LOG State AND Event STRUCTS
- SEND Event STRUCT to MQTT .../cmd/end
*/
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
	device.STA.StaLogging = OP_CODE_JOB_END_REQ // This means there is a pending request for the device to end the current job
	device.STA.StaJobName = device.CmdArchiveName()
	device.STA.Validate()
	// pkg.Json("HandleStartJob(): -> device.STA", device.STA)
	device.GetMappedHDR()
	device.GetMappedCFG()
	device.GetMappedSMP()
	device.GetDeviceDESU()
	device.GetMappedDBG()
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
	device.CmdDBC.Create(&device.STA)       /* TODO: USE WriteSTA(device.STA, &device.CmdDBC) ... */
	device.CmdDBC.Create(&device.EVT)       /* TODO: USE WriteEVT(device.EVT, &device.CmdDBC) ... */

	/* LOG END JOB REQUEST TO ACTIVE JOB */ // fmt.Printf("\nHandleEndJob( ) -> Write to %s \n", device.DESJobName)
	device.STA.StaID = 0
	device.JobDBC.Create(&device.STA) /* TODO: USE WriteSTA(device.STA, &device.JobDBC) ... */

	device.EVT.EvtID = 0
	device.JobDBC.Create(&device.EVT) /* TODO: USE WriteEVT(device.EVT, &device.JobDBC) ... */

	/* MQTT PUB CMD: EVT */
	fmt.Printf("\nHandleEndJob( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDEndJob(device.EVT)

	/* UPDATE THE DEVICES CLIENT MAP */
	DevicesMapWrite(device.DESDevSerial, *device)
	return err
}

/*
	MQTT REPONSE LOGIC - EXPECTED JOB ENDED RESPONSE

- CALLED WHEN THE DEVICE MQTT CLIENT REVIEVES A 'JOB ENDED' EVENT FROM THE DEVICE
*/
func (device *Device) EndJob(sta State) {

	/* UPDATE THE DEVICE EVENT CODE, DISABLING MQTT MESSAGE WRITES TO ACTIVE JOB DB	*/
	device.STA = sta

	/* GET THE FINAL JOB RECORDS BEFORE CLEARING THE ACTIVE JOB DATABASE CONNECTION */
	d := device
	device.JobDBC.Last(&d.ADM)
	device.JobDBC.Last(&d.STA)
	device.JobDBC.Last(&d.HDR)
	device.JobDBC.Last(&d.CFG)
	device.JobDBC.Last(&d.EVT)
	d.SMP = Sample{SmpTime: d.STA.StaTime, SmpJobName: d.STA.StaJobName}

	/* CLEAR THE ACTIVE JOB DATABASE CONNECTION */
	device.JobDBC.Disconnect()

	/* UPDATE DESJobSearch RECORD USING RETRIEVED RECORDS
	THESE VALUES ARE USED FOR SEARCH AND DISPLAY OF THE JOB DATA FOR REPORTING
	*/
	d.Update_DESJobSearch(d.DESRegistration)
	// pkg.Json("(device *Device) EndJob( ) ->  d.Update_DESJobSearch(d.DESRegistration): ", d)

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

	/* GET THE LAST JOB RECORDS RECEIVED FROM THE DEVICE
	THESE VALUES SHOULD BE THE DEFAULTS WHICH THE DEVICE HAS LOADED INTO RAM
	THEY WILL APPEAR IN DEVICE SEARCH WHILE THE DEVICE IS WAITING TO START A NEW JOB
	*/
	device.GetMappedADM()
	device.GetMappedSTA()
	device.GetMappedHDR()
	device.GetMappedCFG()
	device.GetMappedEVT()
	device.GetDeviceDESU()
	device.GetMappedDBG()

	device.SMP = Sample{SmpTime: cmd.DESJobRegTime, SmpJobName: cmd.DESJobName}
	// pkg.Json("(device *Device) EndJobX( ) ->  BEFORE Update_DESJobSearch(): ", device)

	/* UPDATE DESJobSearch RECORD USING RETRIEVED CMD ARCHIVE RECORDS */
	device.Update_DESJobSearch(device.DESRegistration)
	// pkg.Json("(device *Device) EndJob( ) ->  device.Update_DESJobSearch(device.DESRegistration): ", device)

	/* UPDATE THE DEVICES CLIENT MAP */
	DevicesMapWrite(device.DESDevSerial, *device)

	fmt.Printf("\n(device *Device) EndJob( ) COMPLETE: %s\n", jobName)
}

/*
	MQTT REPONSE LOGIC - EXPECTED JOB ENDED RESPONSE

- USED WHEN A DEVICE HAS STARTED A JOB OFFLINE AND ANOTHER JOB IS ALREADY ACTIVE
*/
func (device *Device) OfflineJobEnd(smp Sample) {
	fmt.Printf("\n(*Device) OfflineJobEnd( )... \n")

	/* AVOID REPEAT CALLS WHILE WE END THE ACTIVE JOB */
	sta := device.STA
	sta.StaTime = smp.SmpTime
	sta.StaAddr = device.DESDevSerial
	sta.StaUserID = device.DESU.GetUUIDString()
	sta.StaApp = pkg.DES_APP
	// sta.StaJobName = DON'T UPDATE JOB NAME
	sta.StaLogging = OP_CODE_JOB_OFFLINE_END
	device.STA = sta
	device.UpdateMappedSTA()

	/* CREATE EVT.EvtCode = OP_CODE_JOB_OFFLINE_END  */
	evt := Event{
		EvtTime:   sta.StaTime,
		EvtAddr:   sta.StaAddr,
		EvtUserID: sta.StaUserID,
		EvtApp:    sta.StaApp,

		EvtCode:  sta.StaLogging,
		EvtTitle: GetEventTypeByCode(sta.StaLogging),
		EvtMsg:   sta.StaJobName,
	}
	device.EVT = evt
	device.UpdateMappedEVT()

	/* ENSURE WE ARE CONNECTED TO THE DB AND MQTT CLIENTS */
	device.GetMappedClients()

	/* LOG EVT.OFFLINE_JOB_END TO ACTIVE JOB & CMDARCHIVE */
	go WriteEVT(evt, &device.JobDBC)
	go WriteEVT(evt, &device.CmdDBC)

	/* END THE ACTIVE JOB */
	device.EndJob(sta)

	fmt.Printf("\n(*Device) OfflineJobEnd( ): COMPLETE. \n")
}

/* SAMPLE HANDLING ************************************************************************************/

/*
	CALLED WHEN DES RECEIVES SAMPLES, HANDLES:

- UNKNOWN JOB NAME ( DATABASE DOES NOT EXIST )
- OPERATIONAL ALARMS / NOTIFICATIONS ( SSP / SCVF )
*/
func (device *Device) HandleMQTTSample(mqtts MQTT_Sample) (err error, smp Sample) {

	device.GetMappedSTA()
	sta := device.STA
	// fmt.Printf("\n(*Device) HandleMQTTSample( ): -> RegJob: %s, SMPJob: %s \n, OpCode: %d, StaJob %s\n", device.DESJobName, mqtts.DesJobName, sta.StaLogging, sta.StaJobName)

	/* CREATE Sample STRUCT INTO WHICH WE'LL DECODE THE MQTT_Sample  */
	smp = Sample{SmpJobName: mqtts.DesJobName}

	/* DECODE BASE64URL STRING ( DATA ) */
	if err = smp.DecodeMQTTSample(mqtts.Data); err != nil {
		pkg.LogErr(err)
		return
	}

	/* CHECK SAMPLE JOB NAME */
	if smp.SmpJobName == device.CmdArchiveName() {
		/* WRITE TO JOB CMDARCHIVE
		- SOMETHING HAS GONE WRONG WITH THE DEVICE
		- OR WE ARE TESTING THE DEVICE
		*/
		go WriteSMP(smp, &device.CmdDBC)

		/* TODO: TEST ?... DO NOTHING ...?
		case OP_CODE_DES_REG_REQ:
		case OP_CODE_DES_REGISTERED:
		case OP_CODE_JOB_END_REQ:
		case OP_CODE_JOB_OFFLINE_START:
		case OP_CODE_JOB_OFFLINE_END:
		*/

	} else if smp.SmpJobName == device.DESJobName && sta.StaLogging > OP_CODE_JOB_START_REQ {

		/* WE'RE LOGGING; WRITE TO JOB DATABASE */
		go WriteSMP(smp, &device.JobDBC)

		device.CheckSSPCondition(smp)

		device.CheckSCVFCondition(smp)

	} else if sta.StaLogging == OP_CODE_JOB_ENDED {

		/* DEVICE STARTED A JOB WITHOUT OUR KNOWLEDGE - WE'RE NOT CURRENTLY LOGGING */
		device.OfflineJobStart(smp)

	} else if sta.StaLogging == OP_CODE_JOB_STARTED {

		/* DEVICE ENDED AND STARTED JOBS WITHOUT OUR KNOWLEDGE */
		device.OfflineJobEnd(smp)
		device.OfflineJobStart(smp)
	}

	device.SMP = smp

	/* UPDATE THE DevicesMap - DO NOT CALL IN GOROUTINE  */
	device.UpdateMappedSMP()

	// fmt.Printf("\n(*Device) HandleMQTTSample( ): COMPLETE.\n")
	return
}

/* ??? JOB/REPORT ??? USED WHEN A SAMPLE IS RECEIVED, TO CHECK FOR STABILIZED SHUT-IN PRESSURE ( BUILD-MODE )*/
func (device *Device) CheckSSPCondition(smp Sample) {
	/* TODO */
}

/* ??? JOB/REPORT ??? USED WHEN A SAMPLE IS RECEIVED, TO CHECK FOR STABILIZED FLOW ( FLOW-MODE )*/
func (device *Device) CheckSCVFCondition(smp Sample) {
	/* TODO */
}

/* DEVICE SNAPSHOT *************************************************************************************/

/*
	DEVICE - UPDATE DESJobSearch

JSON MARSHALS DEVICE OBJECT AND WRITES TO DES MAIN DB 'des_job_searches.des_job_json'
*/
func (device *Device) Create_DESJobSearch(reg pkg.DESRegistration) {

	s := pkg.DESJobSearch{
		DESJobToken: device.HDR.SearchToken(),
		DESJobJson:  pkg.ModelToJSONString(device),
		DESJobKey:   reg.DESJobID,
	}

	if res := pkg.DES.DB.Create(&s); res.Error != nil {
		pkg.LogErr(res.Error)
	}
}

/*
	DEVICE - UPDATE DESJobSearch

JSON MARSHALS DEVICE OBJECT AND WRITES TO DES MAIN DB 'des_job_searches.des_job_json'
*/
func (device *Device) Update_DESJobSearch(reg pkg.DESRegistration) {
	fmt.Printf("\n Update_DESJobSearch( ): -> %s: reg.DESJobID: %d\n", reg.DESDevSerial, reg.DESJobID)
	s := pkg.DESJobSearch{}

	res := pkg.DES.DB.Where("des_job_key = ?", reg.DESJobID).First(&s)
	if res.Error != nil {
		pkg.LogErr(res.Error)
		return
	} // pkg.Json("Update_DESJobSearch( ): -> s", s)

	s.DESJobToken = device.HDR.SearchToken()
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
	device.GetDeviceDESU()
	device.GetMappedDBG()
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
	DevicesMapWrite(device.DESDevSerial, *device)

	return
}

/*
	***NOTE*** THE STATE IS A READ ONLY STRUCTURE AT THIS TIME

FUTURE VERSIONS MAY ALLOW DEVICE ADMINISTRATORS TO ALTER SOME STATE VALUES REMOTELY
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
	device.GetDeviceDESU()
	device.GetMappedDBG()
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

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.HDR.HdrTime = time.Now().UTC().UnixMilli()
	device.HDR.HdrAddr = src
	device.HDR.Validate()
	device.GetMappedSTA()
	device.GetMappedCFG()
	device.GetMappedEVT()
	device.GetMappedSMP()
	device.GetDeviceDESU()
	device.GetMappedDBG()
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
	DevicesMapWrite(device.DESDevSerial, *device)

	return
}

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
	device.GetDeviceDESU()
	device.GetMappedDBG()
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
	DevicesMapWrite(device.DESDevSerial, *device)

	return
}

/* PREPARE, LOG, AND SEND A SET EVENT REQUEST TO THE DEVICE */
func (device *Device) CreateEventRequest(src string) (err error) {

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.GetMappedSTA() // fmt.Printf("\nCreateEventRequest( ) -> Mapped STA gotten")
	device.GetMappedHDR() // fmt.Printf("\nCreateEventRequest( ) -> Mapped HDR gotten")
	device.GetMappedCFG() // fmt.Printf("\nCreateEventRequest( ) -> Mapped CFG gotten")
	device.GetMappedSMP() // fmt.Printf("\nCreateEventRequest( ) -> Mapped SMP gotten")
	device.GetDeviceDESU()
	device.GetMappedDBG()
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
	DevicesMapWrite(device.DESDevSerial, *device)

	return
}

/********************************************************************************************************/
/* DEVELOPMENT DATA STRUCTURE ***TODO: REMOVE AFTER DEVELOPMENT*** */
type Debug struct {
	MQTTDelay int32 `json:"mqtt_delay"`
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

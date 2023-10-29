package c001v001

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"

	"os"
	"strconv"
	"strings"

	"time"

	phao "github.com/eclipse/paho.mqtt.golang"

	"github.com/leehayford/des/pkg"
)

/*
	NOT FOR PRODUCTION - SIMULATES A C001V001 DEVICE MQTT DEMO DEVICE

PUBLISHES TO ALL SIG TOPICS AS A SINGLE DEVICE AS A SINGLE DEVICE
SUBSCRIBES TO ALL COMMAND TOPICS AS A SINGLE DEVICE
*/

type DemoModeTransition struct {
	VMin    float32       `json:"v_min"`
	VMax    float32       `json:"v_max"`
	VRes    float32       `json:"v_res"`
	TSpanUp time.Duration `json:"t_span_up"`
	TSpanDn time.Duration `json:"t_span_dn"`
}

type DemoDeviceClient struct {
	Device
	// TZero     time.Time
	MTxCh4    DemoModeTransition `json:"mtx_ch4"`
	MTxHiFlow DemoModeTransition `json:"mtx_hi_flow"`
	MTxLoFlow DemoModeTransition `json:"mtx_lo_flow"`
	MTxBuild  DemoModeTransition `json:"mtx_build"`
	pkg.DESMQTTClient
	Stop  chan struct{}
	Rate  chan int32
	Mode  chan int32
	TZero chan time.Time
}

type DemoDeviceClientsMap map[string]DemoDeviceClient

var DemoDeviceClients = make(DemoDeviceClientsMap)

/* GET THE CURRENT DESRegistration FOR ALL DEMO DEVICES ON THIS DES */
func GetDemoDeviceList() (demos []pkg.DESRegistration, err error) {

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
		Where("des_devs.des_dev_serial LIKE 'DEMO%' ").
		Order("des_devs.des_dev_serial DESC")

	res := qry.Scan(&demos)
	err = res.Error
	return
}

/* REGISTER A DEMO DEVICE ON THIS DES */
func MakeDemoC001V001(serial, userID string) pkg.DESRegistration {

	fmt.Printf("\n\nMakeDemoC001V001() -> %s... \n", serial)

	t := time.Now().UTC().UnixMilli()
	/* CREATE DEMO DEVICE */

	des_dev := pkg.DESDev{
		DESDevRegTime:   t,
		DESDevRegAddr:   "DEMO",
		DESDevRegUserID: userID,
		DESDevRegApp:    "DEMO",
		DESDevSerial:    serial,
		DESDevVersion:   "001",
		DESDevClass:     "001",
	}
	pkg.DES.DB.Create(&des_dev)

	des_job := pkg.DESJob{
		DESJobRegTime:   t,
		DESJobRegAddr:   des_dev.DESDevRegAddr,
		DESJobRegUserID: userID,
		DESJobRegApp:    des_dev.DESDevRegApp,

		DESJobName:  fmt.Sprintf("%s_CMDARCHIVE", serial),
		DESJobStart: 0,
		DESJobEnd:   0,
		DESJobLng:   -180, // -114.75 + rand.Float32() * ( -110.15 + 114.75 ),
		DESJobLat:   90,   // 51.85 + rand.Float32() * ( 54.35 - 51.85 ),
		DESJobDevID: des_dev.DESDevID,
	}
	pkg.DES.DB.Create(&des_job)

	reg := pkg.DESRegistration{
		DESDev: des_dev,
		DESJob: des_job,
	}

	adm := Admin{}
	adm.DefaultSettings_Admin(reg)

	hw := HwID{}
	hw.DefaultSettings_HwID(reg)

	hdr := Header{}
	hdr.DefaultSettings_Header(reg)

	cfg := Config{}
	cfg.DefaultSettings_Config(reg)

	demo := DemoDeviceClient{
		Device: Device{
			DESRegistration: reg,
			ADM:             adm,
			HW:              hw,
			HDR:             hdr,
			CFG:             cfg,
			EVT: Event{
				EvtTime:   t,
				EvtAddr:   des_dev.DESDevRegAddr,
				EvtUserID: userID,
				EvtApp:    des_dev.DESDevRegApp,

				EvtCode:  STATUS_JOB_ENDED,
				EvtTitle: "DEMO DEVICE: Intitial State",
				EvtMsg:   "Created with STATUS_JOB_ENDED event to ensure this newly registered demo device is ready to start a new demo job.",
			},
			SMP: Sample{SmpTime: t, SmpJobName: des_job.DESJobName},
		},
	}
	/* CREATE CMDARCHIVE DATABASE */
	pkg.ADB.CreateDatabase(strings.ToLower(demo.DESJobName))

	demo.ConnectJobDBC()
	// fmt.Printf("\nMakeDemoC001V001(): CONNECTED TO DATABASE: %s\n", demo.JobDBC.ConnStr)

	/* CREATE JOB DB TABLES */
	if err := demo.JobDBC.Migrator().CreateTable(
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

	/* WRITE INITIAL JOB RECORDS */
	for _, typ := range EVENT_TYPES {
		WriteETYP(typ, &demo.JobDBC)
	}
	if err := WriteADM(demo.ADM, &demo.JobDBC); err != nil {
		pkg.TraceErr(err)
	}
	if err := WriteHW(demo.HW, &demo.JobDBC); err != nil {
		pkg.TraceErr(err)
	}
	if err := WriteHDR(demo.HDR, &demo.JobDBC); err != nil {
		pkg.TraceErr(err)
	}
	if err := WriteCFG(demo.CFG, &demo.JobDBC); err != nil {
		pkg.TraceErr(err)
	}
	if err := WriteEVT(demo.EVT, &demo.JobDBC); err != nil {
		pkg.TraceErr(err)
	}
	if err := WriteSMP(demo.SMP, &demo.JobDBC); err != nil {
		pkg.TraceErr(err)
	}

	demo.JobDBC.Disconnect()

	/* WRITE TO FLASH - CMDARCHIVE */
	fmt.Printf("\nMakeDemoC001V001(): WRITE TO FLASH: %s/\n", demo.DESJobName)
	demo.WriteAdmToFlash(demo.DESJobName, demo.ADM)
	demo.WriteHwIDToFlash(demo.DESJobName, demo.HW)
	demo.WriteHdrToFlash(demo.DESJobName, demo.HDR)
	demo.WriteCfgToFlash(demo.DESJobName, demo.CFG)
	demo.WriteSmpToFlash(demo.DESJobName, demo.SMP)
	demo.WriteEvtToFlash(demo.DESJobName, demo.EVT)

	return reg
}

/* RETURNS A 10 CHRACTER SERIAL # LIKE 'DEMO000000' */
func DemoSNMaker(i int) (sn string) {
	iStr := fmt.Sprintf("%d", i)
	l := len(iStr)
	size := 6 - l
	sn0s := string(bytes.Repeat([]byte{0x30}, size))
	return fmt.Sprintf("DEMO%s%s", sn0s, iStr)
}

/* CALLED ON SERVER STARTUP */
func DemoDeviceClient_ConnectAll(qty int) {

	regs, err := GetDemoDeviceList()
	if err != nil {
		pkg.TraceErr(err)
	}

	/* WHERE THERE ARE NO DEMO DEVICES, MAKE qty OF THEM */
	if len(regs) == 0 {
		user := pkg.User{}
		pkg.DES.DB.Last(&user)

		for i := 0; i < qty; i++ {
			regs = append(regs, MakeDemoC001V001(DemoSNMaker(i), user.ID.String()))
		}
		MakeDemoC001V001("RENE123456", user.ID.String())
	}

	for _, reg := range regs {
		demo := DemoDeviceClient{}
		demo.Device.DESRegistration = reg
		demo.DESMQTTClient = pkg.DESMQTTClient{}
		demo.DemoDeviceClient_Connect()
	}
}

/* CALLED ON SERVER SHUT DOWN */
func DemoDeviceClient_DisconnectAll() {
	/* TODO: TEST WHEN IMPLEMENTING
	- UNREGISTER DEVICE
	- GRACEFUL SHUTDOWN
	*/
	fmt.Printf("\nDemoDeviceClient_DisconnectAll()\n")
	for _, d := range DemoDeviceClients {
		d.DemoDeviceClient_Disconnect()
	}
}

func (demo *DemoDeviceClient) DemoDeviceClient_Connect() {

	fmt.Printf("\n\n(demo *DemoDeviceClient) DemoDeviceClient_Connect() -> %s -> connecting... \n", demo.DESDevSerial)

	fmt.Printf("\n(demo *DemoDeviceClient) DemoDeviceClient_Connect() -> %s -> getting last known status... \n", demo.DESDevSerial)
	demo.ConnectJobDBC()
	demo.JobDBC.Last(&demo.ADM)
	demo.JobDBC.Last(&demo.HW)
	demo.JobDBC.Last(&demo.HDR)
	demo.JobDBC.Last(&demo.CFG)
	demo.JobDBC.Last(&demo.SMP)
	demo.JobDBC.Last(&demo.EVT)
	demo.JobDBC.Disconnect() // we don't want to maintain this connection

	// demo.EVT.WG = &sync.WaitGroup{}
	demo.Stop = make(chan struct{})
	demo.Rate = make(chan int32)
	demo.Mode = make(chan int32)
	demo.TZero = make(chan time.Time)

	if err := demo.MQTTDemoDeviceClient_Connect(); err != nil {
		pkg.TraceErr(err)
	}

	/* ADD TO DemoDeviceClients MAP */
	DemoDeviceClients[demo.DESDevSerial] = *demo

	/* RUN THE SIMULATION */
	go demo.Demo_Simulation(demo.HDR.HdrJobName, demo.CFG.CfgVlvTgt, demo.CFG.CfgOpSample)
	time.Sleep(time.Second * 1) // WHY?: Just so the console logs show up in the right order when running local dev

	/* ENSURE WE ARE NOT SENDING DATA IF JOB WAS STOPPED BEFORE SERVER RESTART */
	// pkg.Json("(demo *DemoDeviceClient) DemoDeviceClient_Connect( ) -> demo.DESDev", demo.DESDev)
	// pkg.Json("(demo *DemoDeviceClient) DemoDeviceClient_Connect( ) -> demo.HDR", demo.HDR)
	// pkg.Json("(demo *DemoDeviceClient) DemoDeviceClient_Connect( ) -> demo.EVT", demo.EVT)
	if demo.EVT.EvtCode == STATUS_JOB_ENDED || demo.EVT.EvtCode == STATUS_JOB_END_REQ {
		demo.EndDemoJob(demo.EVT)
	}

	fmt.Printf("\n(demo *DemoDeviceClient) DemoDeviceClient_Connect() -> %s -> connected... \n\n", demo.DESDevSerial)
}
func (demo *DemoDeviceClient) DemoDeviceClient_Disconnect() {
	/* TODO: TEST WHEN IMPLEMENTING
	- UNREGISTER DEVICE
	- GRACEFUL SHUTDOWN
	*/
	fmt.Printf("\n\n(demo *DemoDeviceClient) DemoDeviceClient_Disconnect() -> %s -> disconnecting... \n", demo.DESDevSerial)

	if err := demo.MQTTDeviceClient_Disconnect(); err != nil {
		pkg.TraceErr(err)
	}

	close(demo.Stop)
	demo.Stop = nil

	close(demo.Rate)
	demo.Rate = nil

	close(demo.Mode)
	demo.Mode = nil

	close(demo.TZero)
	demo.TZero = nil

	delete(DemoDeviceClients, demo.DESDevSerial)
}

func (demo *DemoDeviceClient) MQTTDemoDeviceClient_Connect() (err error) {

	/* CREATE MQTT CLIENT ID; 23 CHAR MAXIMUM */
	demo.MQTTUser = pkg.MQTT_USER
	demo.MQTTPW = pkg.MQTT_PW
	demo.MQTTClientID = fmt.Sprintf(
		"%s-%s-%s-DEMO",
		demo.DESDevClass,
		demo.DESDevVersion,
		demo.DESDevSerial,
	)

	/* CONNECT TO THE BROKER WITH 'CleanSession = false'
	AUTOMATICALLY RE-SUBSCRIBE ON RECONNECT AFTER */
	if err = demo.DESMQTTClient.DESMQTTClient_Connect(false, true); err != nil {
		return err
	}

	/* SUBSCRIBE TO ALL MQTTSubscriptions */
	demo.MQTTSubscription_DemoDeviceClient_CMDAdmin().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDAdminReport().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDHwID().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDHeader().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDConfig().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDEvent().Sub(demo.DESMQTTClient)

	return err
}
func (demo *DemoDeviceClient) MQTTDemoDeviceClient_Disconnect() (err error) {

	/* UNSUBSCRIBE FROM ALL MQTTSubscriptions */
	demo.MQTTSubscription_DemoDeviceClient_CMDAdmin().UnSub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDAdminReport().UnSub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDHwID().UnSub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDHeader().UnSub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDConfig().UnSub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDEvent().UnSub(demo.DESMQTTClient)

	/* DISCONNECT THE DESMQTTCLient */
	if err = demo.DESMQTTClient_Disconnect(); err != nil {
		pkg.TraceErr(err)
	}

	fmt.Printf("\n(device) MQTTDemoDeviceClient_Dicconnect( ) -> %s -> disconnected.\n", demo.ClientID)
	return
}

/* SUBSCRIPTIONS ****************************************************************************************/

/* SUBSCRIPTION -> ADMINISTRATION -> UPON RECEIPT, LOG & REPLY TO .../sig/admin */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDAdmin() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDAdmin(),
		Handler: func(c phao.Client, msg phao.Message) {

			demo.DESMQTTClient.WG.Add(1)

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			adm := Admin{}
			if err := json.Unmarshal(msg.Payload(), &adm); err != nil {
				pkg.TraceErr(err)
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteAdmToFlash(demo.CmdArchiveName(), adm)
			adm_rec := adm

			/* UPDATE SOURCE ADDRESS ONLY */
			adm.AdmAddr = demo.DESDevSerial

			if demo.EVT.EvtCode > STATUS_JOB_START_REQ {

				/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB */
				demo.WriteAdmToFlash(demo.DESJobName, adm_rec) 

				/* UPDATE TIME ONLY WHEN LOGGING */
				adm.AdmTime = time.Now().UTC().UnixMilli()

				/* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB */
				demo.WriteAdmToFlash(demo.DESJobName, adm) 
			}

			/* WRITE (AS LOADED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteAdmToFlash(demo.CmdArchiveName(), adm)

			/* LOAD VALUE INTO SIM 'RAM' */
			demo.ADM = adm

			/* SEND CONFIRMATION */
			go demo.MQTTPublication_DemoDeviceClient_SIGAdmin(adm)

			demo.DESMQTTClient.WG.Done()
		},
	}
}
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDAdminReport() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDReport(demo.MQTTTopic_CMDAdmin()),
		Handler: func(c phao.Client, msg phao.Message) {

			demo.DESMQTTClient.WG.Add(1)

			/* MAKE A COPY OF THE ADM TO PUBLISH IN A GO ROUTINE */
			adm := demo.ADM

			/* UPDATE SOURCE ADDRESS AND TIME */
			adm.AdmAddr = demo.DESDevSerial
			adm.AdmTime = time.Now().UTC().UnixMilli()

			/* SEND CONFIRMATION */
			go demo.MQTTPublication_DemoDeviceClient_SIGAdmin(adm)

			demo.DESMQTTClient.WG.Done()
		},
	}
}

/* SUBSCRIPTION -> HARDWARE ID -> UPON RECEIPT, LOG & REPLY TO .../sig/hwid */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDHwID() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDHwID(),
		Handler: func(c phao.Client, msg phao.Message) {

			demo.DESMQTTClient.WG.Add(1)

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			hw := HwID{}
			if err := json.Unmarshal(msg.Payload(), &hw); err != nil {
				pkg.TraceErr(err)
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteHwIDToFlash(demo.CmdArchiveName(), hw)
			hw_rec := hw

			/* UPDATE SOURCE ADDRESS ONLY */
			hw.HwAddr = demo.DESDevSerial

			if demo.EVT.EvtCode > STATUS_JOB_START_REQ {

				/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB */
				demo.WriteHwIDToFlash(demo.DESJobName, hw_rec) 

				/* UPDATE TIME ONLY WHEN LOGGING */
				hw.HwTime = time.Now().UTC().UnixMilli()

				/* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB */
				demo.WriteHwIDToFlash(demo.DESJobName, hw) 
			}

			/* WRITE (AS LOADED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteHwIDToFlash(demo.CmdArchiveName(), hw)

			/* LOAD VALUE INTO SIM 'RAM' */
			demo.HW = hw

			/* SEND CONFIRMATION */
			go demo.MQTTPublication_DemoDeviceClient_SIGHwID(hw)

			demo.DESMQTTClient.WG.Done()
		},
	}
}
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDHwIDReport() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDReport(demo.MQTTTopic_CMDHwID()),
		Handler: func(c phao.Client, msg phao.Message) {

			demo.DESMQTTClient.WG.Add(1)

			/* MAKE A COPY OF THE ADM TO PUBLISH IN A GO ROUTINE */
			hw := demo.HW

			/* UPDATE SOURCE ADDRESS AND TIME */
			hw.HwAddr = demo.DESDevSerial
			hw.HwTime = time.Now().UTC().UnixMilli()

			/* SEND HwID */
			go demo.MQTTPublication_DemoDeviceClient_SIGHwID(hw)

			demo.DESMQTTClient.WG.Done()
		},
	}
}

/* SUBSCRIPTIONS -> HEADER -> UPON RECEIPT, LOG & REPLY TO .../sig/header */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDHeader() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDHeader(),
		Handler: func(c phao.Client, msg phao.Message) {

			demo.DESMQTTClient.WG.Add(1)

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			hdr := Header{}
			if err := json.Unmarshal(msg.Payload(), &hdr); err != nil {
				pkg.TraceErr(err)
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteHdrToFlash(demo.CmdArchiveName(), demo.HDR)
			hdr_rec := hdr

			/* UPDATE SOURCE ADDRESS ONLY */
			hdr.HdrAddr = demo.DESDevSerial

			if demo.EVT.EvtCode > STATUS_JOB_START_REQ {

				/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB */
				demo.WriteHdrToFlash(demo.DESJobName, hdr_rec) 

				/* UPDATE TIME ONLY WHEN LOGGING */
				hdr.HdrTime = time.Now().UTC().UnixMilli()

				/* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB */
				demo.WriteHdrToFlash(demo.DESJobName, hdr) 
			}

			/* WRITE (AS LOADED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteHdrToFlash(demo.CmdArchiveName(), hdr)

			/* LOAD VALUE INTO SIM 'RAM' */
			demo.HDR = hdr

			/* SEND CONFIRMATION */
			go demo.MQTTPublication_DemoDeviceClient_SIGHeader(hdr)

			demo.DESMQTTClient.WG.Done()
		},
	}
}

/* SUBSCRIPTIONS -> CONFIGURATION -> UPON RECEIPT, LOG & REPLY TO .../sig/config */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDConfig() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDConfig(),
		Handler: func(c phao.Client, msg phao.Message) {

			demo.DESMQTTClient.WG.Add(1)

			/* CAPTURE EXISTING CFG */
			exCFG := demo.CFG

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			cfg := Config{}
			if err := json.Unmarshal(msg.Payload(), &cfg); err != nil {
				pkg.TraceErr(err)
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteCfgToFlash(demo.CmdArchiveName(), demo.CFG)
			cfg_rec := demo.CFG

			/* UPDATE SOURCE ADDRESS ONLY */
			cfg.CfgAddr = demo.DESDevSerial

			if demo.EVT.EvtCode > STATUS_JOB_START_REQ {

				/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB */
				demo.WriteCfgToFlash(demo.DESJobName, cfg_rec) 

				/* UPDATE TIME( ONLY WHEN LOGGING) */
				cfg.CfgTime = time.Now().UTC().UnixMilli()

				/* IF SAMPLE DATE HAS CHANGED, SEND UPDATE THE SIMULATION */
				if exCFG.CfgOpSample != cfg.CfgOpSample {
					demo.Rate <- cfg.CfgOpSample
				}

				/* IF VALVE TARGET HAS CHANGED, START A NEW MODE TRANSITION */
				if exCFG.CfgVlvTgt != cfg.CfgVlvTgt {
					demo.Mode <- cfg.CfgVlvTgt
					demo.TZero <- time.Now().UTC()
				}

				/* WRITE (AS LOADED) TO SIM 'FLASH' */
				demo.WriteCfgToFlash(demo.DESJobName, cfg) 
			}

			/* WRITE (AS LOADED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteCfgToFlash(demo.CmdArchiveName(), cfg)

			/* LOAD VALUE INTO SIM 'RAM' */
			demo.CFG = cfg

			/* SEND CONFIRMATION */
			go demo.MQTTPublication_DemoDeviceClient_SIGConfig(cfg)

			demo.DESMQTTClient.WG.Done()
		},
	}
}

/* SUBSCRIPTIONS -> EVENT -> UPON RECEIPT, LOG, HANDLE, & REPLY TO .../sig/event */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDEvent() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDEvent(),
		Handler: func(c phao.Client, msg phao.Message) {

			demo.DESMQTTClient.WG.Add(1)

			/* CAPTURE THE ORIGINAL DEVICE STATE EVENT CODE */
			state := demo.EVT.EvtCode

			/* CAPTURE INCOMING EVENT IN A NEW Event STRUCT TO
			PREVENT PREMATURE CHANGE IN DEVICE STATE */
			evt := Event{}

			/* PARSE / STORE THE EVENT IN CMDARCHIVE */
			if err := json.Unmarshal(msg.Payload(), &evt); err != nil {
				pkg.TraceErr(err)
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteEvtToFlash(demo.CmdArchiveName(), demo.EVT)
			evt_rec := evt

			/* UPDATE SOURCE ADDRESS ONLY */
			evt.EvtAddr = demo.DESDevSerial

			/* CHECK THE RECEIVED EVENT CODE */
			switch evt.EvtCode {

			// case 0:
			/* REGISTRATION EVENT: USED TO ASSIGN THIS DEVICE TO
			A DIFFERENT DATA EXCHANGE SERVER */

			case STATUS_JOB_END_REQ:
				go demo.EndDemoJob(evt)

			case STATUS_JOB_START_REQ:
				go demo.StartDemoJob(evt)

			default:

				/* CHECK THE ORIGINAL DEVICE STATE EVENT CODE
				TO SEE IF WE SHOULD WRITE TO THE ACTIVE JOB */
				if state > STATUS_JOB_START_REQ {

					/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB */
					demo.WriteEvtToFlash(demo.DESJobName, evt_rec) 

					/* UPDATE TIME( ONLY WHEN LOGGING) */
					evt.EvtTime = time.Now().UTC().UnixMilli()

					/* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB */
					demo.WriteEvtToFlash(demo.DESJobName, evt) 
				}

				/* WRITE (AS LOADED) TO SIM 'FLASH' -> CMDARCHIVE */
				demo.WriteEvtToFlash(demo.CmdArchiveName(), evt)

				/* LOAD VALUE INTO SIM 'RAM' */
				demo.EVT = evt

				/* SEND CONFIRMATION */
				go demo.MQTTPublication_DemoDeviceClient_SIGEvent(evt)
			}

			demo.DESMQTTClient.WG.Done()
		},
	}
}

/* PUBLICATIONS ******************************************************************************************/

/* PUBLICATION -> ADMIN -> SIMULATED ADMINS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGAdmin(adm Admin) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/
	sig := pkg.MQTTPublication{

		Topic:   demo.MQTTTopic_SIGAdmin(),
		Message: pkg.ModelToJSONString(adm),
		// Message:  pkg.MakeMQTTMessage(adm.FilterAdmRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	sig.Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> HARDWARE ID  -> SIMULATED HARDWARE IDS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGHwID(hw HwID) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/
	sig := pkg.MQTTPublication{

		Topic:   demo.MQTTTopic_SIGHwID(),
		Message: pkg.ModelToJSONString(hw),
		// Message:  pkg.MakeMQTTMessage(hw.FilterAdmRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	sig.Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> HEADER -> SIMULATED HEADERS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGHeader(hdr Header) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/
	sig := pkg.MQTTPublication{

		Topic:   demo.MQTTTopic_SIGHeader(),
		Message: pkg.ModelToJSONString(hdr),
		// Message:  pkg.MakeMQTTMessage(hdr.FilterHdrRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	sig.Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> CONFIG -> SIMULATED CONFIGS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGConfig(cfg Config) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/
	sig := pkg.MQTTPublication{

		Topic:   demo.MQTTTopic_SIGConfig(),
		Message: pkg.ModelToJSONString(cfg),
		// Message:  pkg.MakeMQTTMessage(cfg.FilterCfgRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	sig.Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> EVENT -> SIMULATED EVENTS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGEvent(evt Event) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/
	sig := pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGEvent(),
		Message:  pkg.ModelToJSONString(evt),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	sig.Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> SAMPLE -> SIMULATED SAMPLES */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGSample(mqtts MQTT_Sample) {

	b64, err := json.Marshal(mqtts)
	if err != nil {
		pkg.TraceErr(err)
	} // pkg.Json("MQTT_Sample:", b64)

	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/
	sig := pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGSample(),
		Message:  string(b64),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	sig.Pub(demo.DESMQTTClient)
}

/* SIMULATIONS *******************************************************************************************/

func (demo *DemoDeviceClient) StartDemoJob(evt Event) {
	fmt.Printf("\n(demo *DemoDeviceClient) StartDemoJob( )...\n")

	/* TODO: MAKE SURE THE PREVIOUS JOB IS ENDED */
	// if state > STATUS_JOB_START_REQ {
	// 	demo.EndDemoJob()
	// }

	demo.DESMQTTClient.WG.Wait()
	demo.DESMQTTClient.WG.Add(1)

	/* CAPTURE TIME VALUE FOR JOB INTITALIZATION: DB/JOB NAME, ADM, HDR, CFG, EVT */
	startTime := time.Now().UTC().UnixMilli()

	/* USED INCASE WE NEED TO CREATE DEFAULT SETTINGS */
	demo.DESJob = pkg.DESJob{
			DESJobRegTime:   startTime,
			DESJobRegAddr:   demo.DESDevSerial,
			DESJobRegUserID: evt.EvtUserID,
			DESJobRegApp:    evt.EvtApp,

			DESJobName:  fmt.Sprintf("%s_%d", demo.DESDevSerial, startTime),
			DESJobStart: startTime,
			DESJobEnd:   0,
			DESJobLng:   -114.75 + rand.Float32()*(-110.15+114.75),
			DESJobLat:   51.85 + rand.Float32()*(54.35-51.85),
			DESJobDevID: demo.DESDevID,
	}

	/* WHERE JOB START ADMIN WAS NOT RECEIVED, USE DEFAULT VALUES */
	if demo.ADM.AdmTime != evt.EvtTime {
		fmt.Printf("(demo *DemoDeviceClient) StartDemoJob -> USING DEFAULT ADMIN.\n")
		demo.ADM.DefaultSettings_Admin(demo.DESRegistration)
	}
	demo.ADM.AdmTime = startTime
	demo.ADM.AdmAddr = demo.DESDevSerial
	demo.ADM.AdmUserID = evt.EvtUserID
	demo.ADM.AdmApp = evt.EvtApp

	demo.HW.HwTime = startTime
	demo.HW.HwAddr = demo.DESDevSerial
	demo.HW.HwUserID = evt.EvtUserID
	demo.HW.HwApp = evt.EvtApp
	demo.HW.HwLogFw = "0.0.009"
	demo.HW.HwModFw = "0.0.007"

	/* WHERE JOB START HEADER WAS NOT RECEIVED, USE DEFAULT VALUES */
	if demo.HDR.HdrTime != evt.EvtTime {
		fmt.Printf("(demo *DemoDeviceClient) StartDemoJob -> USING DEFAULT HEADER\n")
		demo.HDR.DefaultSettings_Header(demo.DESRegistration)
	}
	demo.HDR.HdrTime = startTime
	demo.HDR.HdrAddr = demo.DESDevSerial
	demo.HDR.HdrUserID = evt.EvtUserID
	demo.HDR.HdrApp = evt.EvtApp
	demo.HDR.HdrJobName = fmt.Sprintf("%s_%d", demo.DESDevSerial, startTime)
	demo.HDR.HdrJobStart = startTime
	demo.HDR.HdrJobEnd = 0
	demo.HDR.HdrGeoLng = -114.75 + rand.Float32()*(-110.15+114.75)
	demo.HDR.HdrGeoLat = 51.85 + rand.Float32()*(54.35-51.85)
	if demo.HDR.HdrWellName == "" {
		demo.HDR.HdrWellName = demo.HDR.HdrJobName
	}

	/* WHERE JOB START CONFIG WAS NOT RECEIVED, USE DEFAULT VALUES */
	if demo.CFG.CfgTime != evt.EvtTime {
		fmt.Printf("(demo *DemoDeviceClient) StartDemoJob -> USING DEFAULT CONFIG.\n")
		demo.CFG.DefaultSettings_Config(demo.DESRegistration)
	}
	demo.CFG.CfgTime = startTime
	demo.CFG.CfgAddr = demo.DESDevSerial
	demo.CFG.CfgUserID = evt.EvtUserID
	demo.CFG.CfgApp = evt.EvtApp
	demo.CFG.CfgVlvTgt = MODE_VENT
	demo.CFG.Validate()

	evt.EvtTime = startTime
	evt.EvtAddr = demo.DESDevSerial
	evt.EvtCode = STATUS_JOB_STARTED
	evt.EvtTitle = "JOB STARTED"
	evt.EvtMsg = demo.HDR.HdrJobName

	/* WRITE TO FLASH - CMDARCHIVE */
	demo.WriteAdmToFlash(demo.CmdArchiveName(), demo.ADM)
	demo.WriteHwIDToFlash(demo.CmdArchiveName(), demo.HW)
	demo.WriteHdrToFlash(demo.CmdArchiveName(), demo.HDR)
	demo.WriteCfgToFlash(demo.CmdArchiveName(), demo.CFG)
	demo.WriteEvtToFlash(demo.CmdArchiveName(), evt)

	/* WRITE TO FLASH - JOB */
	demo.WriteAdmToFlash(demo.DESJobName, demo.ADM) 
	demo.WriteHwIDToFlash(demo.DESJobName, demo.HW) 
	demo.WriteHdrToFlash(demo.DESJobName, demo.HDR) 
	demo.WriteCfgToFlash(demo.DESJobName, demo.CFG) 
	demo.WriteEvtToFlash(demo.DESJobName, evt) 

	/* LOAD VALUE INTO SIM 'RAM'
	UPDATE THE DEVICE EVENT CODE, ENABLING MQTT MESSAGE WRITES TO ACTIVE JOB
	AFTER WE HAVE WRITTEN THE INITIAL JOB RECORDS
	*/
	demo.EVT = evt

	/* SEND CONFIRMATION */
	go demo.MQTTPublication_DemoDeviceClient_SIGAdmin(demo.ADM)
	go demo.MQTTPublication_DemoDeviceClient_SIGHwID(demo.HW)
	go demo.MQTTPublication_DemoDeviceClient_SIGHeader(demo.HDR)
	go demo.MQTTPublication_DemoDeviceClient_SIGConfig(demo.CFG)
	go demo.MQTTPublication_DemoDeviceClient_SIGEvent(evt)

	/* RUN JOB... */
	go demo.Demo_Simulation(demo.HDR.HdrJobName, demo.CFG.CfgVlvTgt, demo.CFG.CfgOpSample)

	demo.DESMQTTClient.WG.Done()

	fmt.Printf("\n(demo *DemoDeviceClient) StartDemoJob( ) -> RUNNING %s...\n", demo.HDR.HdrJobName)
}

func (demo *DemoDeviceClient) EndDemoJob(evt Event) {
	fmt.Printf("\n(demo *DemoDeviceClient) EndDemoJob( )...\n")

	demo.DESMQTTClient.WG.Wait()
	demo.DESMQTTClient.WG.Add(1)
	demo.Stop <- struct{}{}

	/* CAPTURE TIME VALUE FOR JOB TERMINATION: HDR, EVT */
	endTime := time.Now().UTC().UnixMilli()

	// demo.GetHdrFromFlash(demo.CmdArchiveName(), &demo.HDR)
	demo.HDR.HdrTime = endTime
	demo.HDR.HdrAddr = demo.DESDevSerial
	demo.HDR.HdrUserID = evt.EvtUserID
	demo.HDR.HdrApp = evt.EvtApp
	demo.HDR.HdrJobEnd = endTime

	evt.EvtTime = endTime
	evt.EvtAddr = demo.DESDevSerial
	evt.EvtCode = STATUS_JOB_ENDED
	evt.EvtTitle = "JOB ENDED"
	evt.EvtMsg = demo.HDR.HdrJobName

	/* LOAD VALUE INTO SIM 'RAM'
	UPDATE THE DEVICE EVENT CODE, DISABLING MQTT MESSAGE WRITES TO ACTIVE JOB
	BEFORE WE HAVE WRITTEN THE FINAL JOB RECORDS
	*/
	demo.EVT = evt

	/* WRITE TO FLASH - CMDARCHIVE */
	demo.WriteEvtToFlash(demo.CmdArchiveName(), evt)
	demo.WriteHdrToFlash(demo.CmdArchiveName(), demo.HDR)

	/* WRITE TO FLASH - JOB */
	demo.WriteEvtToFlash(demo.DESJobName, evt) 
	demo.WriteHdrToFlash(demo.DESJobName, demo.HDR) 

	/* SEND CONFIRMATION */
	go demo.MQTTPublication_DemoDeviceClient_SIGHeader(demo.HDR)
	go demo.MQTTPublication_DemoDeviceClient_SIGEvent(evt)

	demo.DESMQTTClient.WG.Done()

	fmt.Printf("\n(demo *DemoDeviceClient) EndDemoJob( ) -> ENDED: %s\n", demo.HDR.HdrJobName)
}

/*DEMO SIM -> PUBLISH TO MQTT */
func (demo *DemoDeviceClient) Demo_Simulation(job string, mode, rate int32) {
	// fmt.Printf("\n(demo) Demo_Simulation( ) %s -> Starting simulation...\n", demo.DESDevSerial)

	demo.DESMQTTClient.WG.Wait()
	smp := demo.SMP

	/* CREATE RANDOM SIMULATED WELL CONDITIONS */
	demo.Set_MTx()
	smp.SmpCH4 = demo.MTxCh4.VMin
	smp.SmpHiFlow = demo.MTxHiFlow.VMin
	smp.SmpLoFlow = demo.MTxLoFlow.VMin
	smp.SmpPress = demo.MTxBuild.VMin

	tZero := time.Now().UTC()

	TakeSmp := make(chan struct{})

	sleep := true
	go func() {
		for sleep {
			select {

			case <-demo.Stop:
				sleep = false

			case rate = <-demo.Rate:

			default:
				TakeSmp <- struct{}{}
				time.Sleep(time.Millisecond * time.Duration(rate))
			}
		}

		close(TakeSmp)
		TakeSmp = nil
	}()

	run := true
	for run {
		select {

		case <-demo.Stop:
			run = false

		case tZero = <-demo.TZero:

		case mode = <-demo.Mode:

		case <-TakeSmp:
			t := time.Now().UTC()
			demo.Demo_Simulation_Take_Sample(tZero, t, mode, job, &smp)
			demo.WriteSmpToFlash(job, smp)
			smpMQTT := Demo_EncodeMQTTSampleMessage(job, 0, smp)
			demo.MQTTPublication_DemoDeviceClient_SIGSample(smpMQTT)
		}
	}

	fmt.Printf("\n(demo) Demo_Simulation( ) -> SIMULATION STOPPED: %s\n", demo.DESDevSerial)
}

func (demo *DemoDeviceClient) Demo_Simulation_Take_Sample(t0, ti time.Time, mode int32, job string, smp *Sample) {

	smp.SmpTime = ti.UnixMilli()

	if mode == MODE_VENT {
		demo.Set_MTxVent(t0, ti, smp)
	}
	if mode == MODE_BUILD {
		demo.Set_MTxBuild(t0, ti, smp)
	}
	if mode == MODE_HI_FLOW || mode == MODE_LO_FLOW {
		demo.Set_MTxFlow(t0, ti, smp)
	}

	smp.SmpBatAmp = 0.049 + rand.Float32()*0.023
	smp.SmpBatVolt = 12.733 + rand.Float32()*0.072
	smp.SmpMotVolt = 11.9 + rand.Float32()*0.033

	smp.SmpVlvTgt = uint32(mode)
	smp.SmpVlvPos = uint32(mode)

	smp.SmpJobName = job
}

/* CREATE RANDOM SIMULATED WELL CONDITIONS */
var maxCh4 = float32(97.99)
var minCh4 = float32(73.99)

var maxFlow = float32(239.99)
var minFlow = float32(0.23)

var maxPress = float32(6205.99)
var minPress = float32(101.99)

func (demo *DemoDeviceClient) Set_MTx() {

	demo.MTxCh4.VMax = minCh4 + rand.Float32()*(maxCh4-minCh4)
	demo.MTxCh4.VMin = 0.01
	demo.MTxCh4.TSpanUp = time.Duration(time.Second * 35)
	demo.MTxCh4.TSpanDn = time.Duration(time.Second * 70)
	demo.MTxCh4.VRes = float32(100) * 0.005

	demo.MTxHiFlow.VMax = minFlow + rand.Float32()*(maxFlow-minFlow)
	demo.MTxHiFlow.VMin = 0.01
	demo.MTxHiFlow.TSpanUp = time.Duration(time.Second * 70)
	demo.MTxHiFlow.TSpanDn = time.Duration(time.Second * 35)
	demo.MTxHiFlow.VRes = float32(250) * 0.005

	demo.MTxLoFlow.VMax = demo.MTxHiFlow.VMax
	demo.MTxLoFlow.VMin = 0.01
	demo.MTxLoFlow.TSpanUp = time.Duration(time.Second * 70)
	demo.MTxLoFlow.TSpanDn = time.Duration(time.Second * 35)
	demo.MTxLoFlow.VRes = float32(2.0) * 0.005

	demo.MTxBuild.VMax = (demo.MTxHiFlow.VMax / maxFlow) * maxPress
	demo.MTxBuild.VMin = minPress
	demo.MTxBuild.TSpanUp = time.Duration(time.Second * 275)
	demo.MTxBuild.TSpanDn = time.Duration(time.Second * 150)
	demo.MTxBuild.VRes = maxPress * 0.0005

	fmt.Printf("\n(demo *DemoDeviceClient) Set_MTx() -> %s:, %f, %f, %f, %f\n", demo.DESDevSerial, demo.MTxCh4.VMax, demo.MTxHiFlow.VMax, demo.MTxLoFlow.VMax, demo.MTxBuild.VMax)
}

func (demo *DemoDeviceClient) Set_MTxVent(t0, ti time.Time, smp *Sample) {
	smp.SmpCH4 = demo.MTxCh4.MTx_CalcModeTransValue(t0, ti, smp.SmpCH4, demo.MTxCh4.VMin, true)
	smp.SmpHiFlow = demo.MTxHiFlow.MTx_CalcModeTransValue(t0, ti, smp.SmpHiFlow, demo.MTxHiFlow.VMin, true)
	smp.SmpLoFlow = demo.MTxLoFlow.MTx_CalcModeTransValue(t0, ti, smp.SmpLoFlow, demo.MTxLoFlow.VMin, true)
	smp.SmpPress = demo.MTxBuild.MTx_CalcModeTransValue(t0, ti, smp.SmpPress, demo.MTxBuild.VMin, true)
}
func (demo *DemoDeviceClient) Set_MTxBuild(t0, ti time.Time, smp *Sample) {
	smp.SmpCH4 = demo.MTxCh4.MTx_CalcModeTransValue(t0, ti, smp.SmpCH4, demo.MTxCh4.VMin, false)
	smp.SmpHiFlow = demo.MTxHiFlow.MTx_CalcModeTransValue(t0, ti, smp.SmpHiFlow, demo.MTxHiFlow.VMin, false)
	smp.SmpLoFlow = demo.MTxLoFlow.MTx_CalcModeTransValue(t0, ti, smp.SmpLoFlow, demo.MTxLoFlow.VMin, false)
	smp.SmpPress = demo.MTxBuild.MTx_CalcModeTransValue(t0, ti, smp.SmpPress, demo.MTxBuild.VMax, false)
}
func (demo *DemoDeviceClient) Set_MTxFlow(t0, ti time.Time, smp *Sample) {
	smp.SmpCH4 = demo.MTxCh4.MTx_CalcModeTransValue(t0, ti, smp.SmpCH4, demo.MTxCh4.VMax, false)
	smp.SmpHiFlow = demo.MTxHiFlow.MTx_CalcModeTransValue(t0, ti, smp.SmpHiFlow, demo.MTxHiFlow.VMax, false)
	smp.SmpLoFlow = demo.MTxLoFlow.MTx_CalcModeTransValue(t0, ti, smp.SmpLoFlow, demo.MTxLoFlow.VMax, false)
	fp := ((demo.MTxHiFlow.VMax / maxFlow) * minPress * 2) + minPress
	smp.SmpPress = demo.MTxBuild.MTx_CalcModeTransValue(t0, ti, smp.SmpPress, fp, false)
}

func Demo_EncodeMQTTSampleMessage(job string, i int, smp Sample) MQTT_Sample {
	// fmt.Println("\nDemo_EncodeMQTTSampleMessage()...")

	data := []pkg.TimeSeriesData{
		{Data: []pkg.TSDPoint{{X: smp.SmpTime, Y: smp.SmpCH4}}, Min: 0, Max: 100},
		{Data: []pkg.TSDPoint{{X: smp.SmpTime, Y: smp.SmpHiFlow}}, Min: 0, Max: 250},
		{Data: []pkg.TSDPoint{{X: smp.SmpTime, Y: smp.SmpLoFlow}}, Min: 0, Max: 2},
		{Data: []pkg.TSDPoint{{X: smp.SmpTime, Y: smp.SmpPress}}, Min: 0, Max: 1500},
		{Data: []pkg.TSDPoint{{X: smp.SmpTime, Y: smp.SmpBatAmp}}, Min: 0, Max: 1.5},
		{Data: []pkg.TSDPoint{{X: smp.SmpTime, Y: smp.SmpBatVolt}}, Min: 0, Max: 15},
		{Data: []pkg.TSDPoint{{X: smp.SmpTime, Y: smp.SmpMotVolt}}, Min: 0, Max: 15},
		{Data: []pkg.TSDPoint{{X: smp.SmpTime, Y: float32(smp.SmpVlvTgt)}}, Min: 0, Max: 10},
		{Data: []pkg.TSDPoint{{X: smp.SmpTime, Y: float32(smp.SmpVlvPos)}}, Min: 0, Max: 10},
	}

	x := data[0].Data[i].X                  // fmt.Printf("Time:\t%d\n", x)
	var ch float32 = data[0].Data[i].Y      // fmt.Printf("CH4:\t%f\n", ch)
	var hf float32 = data[1].Data[i].Y      // fmt.Printf("High Flow:\t%f\n", hf)
	var lf float32 = data[2].Data[i].Y      // fmt.Printf("Low Flow:\t%f\n", lf)
	var p float32 = data[3].Data[i].Y       // fmt.Printf("Pressure:\t%f\n", p)
	var bc float32 = data[4].Data[i].Y      // fmt.Printf("Batt C:\t%f\n", bc)
	var bv float32 = data[5].Data[i].Y      // fmt.Printf("Batt V:\t%f\n", bv)
	var mv float32 = data[6].Data[i].Y      // fmt.Printf("Motor V:\t%f\n", mv)
	var vt int16 = int16(data[7].Data[i].Y) // fmt.Printf("Target V:\t%d\n", vt)
	var vp int16 = int16(data[8].Data[i].Y) // fmt.Printf("Target V:\t%d\n", vp)

	var hex []byte
	hex = append(hex, pkg.GetBytes(x)...)
	hex = append(hex, pkg.GetBytes(ch)...)
	hex = append(hex, pkg.GetBytes(hf)...)
	hex = append(hex, pkg.GetBytes(lf)...)
	hex = append(hex, pkg.GetBytes(p)...)
	hex = append(hex, pkg.GetBytes(bc)...)
	hex = append(hex, pkg.GetBytes(bv)...)
	hex = append(hex, pkg.GetBytes(mv)...)
	hex = append(hex, pkg.GetBytes(vt)...)
	hex = append(hex, pkg.GetBytes(vp)...)
	// fmt.Printf("Hex:\t%X\n", hex)

	b64 := pkg.BytesToBase64(hex)
	// fmt.Printf("Base64:\t%s\n\n", b64)

	msg := MQTT_Sample{
		DesJobName: job,
		Data:       []string{b64},
	}

	return msg
}

func (mtx *DemoModeTransition) MTx_CalcModeTransValue(t_start, ti time.Time, vi, v_end float32, vent bool) (value float32) {

	/* SPAN OF THE VALUE TRANSITION */
	v_span := float64(v_end - vi)

	/* TOTAL TRANSITION TIME */
	var t_span time.Duration

	if vent {
		t_span = time.Duration(time.Second * 30)
	} else if vi < v_end {
		t_span = mtx.TSpanUp
	} else {
		t_span = mtx.TSpanDn
	}

	/* TIME RIGHT NOW RELATIVE TO THE TOTAL TRANSITION TIME */
	t_rel := float64(ti.Sub(t_start).Seconds() / t_span.Seconds())

	/* MAGIC NUMBERS BASED ON CURRENT VALUE AND RELATIVE TIME */
	a := v_span * math.Pow(t_rel, 2)
	bx := 0.5125
	b := 1 - math.Pow((bx-t_rel), 4)

	/* SIMULATED VALUE */
	value = vi + float32(a*b)

	/* MAX ERROR */
	res := mtx.VRes

	if b < 0.8 {
		value = v_end
	} else {
		/* ENSURE THE SIMULATED VALUE DOESN'T DIP THE WRONG WAY */
		if v_span > 1 && value < (vi+res) {
			value = vi + res
		} else if v_span < 1 && value > (vi-res) {
			value = vi - res
		}
	}

	/* ENSURE THE SIMULATED VALUE REMAINS HIGHER THAN MIN */
	if value < mtx.VMin {
		value = mtx.VMin + res
	}

	/* ENSURE THE SIMULATED VALUE REMAINS LOWER THAN MAX */
	if value > mtx.VMax {
		value = mtx.VMax - res
	}

	/* ADD SOME NOISE */
	min := value - res
	value = min + rand.Float32()*res
	return
}

func YSinX(t0, ti time.Time, max, shift float64) (y float32) {

	freq := 0.5
	dt := ti.Sub(t0).Seconds()
	a := max / 2

	return float32(a * (math.Sin(dt*freq+(freq/shift)) + 1))
}
func YCosX(t0, ti time.Time, max, shift float64) (y float32) {

	freq := 0.5
	dt := ti.Sub(t0).Seconds()
	a := max / 2

	return float32(a * (math.Cos(dt*freq+(freq/shift)) + 1))
}

/* FOR TESTING SIMULATED FLASH WRITE */
func DemoSimFlashTest() {

	cfg := Config{
		CfgTime:   time.Now().UTC().UnixMilli(),
		CfgAddr:   "aaaabbbbccccddddeeeeffffgggghhhh",
		CfgUserID: "ef0589a4-5ad4-45ea-9575-5aaee0568b0c",
		CfgApp:    "aaaabbbbccccddddeeeeffffgggghhhh",

		/* JOB */
		CfgSCVD:     596.8, // m
		CfgSCVDMult: 10.5,  // kPa / m
		CfgSSPRate:  1.95,  // kPa / hour
		CfgSSPDur:   6.0,   // hour
		CfgHiSCVF:   201.4, //  L/min
		CfgFlowTog:  1.85,  // L/min

		/* VALVE */
		CfgVlvTgt: 2, // vent
		CfgVlvPos: 2, // vent

		/* OP PERIODS*/
		CfgOpSample: 1000, // millisecond
		CfgOpLog:    1000, // millisecond
		CfgOpTrans:  1000, // millisecond

		/* DIAG PERIODS */
		CfgDiagSample: 10000,  // millisecond
		CfgDiagLog:    100000, // millisecond
		CfgDiagTrans:  600000, // millisecond
	}

	cfgBytes := cfg.ConfigToBytes()
	fmt.Printf("\ncfgBytes ( %d ) : %v\n", len(cfgBytes), cfgBytes)

	dir := fmt.Sprintf("demo/%s", "test")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.TraceErr(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/cfg.bin", dir), os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		pkg.TraceErr(err)
	}
	defer f.Close()

	_, err = f.Write(cfg.ConfigToBytes())
	if err != nil {
		pkg.TraceErr(err)
	}

	f.Close()
}

/* ADM DEMO MEMORY -> JSON*/
func (demo DemoDeviceClient) WriteAdmToFlash(jobName string, adm Admin) (err error) {

	admJson := pkg.ModelToJSONString(adm)
	// fmt.Printf("\nadmBytes ( %d ) : %x\n", len(admBytes), admBytes)

	dir := fmt.Sprintf("demo/%s", jobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.TraceErr(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/adm.json", dir), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return pkg.TraceErr(err)
	}
	defer f.Close()

	_, err = f.WriteString(admJson)
	if err != nil {
		return pkg.TraceErr(err)
	}

	f.Close()
	return
}

/* ADM DEMO MEMORY -> 272 BYTES -> HxD 34 x 8 */
func (demo DemoDeviceClient) WriteAdmToFlashHex(jobName string, adm Admin) (err error) {

	admBytes := adm.AdminToBytes()
	// fmt.Printf("\nadmBytes ( %d ) : %x\n", len(admBytes), admBytes)

	dir := fmt.Sprintf("demo/%s", jobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.TraceErr(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/adm.bin", dir), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return pkg.TraceErr(err)
	}
	defer f.Close()

	_, err = f.Write(admBytes)
	if err != nil {
		return pkg.TraceErr(err)
	}

	f.Close()
	return
}
func (demo *DemoDeviceClient) ReadAdmFromFlashHex(jobName string) (adm []byte, err error) {

	dir := fmt.Sprintf("demo/%s", jobName)
	f, err := os.OpenFile(fmt.Sprintf("%s/adm.bin", dir), os.O_RDONLY, 0600)
	if err != nil {
		return nil, pkg.TraceErr(err)
	}

	admFile, err := ioutil.ReadAll(f)
	if err != nil {
		pkg.TraceErr(err)
		return
	}
	eof := len(admFile)
	adm = admFile[eof-272 : eof]
	// fmt.Printf("\nadmBytes ( %d ) : %v\n", len(adm), adm)

	f.Close()
	return
}
func (demo *DemoDeviceClient) GetAdmFromFlashHex(jobName string, adm *Admin) {
	b, err := demo.ReadAdmFromFlashHex(jobName)
	if err != nil {
		pkg.TraceErr(err)
	}
	adm.AdminFromBytes(b)
}

/* HW DEMO MEMORY -> JSON */
func (demo DemoDeviceClient) WriteHwIDToFlash(jobName string, hw HwID) (err error) {

	hwJson := pkg.ModelToJSONString(hw)
	// fmt.Printf("\nhwBytes ( %d ) : %x\n", len(hwBytes), hwBytes)

	dir := fmt.Sprintf("demo/%s", jobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.TraceErr(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/hw.json", dir), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return pkg.TraceErr(err)
	}
	defer f.Close()

	_, err = f.WriteString(hwJson)
	if err != nil {
		return pkg.TraceErr(err)
	}

	f.Close()
	return
}

/* HW DEMO MEMORY -> 152 BYTES -> HxD 38 x 4 */
func (demo DemoDeviceClient) WriteHwIDToFlashHex(jobName string, hw HwID) (err error) {

	hwBytes := hw.HwIDToBytes()
	// fmt.Printf("\nhwBytes ( %d ) : %x\n", len(hwBytes), hwBytes)

	dir := fmt.Sprintf("demo/%s", jobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.TraceErr(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/hw.bin", dir), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return pkg.TraceErr(err)
	}
	defer f.Close()

	_, err = f.Write(hwBytes)
	if err != nil {
		return pkg.TraceErr(err)
	}

	f.Close()
	return
}
func (demo *DemoDeviceClient) ReadHwIDFromFlashHex(jobName string) (hw []byte, err error) {

	dir := fmt.Sprintf("demo/%s", jobName)
	f, err := os.OpenFile(fmt.Sprintf("%s/hw.bin", dir), os.O_RDONLY, 0600)
	if err != nil {
		return nil, pkg.TraceErr(err)
	}

	hwFile, err := ioutil.ReadAll(f)
	if err != nil {
		pkg.TraceErr(err)
		return
	}
	eof := len(hwFile)
	hw = hwFile[eof-136 : eof]
	fmt.Printf("\nhwBytes ( %d ) : %v\n", len(hw), hw)

	f.Close()
	return
}
func (demo *DemoDeviceClient) GetHwIDFromFlashHex(jobName string, hw *HwID) {
	b, err := demo.ReadHwIDFromFlashHex(jobName)
	if err != nil {
		pkg.TraceErr(err)
	}
	hw.HwIDFromBytes(b)
}

/* HDR DEMO MEMORY -> JSON */
func (demo *DemoDeviceClient) WriteHdrToFlash(jobName string, hdr Header) (err error) {

	hdrJson := pkg.ModelToJSONString(hdr)
	// fmt.Printf("\nhdrBytes ( %d ) : %x\n", len(hdrBytes), hdrBytes)

	dir := fmt.Sprintf("demo/%s", jobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.TraceErr(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/hdr.json", dir), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return pkg.TraceErr(err)
	}
	defer f.Close()

	_, err = f.WriteString(hdrJson)
	if err != nil {
		return pkg.TraceErr(err)
	}

	f.Close()
	return
}

/* HDR DEMO MEMORY -> 324 BYTES -> HxD 54 x 6 */
func (demo *DemoDeviceClient) WriteHdrToFlashHex(jobName string, hdr Header) (err error) {

	hdrBytes := hdr.HeaderToBytes()
	// fmt.Printf("\nhdrBytes ( %d ) : %x\n", len(hdrBytes), hdrBytes)

	dir := fmt.Sprintf("demo/%s", jobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.TraceErr(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/hdr.bin", dir), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return pkg.TraceErr(err)
	}
	defer f.Close()

	_, err = f.Write(hdrBytes)
	if err != nil {
		return pkg.TraceErr(err)
	}

	f.Close()
	return
}
func (demo *DemoDeviceClient) ReadHdrFromFlashHex(jobName string) (hdr []byte, err error) {

	dir := fmt.Sprintf("demo/%s", jobName)
	f, err := os.OpenFile(fmt.Sprintf("%s/hdr.bin", dir), os.O_RDONLY, 0600)
	if err != nil {
		return nil, pkg.TraceErr(err)
	}

	hdrFile, err := ioutil.ReadAll(f)
	if err != nil {
		pkg.TraceErr(err)
		return
	}
	eof := len(hdrFile)
	hdr = hdrFile[eof-324 : eof]
	// fmt.Printf("\nhdrBytes ( %d ) : %v\n", len(hdr), hdr)

	f.Close()
	return
}
func (demo *DemoDeviceClient) GetHdrFromFlashHex(jobName string, hdr *Header) {
	b, err := demo.ReadHdrFromFlashHex(jobName)
	if err != nil {
		pkg.TraceErr(err)
	}
	hdr.HeaderFromBytes(b)
}

/* CFG DEMO MEMORY -> JSON */
func (demo *DemoDeviceClient) WriteCfgToFlash(jobName string, cfg Config) (err error) {

	cfgJson := pkg.ModelToJSONString(cfg)
	// fmt.Printf("\ncfgBytes ( %d ) : %x\n", len(cfgBytes), cfgBytes)

	dir := fmt.Sprintf("demo/%s", jobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.TraceErr(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/cfg.json", dir), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return pkg.TraceErr(err)
	}
	defer f.Close()

	_, err = f.WriteString(cfgJson)
	if err != nil {
		return pkg.TraceErr(err)
	}

	f.Close()
	return
}

/* CFG DEMO MEMORY -> 172 BYTES -> HxD 43 x 4 */
func (demo *DemoDeviceClient) WriteCfgToFlashHex(jobName string, cfg Config) (err error) {

	cfgBytes := cfg.ConfigToBytes()
	// fmt.Printf("\ncfgBytes ( %d ) : %x\n", len(cfgBytes), cfgBytes)

	dir := fmt.Sprintf("demo/%s", jobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.TraceErr(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/cfg.bin", dir), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return pkg.TraceErr(err)
	}
	defer f.Close()

	_, err = f.Write(cfgBytes)
	if err != nil {
		return pkg.TraceErr(err)
	}

	f.Close()
	return
}
func (demo *DemoDeviceClient) ReadCfgFromFlashHex(jobName string) (cfg []byte, err error) {

	dir := fmt.Sprintf("demo/%s", jobName)
	f, err := os.OpenFile(fmt.Sprintf("%s/cfg.bin", dir), os.O_RDONLY, 0600)
	if err != nil {
		return nil, pkg.TraceErr(err)
	}

	cfgFile, err := ioutil.ReadAll(f)
	if err != nil {
		pkg.TraceErr(err)
		return
	}
	eof := len(cfgFile)
	cfg = cfgFile[eof-172 : eof]
	// fmt.Printf("\ncfgBytes ( %d ) : %v\n", len(cfg), cfg)

	f.Close()
	return
}
func (demo *DemoDeviceClient) GetCfgFromFlashHex(jobName string, cfg *Config) {
	b, err := demo.ReadCfgFromFlashHex(jobName)
	if err != nil {
		pkg.TraceErr(err)
	}
	cfg.ConfigFromBytes(b)
}

/* EVT DEMO MEMORY -> JSON */
func (demo *DemoDeviceClient) WriteEvtToFlash(jobName string, evt Event) (err error) {

	evtJson := pkg.ModelToJSONString(evt)
	// fmt.Printf("\nevtBytes ( %d ) : %x\n", len(evtBytes), evtBytes)

	dir := fmt.Sprintf("demo/%s/events", jobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.TraceErr(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/%d.json", dir, evt.EvtTime), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return pkg.TraceErr(err)
	}
	defer f.Close()

	_, err = f.WriteString(evtJson)
	if err != nil {
		return pkg.TraceErr(err)
	}

	f.Close()
	return
}

/* EVT DEMO MEMORY -> 156 BYTES + MESSAGE(128) -> HxD 29  x 4 (Code @  116 dec*/
func (demo *DemoDeviceClient) WriteEvtToFlashHex(jobName string, evt Event) (err error) {

	evtBytes := evt.EventToBytes()
	// fmt.Printf("\nevtBytes ( %d ) : %x\n", len(evtBytes), evtBytes)

	dir := fmt.Sprintf("demo/%s/events", jobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.TraceErr(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/%d.bin", dir, evt.EvtTime), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return pkg.TraceErr(err)
	}
	defer f.Close()

	_, err = f.Write(evtBytes)
	if err != nil {
		return pkg.TraceErr(err)
	}

	f.Close()
	return
}
func (demo *DemoDeviceClient) ReadEvtFromFlashHex(jobName string, time int64) (evt []byte, err error) {

	dir := fmt.Sprintf("demo/%s/events", jobName)
	f, err := os.OpenFile(fmt.Sprintf("%s/%d.bin", dir, time), os.O_RDONLY, 0600)
	if err != nil {
		pkg.TraceErr(err)
		return
	}

	evt, err = ioutil.ReadAll(f)
	if err != nil {
		pkg.TraceErr(err)
		return
	}
	// fmt.Printf("\nevtBytes ( %d ) : %v\n", len(evt), evt)

	f.Close()
	return
}
func (demo *DemoDeviceClient) GetEvtFromFlashHex(jobName string, time int64, evt *Event) {
	b, err := demo.ReadEvtFromFlashHex(jobName, time)
	if err != nil {
		pkg.TraceErr(err)
	}
	evt.EventFromBytes(b)
}
func (demo *DemoDeviceClient) ReadEvtDirHex(jobName string) (evts []Event) {
	fs, err := ioutil.ReadDir(fmt.Sprintf("demo/%s/events", jobName))
	if err != nil {
		pkg.TraceErr(err)
		return
	}
	for _, f := range fs {
		i, err := strconv.ParseInt(strings.Split(f.Name(), ".")[0], 10, 64)
		if err != nil {
			pkg.TraceErr(err)
		} else {
			evt := &Event{}
			demo.GetEvtFromFlashHex(jobName, i, evt) // pkg.Json("(demo *DemoDeviceClient) ReadEvtDir( )", evt)
			evts = append(evts, *evt)
		}
	}
	return
}

/* SMP DEMO MEMORY -> 40 BYTES -> HxD 40 x 1 */
func (demo *DemoDeviceClient) WriteSmpToFlash(jobName string, smp Sample) (err error) {

	smpBytes := smp.SampleToBytes()
	// fmt.Printf("\nsmpBytes ( %d ) : %v\n", len(smpBytes), smpBytes)

	dir := fmt.Sprintf("demo/%s", jobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.TraceErr(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/smp.bin", dir), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return pkg.TraceErr(err)
	}
	defer f.Close()

	_, err = f.Write(smpBytes)
	if err != nil {
		return pkg.TraceErr(err)
	}

	f.Close()

	return
}

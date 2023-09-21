package c001v001

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	// "net/url"
	"os"
	"strconv"
	"strings"
	"time"

	phao "github.com/eclipse/paho.mqtt.golang"
	// "github.com/gofiber/contrib/websocket"

	"github.com/leehayford/des/pkg"
)

/* NOT FOR PRODUCTION - SIMULATES A C001V001 DEVICE MQTT DEMO DEVICE 
	PUBLISHES TO ALL SIG TOPICS AS A SINGLE DEVICE AS A SINGLE DEVICE
	SUBSCRIBES TO ALL COMMAND TOPICS AS A SINGLE DEVICE
*/
type Sim struct {
	Qty      int                `json:"qty"`
	Dur      int64              `json:"dur"`
	FillQty  int64              `json:"fill_qty"`
	MTxCh4   DemoModeTransition `json:"mtx_ch4"`
	MTxFlow  DemoModeTransition `json:"mtx_flow"`
	MTxBuild DemoModeTransition `json:"mtx_build"`
}

type DemoModeTransition struct {
	VMin    float32       `json:"v_min"`
	VMax    float32       `json:"v_max"`
	TSpanUp time.Duration `json:"t_span_up"`
	TSpanDn time.Duration `json:"t_span_dn"`
}

type DemoDeviceClient struct {
	Device

	// // ADM     Admin    // RAM Value
	// ADMFile *os.File // Flash

	// // HDR     Header   // RAM Value
	// HDRFile *os.File // Flash

	// // CFG     Config   // RAM Value
	// CFGFile *os.File // Flash

	// // EVT     Event    // RAM Value
	// EVTFile *os.File // Flash

	// // SMP 	Sample // RAM Value
	// SMPFile *os.File // Flash

	Sim
	sizeChan   chan int
	sentChan   chan int
	WSClientID string
	CTX        context.Context
	Cancel     context.CancelFunc
	pkg.DESMQTTClient
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

		DESJobName:  fmt.Sprintf("%s_0000000000000", serial),
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

	adm := (&Job{}).RegisterJob_Default_JobAdmin()
	adm.AdmTime = t
	adm.AdmAddr = des_dev.DESDevRegAddr
	adm.AdmUserID = userID
	adm.AdmApp = des_dev.DESDevRegApp

	hdr := (&Job{}).RegisterJob_Default_JobHeader()
	hdr.HdrTime = t
	hdr.HdrAddr = des_dev.DESDevRegAddr
	hdr.HdrUserID = userID
	hdr.HdrApp = des_dev.DESDevRegApp

	cfg := (&Job{}).RegisterJob_Default_JobConfig()
	cfg.CfgTime = t
	cfg.CfgAddr = des_dev.DESDevRegAddr
	cfg.CfgUserID = userID
	cfg.CfgApp = des_dev.DESDevRegApp

	demo := DemoDeviceClient{
		Device: Device{
			DESRegistration: reg,
			ADM: adm,
			HDR: hdr,
			CFG: cfg,
			EVT: Event{
				EvtTime:   t,
				EvtAddr:  des_dev.DESDevRegAddr,
				EvtUserID: userID,
				EvtApp:   des_dev.DESDevRegApp,

				EvtCode:  STATUS_JOB_ENDED,
				EvtTitle: "Intitial State",
				EvtMsg:   "End Job event to ensure this newly registered demo device is ready to start a new demo job.",
			},
			SMP: Sample{SmpTime: t, SmpJobName: des_job.DESJobName},
			Job: Job{DESRegistration: reg},
		},
	}
	/* CREATE CMDARCHIVE DATABASE */
	pkg.ADB.CreateDatabase( strings.ToLower(demo.Job.DESJobName))

	demo.ConnectJobDBC()
	// fmt.Printf("\nMakeDemoC001V001(): CONNECTED TO DATABASE: %s\n", demo.JobDBC.ConnStr)

	/* CREATE JOB DB TABLES */
	if err := demo.JobDBC.Migrator().CreateTable(
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
		demo.JobDBC.Write(&typ)
	}
	if err := demo.JobDBC.Write(&demo.ADM); err != nil {
		pkg.TraceErr(err)
	}
	if err := demo.JobDBC.Write(&demo.HDR); err != nil {
		pkg.TraceErr(err)
	}
	if err := demo.JobDBC.Write(&demo.CFG); err != nil {
		pkg.TraceErr(err)
	}
	if err := demo.JobDBC.Write(&demo.EVT); err != nil {
		pkg.TraceErr(err)
	}
	if err := demo.JobDBC.Write(&demo.SMP); err != nil {
		pkg.TraceErr(err)
	}

	demo.JobDBC.Disconnect()

	/* WRITE TO FLASH - JOB_0 */
	
	fmt.Printf("\nMakeDemoC001V001(): WRITE TO FLASH: %s/\n", demo.Job.DESJobName)
	demo.WriteAdmToFlash(demo.DESJobName, demo.ADM)
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
		demo.Device.Job = Job{DESRegistration: reg}
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
	demo.JobDBC.Last(&demo.HDR)
	demo.JobDBC.Last(&demo.CFG)
	demo.JobDBC.Last(&demo.EVT)
	demo.JobDBC.Last(&demo.SMP)
	demo.JobDBC.Disconnect() // we don't want to maintain this connection

	if err := demo.MQTTDemoDeviceClient_Connect(); err != nil {
		pkg.TraceErr(err)
	}

	/* ADD TO DemoDeviceClients MAP */
	DemoDeviceClients[demo.DESDevSerial] = *demo
	
	/* RUN THE SIMULATION */
	go demo.Demo_Simulation(time.Now().UTC())
	time.Sleep(time.Second * 1) // WHY?: Just so the console logs show up in the right order when running local dev

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
	delete(DemoDeviceClients, demo.DESDevSerial)
}

func (demo *DemoDeviceClient) MQTTDemoDeviceClient_Connect() (err error) {

	demo.MQTTUser = pkg.MQTT_USER
	demo.MQTTPW = pkg.MQTT_PW
	demo.MQTTClientID = fmt.Sprintf(
		"%s-%s-%s-DEMO",
		demo.DESDevClass,
		demo.DESDevVersion,
		demo.DESDevSerial,
	)
	if err = demo.DESMQTTClient.DESMQTTClient_Connect(false); err != nil {
		return err
	}

	demo.MQTTSubscription_DemoDeviceClient_CMDAdmin().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDHeader().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDConfig().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDEvent().Sub(demo.DESMQTTClient)

	// pkg.MQTTDemoClients[demo.DESDevSerial] = demo.DESMQTTClient
	// demoClient := pkg.MQTTDemoClients[demo.DESDevSerial]
	// fmt.Printf("\n%s client ID: %s\n", demo.DESDevSerial, demoClient.MQTTClientID)

	// DemoDeviceClients[demo.DESDevSerial] = *demo
	// d := c001v001.DemoDeviceClients[demo.DESDevSerial]
	// fmt.Printf("\nCached DemoDeviceClient %s, current event code: %d\n", d.DESDevSerial, d.EVT.EvtCode)
	// fmt.Printf("\n(demo) MQTTDemoDeviceClient_Connect( ) -> ClientID: %s\n", demo.ClientID)

	return err
}
func (demo *DemoDeviceClient) MQTTDemoDeviceClient_Disconnect() {

	/* UNSUBSCRIBE FROM ALL MQTTSubscriptions */
	demo.MQTTSubscription_DemoDeviceClient_CMDAdmin().UnSub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDHeader().UnSub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDConfig().UnSub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDEvent().UnSub(demo.DESMQTTClient)

	/* DISCONNECT THE DESMQTTCLient */
	demo.DESMQTTClient_Disconnect()

	fmt.Printf("\n(device) MQTTDemoDeviceClient_Dicconnect( ): Complete -> ClientID: %s\n", demo.ClientID)
}


/* SUBSCRIPTIONS ****************************************************************************************/

/* SUBSCRIPTION -> ADMINISTRATION -> UPON RECEIPT, LOG & REPLY TO .../sig/admin */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDAdmin() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDAdmin(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* LOAD VALUE INTO SIM 'RAM' */
			if err := json.Unmarshal(msg.Payload(), &demo.ADM); err != nil {
				pkg.TraceErr(err)
				// return
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB 0 */
			demo.WriteAdmToFlash(demo.ZeroJobName(), demo.ADM)
			// adm := demo.ADM

			/* UPDATE SOURCE ONLY */
			demo.ADM.AdmAddr = demo.DESDevSerial

			if demo.EVT.EvtCode > STATUS_JOB_START_REQ {

				// /* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB X */
				// demo.WriteAdmToFlash(demo.Job.DESJobName, adm)

				/* UPDATE TIME ONLY WHEN LOGGING */
				demo.ADM.AdmTime = time.Now().UTC().UnixMilli()

				/* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB X */
				demo.WriteAdmToFlash(demo.Job.DESJobName, demo.ADM)

			}

			// /* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB 0 */
			// demo.WriteAdmToFlash(demo.ZeroJobName(), demo.ADM)

			/* SEND CONFIRMATION */
			demo.MQTTPublication_DemoDeviceClient_SIGAdmin(&demo.ADM)

		},
	}
}

/* SUBSCRIPTIONS -> HEADER -> UPON RECEIPT, LOG & REPLY TO .../sig/header */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDHeader() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDHeader(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* LOAD VALUE INTO SIM 'RAM' */
			if err := json.Unmarshal(msg.Payload(), &demo.HDR); err != nil {
				pkg.TraceErr(err)
				// return
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB 0 */
			demo.WriteHdrToFlash(demo.ZeroJobName(), demo.HDR)
			// hdr := demo.HDR

			/* UPDATE SOURCE ONLY */
			demo.HDR.HdrAddr = demo.DESDevSerial

			if demo.EVT.EvtCode > STATUS_JOB_START_REQ {

				// /* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB X */
				// demo.WriteHdrToFlash(demo.Job.DESJobName, hdr)

				/* UPDATE TIME ONLY WHEN LOGGING */
				demo.HDR.HdrTime = time.Now().UTC().UnixMilli()

				/* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB X */
				demo.WriteHdrToFlash(demo.Job.DESJobName, demo.HDR)
			}

			// /* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB 0 */
			// demo.WriteHdrToFlash(demo.ZeroJobName(), demo.HDR)

			/* SEND CONFIRMATION */
			demo.MQTTPublication_DemoDeviceClient_SIGHeader(&demo.HDR)
		},
	}
}

/* SUBSCRIPTIONS -> CONFIGURATION -> UPON RECEIPT, LOG & REPLY TO .../sig/config */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDConfig() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDConfig(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* LOAD VALUE INTO SIM 'RAM' */
			if err := json.Unmarshal(msg.Payload(), &demo.CFG); err != nil {
				pkg.TraceErr(err)
				// return
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB 0 */
			demo.WriteCfgToFlash(demo.ZeroJobName(), demo.CFG)
			// cfg := demo.CFG

			/* UPDATE SOURCE ONLY */
			demo.CFG.CfgAddr = demo.DESDevSerial

			if demo.EVT.EvtCode > STATUS_JOB_START_REQ {

				// /* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB X */
				// demo.WriteCfgToFlash(demo.Job.DESJobName, cfg)

				/* UPDATE TIME( ONLY WHEN LOGGING) */
				demo.CFG.CfgTime = time.Now().UTC().UnixMilli()

				/* WRITE (AS LOADED) TO SIM 'FLASH' */
				demo.WriteCfgToFlash(demo.Job.DESJobName, demo.CFG)
			}

			// /* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB 0 */
			// demo.WriteCfgToFlash(demo.ZeroJobName(), demo.CFG)

			/* SEND CONFIRMATION */
			demo.MQTTPublication_DemoDeviceClient_SIGConfig(&demo.CFG)
		},
	}
}

/* SUBSCRIPTIONS -> EVENT -> UPON RECEIPT, LOG, HANDLE, & REPLY TO .../sig/event */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDEvent() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDEvent(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* CAPTURE THE ORIGINAL DEVICE STATE EVENT CODE */
			state := demo.EVT.EvtCode

			/* LOAD VALUE INTO SIM 'RAM' */
			if err := json.Unmarshal(msg.Payload(), &demo.EVT); err != nil {
				pkg.TraceErr(err)
				// return
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB 0 */
			demo.WriteEvtToFlash(demo.ZeroJobName(), demo.EVT)
			// evt := demo.EVT

			/* UPDATE SOURCE ONLY */
			demo.EVT.EvtAddr = demo.DESDevSerial

			/* CHECK THE RECEIVED EVENT CODE */
			switch demo.EVT.EvtCode {

			// case 0: 
				/* REGISTRATION EVENT: USED TO ASSIGN THIS DEVICE TO 
				A DIFFERENT DATA EXCHANGE SERVER */
	
			case STATUS_JOB_END_REQ: // End Job
				demo.EndDemoJob()

			case STATUS_JOB_START_REQ: // Start Job
				demo.StartDemoJob(/*state*/)

			default:

				/* CHECK THE ORIGINAL DEVICE STATE EVENT CODE 
				TO SEE IF WE SHOULD WRITE TO THE ACTIVE JOB */
				if state > STATUS_JOB_START_REQ {

					// /* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB X */
					// demo.WriteEvtToFlash(demo.Job.DESJobName, evt)
				
					/* UPDATE TIME( ONLY WHEN LOGGING) */
					demo.EVT.EvtTime = time.Now().UTC().UnixMilli()

					/* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB X */
					demo.WriteEvtToFlash(demo.Job.DESJobName, demo.EVT)
				}

				// /* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB 0 */
				// demo.WriteEvtToFlash(demo.ZeroJobName(), demo.EVT)
	
				/* SEND CONFIRMATION */
				demo.MQTTPublication_DemoDeviceClient_SIGEvent(&demo.EVT)
			}
		},
	}
}


/* PUBLICATIONS ******************************************************************************************/

/* PUBLICATION -> ADMIN -> SIMULATED ADMINS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGAdmin(adm *Admin) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/
	go (pkg.MQTTPublication{

		Topic:   demo.MQTTTopic_SIGAdmin(),
		Message: pkg.MakeMQTTMessage(adm),
		// Message:  pkg.MakeMQTTMessage(adm.FilterAdmRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> HEADER -> SIMULATED HEADERS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGHeader(hdr *Header) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/
	go (pkg.MQTTPublication{

		Topic:   demo.MQTTTopic_SIGHeader(),
		Message: pkg.MakeMQTTMessage(hdr),
		// Message:  pkg.MakeMQTTMessage(hdr.FilterHdrRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> CONFIG -> SIMULATED CONFIGS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGConfig(cfg *Config) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/
	go (pkg.MQTTPublication{

		Topic:   demo.MQTTTopic_SIGConfig(),
		Message: pkg.MakeMQTTMessage(cfg),
		// Message:  pkg.MakeMQTTMessage(cfg.FilterCfgRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> EVENT -> SIMULATED EVENTS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGEvent(evt *Event) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/
	go (pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGEvent(),
		Message:  pkg.MakeMQTTMessage(evt),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> SAMPLE -> SIMULATED SAMPLES */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGSample(mqtts *MQTT_Sample) {

	b64, err := json.Marshal(mqtts)
	if err != nil {
		pkg.TraceErr(err)
	} // pkg.Json("MQTT_Sample:", b64)

	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/
	go (pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGSample(),
		Message:  string(b64),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}


/* SIMULATIONS *******************************************************************************************/

func (demo *DemoDeviceClient) StartDemoJob(/*state*/) {
	fmt.Printf("\n(demo *DemoDeviceClient) StartDemoJob( )...\n")

	/* TODO: MAKE SURE THE PREVIOUS JOB IS ENDED */
	// if state > STATUS_JOB_START_REQ {
	// 	demo.EndDemoJob()
	// }

	/* CAPTURE TIME VALUE FOR JOB INTITALIZATION: DB/JOB NAME, ADM, HDR, CFG, EVT */
	t0 := time.Now().UTC()
	startTime := t0.UnixMilli()

	/* WHERE JOB START ADMIN WAS NOT RECEIVED, USE DEFAULT VALUES */
	if demo.ADM.AdmTime != demo.EVT.EvtTime {
		fmt.Printf("(demo *DemoDeviceClient) StartDemoJob -> USING DEFAULT ADMIN.\n")
		demo.ADM = demo.RegisterJob_Default_JobAdmin()
	}
	demo.ADM.AdmTime = startTime
	demo.ADM.AdmAddr = demo.DESDevSerial
	demo.ADM.AdmUserID = demo.EVT.EvtUserID
	demo.ADM.AdmApp = demo.EVT.EvtApp

	/* WHERE JOB START HEADER WAS NOT RECEIVED, USE DEFAULT VALUES */
	if demo.HDR.HdrTime != demo.EVT.EvtTime {
		fmt.Printf("(demo *DemoDeviceClient) StartDemoJob -> USING DEFAULT HEADER\n")
		demo.HDR = demo.RegisterJob_Default_JobHeader()
	}
	demo.HDR.HdrTime = startTime
	demo.HDR.HdrAddr = demo.DESDevSerial
	demo.HDR.HdrUserID = demo.EVT.EvtUserID
	demo.HDR.HdrApp = demo.EVT.EvtApp
	demo.HDR.HdrJobName = fmt.Sprintf("%s_%d", demo.DESDevSerial, startTime)
	demo.HDR.HdrJobStart = startTime
	demo.HDR.HdrJobEnd = 0
	demo.HDR.HdrGeoLng = -114.75 + rand.Float32()*(-110.15+114.75)
	demo.HDR.HdrGeoLat = 51.85 + rand.Float32()*(54.35-51.85)

	/* WHERE JOB START CONFIG WAS NOT RECEIVED, USE DEFAULT VALUES */
	if demo.CFG.CfgTime != demo.EVT.EvtTime {
		fmt.Printf("(demo *DemoDeviceClient) StartDemoJob -> USING DEFAULT CONFIG.\n")
		demo.CFG = demo.RegisterJob_Default_JobConfig()
	}
	demo.CFG.CfgTime = startTime
	demo.CFG.CfgAddr = demo.DESDevSerial
	demo.CFG.CfgUserID = demo.EVT.EvtUserID
	demo.CFG.CfgApp = demo.EVT.EvtApp
	demo.CFG.CfgVlvTgt = MODE_VENT

	demo.EVT.EvtTime = startTime
	demo.EVT.EvtAddr = demo.DESDevSerial
	demo.EVT.EvtCode = STATUS_JOB_STARTED
	demo.EVT.EvtTitle = "JOB STARTED"
	demo.EVT.EvtMsg = demo.HDR.HdrJobName

	/* TAKE A SAMPLE */
	demo.Demo_Simulation_Take_Sample(t0, time.Now().UTC())

	/* WRITE TO FLASH - JOB 0 */
	demo.WriteAdmToFlash(demo.ZeroJobName(), demo.ADM)
	demo.WriteHdrToFlash(demo.ZeroJobName(), demo.HDR)
	demo.WriteCfgToFlash(demo.ZeroJobName(), demo.CFG)
	demo.WriteEvtToFlash(demo.ZeroJobName(), demo.EVT)
	demo.WriteSmpToFlash(demo.ZeroJobName(), demo.SMP)

	/* WRITE TO FLASH - JOB X */
	demo.Job = Job{
		DESRegistration: pkg.DESRegistration{
			DESDev: demo.DESDev,
			DESJob: pkg.DESJob{
				DESJobRegTime:   startTime,
				DESJobRegAddr:   demo.EVT.EvtAddr,
				DESJobRegUserID: demo.EVT.EvtUserID,
				DESJobRegApp:    demo.EVT.EvtApp,

				DESJobName:  demo.HDR.HdrJobName,
				DESJobStart: startTime,
				DESJobEnd:   0,
				DESJobLng:   demo.HDR.HdrGeoLng,
				DESJobLat:   demo.HDR.HdrGeoLat,
				DESJobDevID: demo.DESDevID,
			},
		},
	}
	demo.WriteAdmToFlash(demo.Job.DESJobName, demo.ADM)
	demo.WriteHdrToFlash(demo.Job.DESJobName, demo.HDR)
	demo.WriteCfgToFlash(demo.Job.DESJobName, demo.CFG)
	demo.WriteEvtToFlash(demo.Job.DESJobName, demo.EVT)
	demo.WriteSmpToFlash(demo.Job.DESJobName, demo.SMP)

	/* SEND CONFIRMATION */
	demo.MQTTPublication_DemoDeviceClient_SIGAdmin(&demo.ADM)
	demo.MQTTPublication_DemoDeviceClient_SIGHeader(&demo.HDR)
	demo.MQTTPublication_DemoDeviceClient_SIGConfig(&demo.CFG)
	smp := Demo_EncodeMQTTSampleMessage(demo.HDR.HdrJobName, 0, demo.SMP)
	demo.MQTTPublication_DemoDeviceClient_SIGSample(&smp)
	demo.MQTTPublication_DemoDeviceClient_SIGEvent(&demo.EVT)

	// time.Sleep(time.Second * 10)
	/* RUN JOB... */
	go demo.Demo_Simulation(t0)
	fmt.Printf("\n(demo *DemoDeviceClient) StartDemoJob( ) -> RUNNING %s...\n", demo.HDR.HdrJobName)
}

func (demo *DemoDeviceClient) EndDemoJob() {
	// fmt.Printf("\n(demo *DemoDeviceClient) EndDemoJob( )...\n")

	/* CAPTURE TIME VALUE FOR JOB TERMINATION: HDR, EVT */
	t0 := time.Now().UTC()
	endTime := t0.UnixMilli()

	// demo.GetHdrFromFlash(demo.ZeroJobName(), &demo.HDR)
	demo.HDR.HdrTime = endTime
	demo.HDR.HdrAddr = demo.DESDevSerial
	demo.HDR.HdrUserID = demo.EVT.EvtUserID
	demo.HDR.HdrApp = demo.EVT.EvtApp
	demo.HDR.HdrJobEnd = endTime

	// demo.EVT = evt
	demo.EVT.EvtTime = endTime
	demo.EVT.EvtAddr = demo.DESDevSerial
	demo.EVT.EvtCode = STATUS_JOB_ENDED
	demo.EVT.EvtTitle = "JOB ENDED"
	demo.EVT.EvtMsg = demo.HDR.HdrJobName

	/* WRITE TO FLASH - JOB 0 */
	demo.WriteEvtToFlash(demo.ZeroJobName(), demo.EVT)
	demo.WriteHdrToFlash(demo.ZeroJobName(), demo.HDR)

	/* WRITE TO FLASH - JOB X */
	demo.WriteEvtToFlash(demo.Job.DESJobName, demo.EVT)
	demo.WriteHdrToFlash(demo.Job.DESJobName, demo.HDR)

	/* SEND CONFIRMATION */
	demo.MQTTPublication_DemoDeviceClient_SIGHeader(&demo.HDR)
	demo.MQTTPublication_DemoDeviceClient_SIGEvent(&demo.EVT)

	fmt.Printf("\n(demo *DemoDeviceClient) EndDemoJob( ) -> ENDED: %s\n", demo.HDR.HdrJobName)
}


/*DEMO SIM -> PUBLISH TO MQTT */
func (demo *DemoDeviceClient) Demo_Simulation(t0 time.Time) {

	for demo.EVT.EvtCode > STATUS_JOB_START_REQ {
		t := time.Now().UTC()
		demo.Demo_Simulation_Take_Sample(t0, t)
		demo.WriteSmpToFlash(demo.Job.DESJobName, demo.SMP)
		smp := Demo_EncodeMQTTSampleMessage(demo.Job.DESJobName, 0, demo.SMP)
		demo.MQTTPublication_DemoDeviceClient_SIGSample(&smp)

		time.Sleep(time.Millisecond * time.Duration(demo.CFG.CfgOpSample))

	}
	// pkg.Json("(demo *DemoDeviceClient) Demo_Simulation( )", demo.EVT)
	fmt.Printf("\n(demo) Demo_Simulation( ) %s waiting for job start...\n", demo.DESDevSerial)
}
func (demo *DemoDeviceClient) Demo_Simulation_Take_Sample(t0, ti time.Time) {

	demo.SMP.SmpTime = ti.UnixMilli()
	demo.SMP.SmpCH4 = Demo_Mode_Transition(t0, ti, time.Duration(time.Second*250), 97.99999, 0.01)
	demo.SMP.SmpHiFlow = Demo_Mode_Transition(t0, ti, time.Duration(time.Second*30), 1.79999, 0.01)
	demo.SMP.SmpLoFlow = Demo_Mode_Transition(t0, ti, time.Duration(time.Second*30), 1.79999, 0.01)
	demo.SMP.SmpPress = Demo_Mode_Transition(t0, ti, time.Duration(time.Second*600), 18.99999, 699.99999)

	demo.SMP.SmpBatAmp = 0.049 + rand.Float32()*0.023
	demo.SMP.SmpBatVolt = 12.733 + rand.Float32()*0.072
	demo.SMP.SmpMotVolt = 11.9 + rand.Float32()*0.033

	demo.SMP.SmpVlvTgt = uint32(demo.CFG.CfgVlvTgt)
	demo.SMP.SmpVlvPos = uint32(demo.CFG.CfgVlvPos)

	demo.SMP.SmpJobName = demo.HDR.HdrJobName

/* TODO:  CHECK ALARMS BASED ON NEW SAMPLE VALUES...
	SEND RESULTING EVENTS / CONFIG CHANGES
	*/
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
func Demo_Mode_Transition(t_start, ti time.Time, t_span time.Duration, v_start, v_end float32) (v float32) {

	// dt := ti.Sub(t_start).Seconds()
	t_rel := float64(ti.Sub(t_start).Seconds() / t_span.Seconds())

	// fmt.Printf("dt: %f, t_span: %v, t_rel: %f\n", dt, t_span.Seconds(), t_rel)
	v_span := float64(v_end - v_start)

	a := v_span * math.Pow(t_rel, 2)

	var bx float64
	if t_rel > 0.5 {
		bx = 0.45
	} else {
		bx = 0.5
	}
	b := 1 - math.Pow((bx-t_rel), 4)
	// fmt.Printf("\nt_rel: %f, a: %f, b: %f\n", t_rel, a, b)

	if b < 0.8 {

		v = v_end
	} else {

		v = v_start + float32(a*b)
	}

	res := float32(v_span) * 0.005
	min := v - res
	v = min + rand.Float32()*res
	// fmt.Printf("%f : %f\n", t_rel, v)
	return
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

/* ADM DEMO MEMORY -> 288 BYTES -> HxD 72 */
func (demo DemoDeviceClient) WriteAdmToFlash(jobName string, adm Admin) (err error) {

	admBytes := adm.AdminToBytes()
	// fmt.Printf("\nadmBytes ( %d ) : %v\n", len(admBytes), admBytes)

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
func (demo *DemoDeviceClient) ReadAdmFromFlash(jobName string) (adm []byte, err error) {

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
	adm = admFile[eof-288 : eof]
	// fmt.Printf("\nadmBytes ( %d ) : %v\n", len(adm), adm)

	f.Close()
	return
}
func (demo *DemoDeviceClient) GetAdmFromFlash(jobName string, adm *Admin) {
	b, err := demo.ReadAdmFromFlash(jobName)
	if err != nil {
		pkg.TraceErr(err)
	}
	adm.AdminFromBytes(b)
}

/* HDR DEMO MEMORY -> 324 BYTES -> HxD 81 */
func (demo *DemoDeviceClient) WriteHdrToFlash(jobName string, hdr Header) (err error) {

	hdrBytes := hdr.HeaderToBytes()
	// fmt.Printf("\nhdrBytes ( %d ) : %v\n", len(hdrBytes), hdrBytes)

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
func (demo *DemoDeviceClient) ReadHdrFromFlash(jobName string) (hdr []byte, err error) {

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
func (demo *DemoDeviceClient) GetHdrFromFlash(jobName string, hdr *Header) {
	b, err := demo.ReadHdrFromFlash(jobName)
	if err != nil {
		pkg.TraceErr(err)
	}
	hdr.HeaderFromBytes(b)
}

/* CFG DEMO MEMORY -> 172 BYTES -> HxD 43  */
func (demo *DemoDeviceClient) WriteCfgToFlash(jobName string, cfg Config) (err error) {

	cfgBytes := cfg.ConfigToBytes()
	// fmt.Printf("\ncfgBytes ( %d ) : %v\n", len(cfgBytes), cfgBytes)

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
func (demo *DemoDeviceClient) ReadCfgFromFlash(jobName string) (cfg []byte, err error) {

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
func (demo *DemoDeviceClient) GetCfgFromFlash(jobName string, cfg *Config) {
	b, err := demo.ReadCfgFromFlash(jobName)
	if err != nil {
		pkg.TraceErr(err)
	}
	cfg.ConfigFromBytes(b)
}

/* EVT DEMO MEMORY -> 156 BYTES + MESSAGE -> HxD 39 */
func (demo *DemoDeviceClient) WriteEvtToFlash(jobName string, evt Event) (err error) {

	evtBytes := evt.EventToBytes()
	// fmt.Printf("\nevtBytes ( %d ) : %v\n", len(evtBytes), evtBytes)

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
func (demo *DemoDeviceClient) ReadEvtFromFlash(jobName string, time int64) (evt []byte, err error) {

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
func (demo *DemoDeviceClient) GetEvtFromFlash(jobName string, time int64, evt *Event) {
	b, err := demo.ReadEvtFromFlash(jobName, time)
	if err != nil {
		pkg.TraceErr(err)
	}
	evt.EventFromBytes(b)
}
func (demo *DemoDeviceClient) ReadEvtDir(jobName string) (evts []Event) {
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
			demo.GetEvtFromFlash(jobName, i, evt) // pkg.Json("(demo *DemoDeviceClient) ReadEvtDir( )", evt)
			evts = append(evts, *evt)
		}
	}
	return
}

func (demo *DemoDeviceClient) WriteSmpToFlash(jobName string, smp Sample) (err error) {

	smpBytes := smp.FilterSmpBytes()
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

/* func (demo DemoDeviceClient) WSDemoDeviceClient_Connect(c *websocket.Conn) {

	simStr, _ := url.QueryUnescape(c.Query("sim"))
	des_regStr, _ := url.QueryUnescape(c.Query("des_reg"))

	des_reg := pkg.DESRegistration{}
	if err := json.Unmarshal([]byte(des_regStr), &des_reg); err != nil {
		pkg.TraceErr(err)
	}
	des_reg.DESDevRegAddr = c.RemoteAddr().String()
	des_reg.DESJobRegAddr = c.RemoteAddr().String()

	sim := Sim{}
	if err := json.Unmarshal([]byte(simStr), &sim); err != nil {
		pkg.TraceErr(err)
	}

	wscid := fmt.Sprintf("%s-DEMO", des_reg.DESDevSerial)

	demo = DemoDeviceClient{
		Device: Device{
			DESRegistration: des_reg,
			Job:             Job{DESRegistration: des_reg},
		},
		Sim:        sim,
		WSClientID: wscid,
	} // fmt.Printf("\nHandleDemo_Run_Sim(...) -> ddc: %v\n\n", demo)

	demo.Device.Job.GetJobData(1)

	demo.sizeChan = make(chan int)
	defer func() {
		close(demo.sizeChan)
		demo.sizeChan = nil
	}()
	demo.sentChan = make(chan int)
	defer func() {
		close(demo.sentChan)
		demo.sentChan = nil
	}()

	// if no demo client ...
	// demo.MQTTDemoDeviceClient_Connect()

	pkg.TraceFunc("Call -> demo.ReadEvtDir(zero)")
	evts := demo.ReadEvtDir(demo.ZeroJobName())
	evt := evts[len(evts)-1]

	if evt.EvtCode < 2 {
		fmt.Printf("\n%s: waiting for job start event...\n", demo.DESDevSerial)
	} else {
		fmt.Printf("\n%s: simulation running...\n", demo.DESDevSerial)
		// go demo.Demo_Simulation(time.Now().UTC())
	}

	open := true
	go func() {
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				fmt.Printf("WSDemoDeviceClient_Connect -> c.ReadMessage() ERROR:\n%s", err.Error())
				break
			}
			if string(msg) == "close" {
				fmt.Printf("WSDemoDeviceClient_Connect -> go func() -> c.ReadMessage(): %s\n", string(msg))
				demo.MQTTDemoDeviceClient_Disconnect()
				open = false
				break
			}
		}
		fmt.Printf("WSDemoDeviceClient_Connect -> go func() done\n")
	}()

	for open {
		select {
		case size := <-demo.sizeChan:
			c.WriteJSON(size)
		case sent := <-demo.sentChan:
			c.WriteJSON(sent)
		}
	}
	return
} */

package c001v001

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	phao "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofiber/contrib/websocket"

	"github.com/leehayford/des/pkg"
)

/*
	MQTT DEVICE CLIENT

PUBLISHES ALL COMMANDS TO A SINGLE DEVICE
SUBSCRIBES TO ALL SIGNALS FOR A SINGLE DEVICE
  - WRITES MESSAGES TO THE JOB DATABASE
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

	// ADM     Admin    // RAM Value
	ADMFile *os.File // Flash

	// HDR     Header   // RAM Value
	HDRFile *os.File // Flash

	// CFG     Config   // RAM Value
	CFGFile *os.File // Flash

	// EVT     Event    // RAM Value
	EVTFile *os.File // Flash

	// SMP 	Sample // RAM Value
	SMPFile *os.File // Flash

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

func (demo DemoDeviceClient) WSDemoDeviceClient_Connect(c *websocket.Conn) {

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

	wscid := fmt.Sprintf("%s-DEMO",	des_reg.DESDevSerial)

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
}

/*
	 NOT FOR PRODUCTION - SIMULATES A C001V001 DEVICE
		MQTT DEMO DEVICE CLIENT

PUBLISHES ALL SIG TOPICS AS A SINGLE DEVICE
SUBSCRIBES TO ALL COMMAND TOPICS AS A SINGLE DEVICE
*/
func (demo *DemoDeviceClient) MQTTDemoDeviceClient_Connect() (err error) {

	demo.MQTTUser = pkg.MQTT_USER
	demo.MQTTPW = pkg.MQTT_PW
	demo.MQTTClientID = fmt.Sprintf(
		"%s-%s-%s-DEMO",
		demo.DESDevClass,
		demo.DESDevVersion,
		demo.DESDevSerial,
	)
	if err = demo.DESMQTTClient.DESMQTTClient_Connect(); err != nil {
		return err
	}

	demo.MQTTSubscription_DemoDeviceClient_CMDAdmin().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDHeader().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDConfig().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDEvent().Sub(demo.DESMQTTClient)

	// pkg.MQTTDemoClients[demo.DESDevSerial] = demo.DESMQTTClient
	// demoClient := pkg.MQTTDemoClients[demo.DESDevSerial]
	// fmt.Printf("\n%s client ID: %s\n", demo.DESDevSerial, demoClient.MQTTClientID)

	DemoDeviceClients[demo.DESDevSerial] = *demo
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

	// delete(pkg.MQTTDemoClients, demo.DESDevSerial)

	fmt.Printf("\n(device) MQTTDemoDeviceClient_Dicconnect( ): Complete -> ClientID: %s\n", demo.ClientID)
}

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

	cfgBytes := cfg.FilterCfgBytes()
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

	_, err = f.Write(cfg.FilterCfgBytes())
	if err != nil {
		pkg.TraceErr(err)
	}

	f.Close()
}

func MakeDemoC001V001(serial, userID string) pkg.DESRegistration {

	t := time.Now().UTC().UnixMilli()
	/* CREATE DEMO DEVICES */
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

	job := &Job{
		DESRegistration: pkg.DESRegistration{
			DESDev: des_dev,
			DESJob: pkg.DESJob{
				DESJobRegTime:   t,
				DESJobRegAddr:   "DEMO",
				DESJobRegUserID: userID,
				DESJobRegApp:    "DEMO",

				DESJobName:  fmt.Sprintf("%s_0000000000000", serial),
				DESJobStart: 0,
				DESJobEnd:   0,
				DESJobLng:   -180, // -114.75 + rand.Float32() * ( -110.15 + 114.75 ),
				DESJobLat:   90,   // 51.85 + rand.Float32() * ( 54.35 - 51.85 ),
				DESJobDevID: des_dev.DESDevID,
			},
		},
	}
	job.Admins = []Admin{(job).RegisterJob_Default_JobAdmin()}
	job.Headers = []Header{(job).RegisterJob_Default_JobHeader()}
	job.Configs = []Config{(job).RegisterJob_Default_JobConfig()}
	job.Events = []Event{(job).RegisterJob_Default_JobEvent()}
	job.Samples = []Sample{{SmpTime: t, SmpJobName: job.DESJobName}}
	job.RegisterJob()

	demo := DemoDeviceClient{
		Device: Device{
			DESRegistration: job.DESRegistration,
			Job:             Job{DESRegistration: job.DESRegistration},
		},
	}

	/* WRITE TO FLASH - JOB_0 */
	demo.WriteAdmToFlash(job.DESJobName, job.Admins[0])
	demo.WriteHdrToFlash(job.DESJobName, job.Headers[0])
	demo.WriteCfgToFlash(job.DESJobName, job.Configs[0])
	demo.WriteEvtToFlash(job.DESJobName, job.Events[0])
	demo.WriteEvtToFlash(job.DESJobName, Event{
		EvtTime:   time.Now().UTC().UnixMilli(),
		EvtAddr:   "DEMO",
		EvtUserID: userID,
		EvtApp:    "DEMO",

		EvtCode:  1,
		EvtTitle: "Intitial State",
		EvtMsg:   "End Job event to ensure this newly registered demo device is ready to start a new demo job.",
	})

	return job.DESRegistration
}


/*
SUBSCRIPTIONS
*/

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

			if demo.EVT.EvtCode > 2 {

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

			if demo.EVT.EvtCode > 2 {

				/* UPDATE TIME ONLY WHEN LOGGING */
				demo.HDR.HdrTime = time.Now().UTC().UnixMilli()

				// /* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB X */
				// demo.WriteHdrToFlash(demo.Job.DESJobName, hdr)

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

			if demo.EVT.EvtCode > 2 {

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

			evt := Event{}
			if err := json.Unmarshal(msg.Payload(), &evt); err != nil {
				pkg.TraceErr(err)
				// return
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB 0 */
			demo.WriteEvtToFlash(demo.ZeroJobName(), evt)
			
			switch evt.EvtCode {

			case 0:
				fmt.Printf("\nRegistration Event: Used to change DES; not implemented...")

			case 1: // End Job
				demo.EndDemoJob(evt)

			case 2: // Start Job
				demo.StartDemoJob(evt)

			// case 10: // Mode Vent
			// case 11: // Mode Build
			// case 12: // Mode Hi Flow
			// case 13: // Mode Lo Flow
			default:	
				/* NOT TESTED */
				state := demo.EVT.EvtCode

				/* UPDATE TIME / SOURCE */
				demo.EVT = evt
				demo.EVT.EvtTime = time.Now().UTC().UnixMilli()
				demo.EVT.EvtAddr = demo.DESDevSerial

				if state > 2 {
					
					// /* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB X */
					// demo.WriteEvtToFlash(demo.Job.DESJobName, evt)
					
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

/*
PUBLICATIONS
*/
/* PUBLICATION -> ADMIN -> SIMULATED ADMINS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGAdmin(adm *Admin) {
	(pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGAdmin(),
		Message:  pkg.MakeMQTTMessage(adm.FilterAdmRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> HEADER -> SIMULATED HEADERS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGHeader(hdr *Header) {
	(pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGHeader(),
		Message:  pkg.MakeMQTTMessage(hdr.FilterHdrRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> CONFIG -> SIMULATED CONFIGS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGConfig(cfg *Config) {
	(pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGConfig(),
		Message:  pkg.MakeMQTTMessage(cfg.FilterCfgRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> EVENT -> SIMULATED EVENTS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGEvent(evt *Event) {
	(pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGEvent(),
		Message:  pkg.MakeMQTTMessage(evt.FilterEvtRecord()),
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
	(pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGSample(),
		Message:  string(b64),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}

/*DEMO SIM -> PUBLISH TO MQTT */
func (demo *DemoDeviceClient) Demo_Simulation(t0 time.Time) {

	// i := 0
	for demo.EVT.EvtCode > 1 {
		t := time.Now().UTC()
		demo.Demo_Simulation_Take_Sample(t0, t)
		demo.WriteSmpToFlash(demo.Job.DESJobName, demo.SMP)
		smp := Demo_EncodeMQTTSampleMessage(demo.Job.DESJobName, 0, demo.SMP)
		demo.MQTTPublication_DemoDeviceClient_SIGSample(&smp)

		time.Sleep(time.Millisecond * time.Duration(demo.CFG.CfgOpSample))
		// if i % 3 == 0 {
		// 	demo.CFG.CfgVlvTgt = 0
		// }
		// if i % 5 == 0 {
		// 	demo.CFG.CfgVlvTgt = 4
		// }
		// if i % 7 == 0 {
		// 	demo.CFG.CfgVlvTgt = 6
		// }
		// i++
		// fmt.Printf("(demo *DemoDeviceClient) Demo_Simulation( ): %s -> %d\n", demo.DESDevSerial, t.UnixMilli())
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

// func Demo_Take_Sim_Sample(t0, ti time.Time, job string) MQTT_Sample {

// 	tumic := ti.UnixMilli()
// 	data := []pkg.TimeSeriesData{
// 		/* "AAABgss3rYBCxs2nO2VgQj6qrwk/JpeNPv6JZUFWw+1BUWVuAAQABA==" */
// 		{ // methane
// 			Data: []pkg.TSDPoint{{
// 				X: tumic,
// 				Y: Demo_Mode_Transition(t0, ti, time.Duration(time.Second*250), 97.99999, 0.01),
// 			}},
// 			Min: 0,
// 			Max: 100,
// 		},
// 		{ // high_flow
// 			Data: []pkg.TSDPoint{{
// 				X: tumic,
// 				Y: Demo_Mode_Transition(t0, ti, time.Duration(time.Second*30), 1.79999, 0.01),
// 			}},
// 			Min: 0,
// 			Max: 250,
// 		},
// 		{ // low_flow
// 			Data: []pkg.TSDPoint{{
// 				X: tumic,
// 				Y: Demo_Mode_Transition(t0, ti, time.Duration(time.Second*30), 1.79999, 0.01),
// 			}},
// 			Min: 0,
// 			Max: 2,
// 		},
// 		{ // pressure
// 			Data: []pkg.TSDPoint{{
// 				X: tumic,
// 				Y: Demo_Mode_Transition(t0, ti, time.Duration(time.Second*600), 18.99999, 699.99999),
// 			}},
// 			Min: 0,
// 			Max: 1500,
// 		},
// 		{ // battery_current
// 			// Data: []pkg.TSDPoint{{X: tumic, Y: YSinX(t0, ti, 0.249, 0.09)}},
// 			Data: []pkg.TSDPoint{{X: tumic, Y: 0.049 + rand.Float32()*0.023}},
// 			Min:  0,
// 			Max:  1.5,
// 		},
// 		{ // battery_voltage
// 			// Data: []pkg.TSDPoint{{X: tumic, Y: YCosX(t0, ti, 13.9, 0.8)}},
// 			Data: []pkg.TSDPoint{{X: tumic, Y: 12.733 + rand.Float32()*0.072}},
// 			Min:  0,
// 			Max:  15,
// 		},
// 		{ // motor_voltage
// 			// Data: []pkg.TSDPoint{{X: tumic, Y: YSinX(t0, ti, 12.9, 0.9)}},
// 			Data: []pkg.TSDPoint{{X: tumic, Y: 11.9 + rand.Float32()*0.033}},
// 			Min:  0,
// 			Max:  15,
// 		},
// 		{ // valve_target
// 			Data: []pkg.TSDPoint{{X: tumic, Y: 0}},
// 			Min:  0,
// 			Max:  10,
// 		},
// 		{ // valve_position
// 			Data: []pkg.TSDPoint{{X: tumic, Y: 0}},
// 			Min:  0,
// 			Max:  10,
// 		},
// 	}

// 	/*
// 	TODO: CHECK ALARMS BASED ON NEW SAMPLE VALUES...
// 	*/

//		// return Demo_EncodeMQTTSampleMessage(job, 0, data)
//		return
//	}
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

/* ADM DEMO MEMORY */
func (demo *DemoDeviceClient) WriteAdmToFlash(jobName string, adm Admin) (err error) {

	admBytes := adm.FilterAdmBytes()
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
	adm.MakeAdmFromBytes(b)
}

/* HDR DEMO MEMORY */
func (demo *DemoDeviceClient) WriteHdrToFlash(jobName string, hdr Header) (err error) {

	hdrBytes := hdr.FilterHdrBytes()
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
	hdr.MakeHdrFromBytes(b)
}

/* CFG DEMO MEMORY */
func (demo *DemoDeviceClient) WriteCfgToFlash(jobName string, cfg Config) (err error) {

	cfgBytes := cfg.FilterCfgBytes()
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
	cfg.MakeCfgFromBytes(b)
}

/* EVT DEMO MEMORY */
func (demo *DemoDeviceClient) WriteEvtToFlash(jobName string, evt Event) (err error) {

	evtBytes := evt.FilterEvtBytes()
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
	evt.MakeEvtFromBytes(b)
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

func (demo *DemoDeviceClient) StartDemoJob(evt Event) {
	fmt.Printf("\n(demo *DemoDeviceClient) StartDemoJob( )...\n")

	// /* MAKE SURE THE PREVIOUS JOB IS ENDED */
	// evts := demo.ReadEvtDir(demo.ZeroJobName())
	// if evts[len(evts)-1].EvtCode > 1 {
	// 	demo.EndDemoJob(Event{
	// 		EvtTime:   evt.EvtTime,
	// 		EvtAddr:   evt.EvtAddr,
	// 		EvtUserID: evt.EvtUserID,
	// 		EvtApp:    evt.EvtApp,

	// 		EvtCode:  1, // JOB END
	// 		EvtTitle: "Unexpected Job Start Request",
	// 		EvtMsg:   "Ending current job to start a new job.",
	// 	})
	// }

	/* CAPTURE TIME VALUE FOR JOB INTITALIZATION: DB/JOB NAME, ADM, HDR, CFG, EVT */
	t0 := time.Now().UTC()
	startTime := t0.UnixMilli()


	// Get last ADM from Job_0 -> if time doesn't match JOB START event time, use default Admin settings
	// demo.GetAdmFromFlash(demo.ZeroJobName(), &demo.ADM)
	if demo.ADM.AdmTime != evt.EvtTime {
		fmt.Printf("(demo *DemoDeviceClient) StartDemoJob -> USING DEFAULT ADMIN.\n")
		demo.ADM = demo.RegisterJob_Default_JobAdmin()
	}
	demo.ADM.AdmTime = startTime
	demo.ADM.AdmAddr = demo.DESDevSerial
	demo.ADM.AdmUserID = evt.EvtUserID
	demo.ADM.AdmApp = evt.EvtApp

	// Get last HDR from Job_0 -> if time doesn't match JOB START event time, use default Header settings
	// demo.GetHdrFromFlash(demo.ZeroJobName(), &demo.HDR)
	if demo.HDR.HdrTime != evt.EvtTime {
		fmt.Printf("(demo *DemoDeviceClient) StartDemoJob -> USING DEFAULT HEADER\n")
		demo.HDR = demo.RegisterJob_Default_JobHeader()
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

	// Get last CFG from Job_0 -> if time doesn't match JOB START event time, use default Config settings
	// demo.GetCfgFromFlash(demo.ZeroJobName(), &demo.CFG)
	if demo.CFG.CfgTime != evt.EvtTime {
		fmt.Printf("(demo *DemoDeviceClient) StartDemoJob -> USING DEFAULT CONFIG.\n")
		demo.CFG = demo.RegisterJob_Default_JobConfig()
	}
	demo.CFG.CfgTime = startTime
	demo.CFG.CfgAddr = demo.DESDevSerial
	demo.CFG.CfgUserID = evt.EvtUserID
	demo.CFG.CfgApp = evt.EvtApp
	demo.CFG.CfgVlvTgt = MODE_VENT

	// update Event Time
	demo.EVT = evt
	demo.EVT.EvtTime = startTime
	demo.EVT.EvtAddr = demo.DESDevSerial
	demo.EVT.EvtTitle = "JOB STARTED"
	demo.EVT.EvtMsg = demo.HDR.HdrJobName

	demo.Demo_Simulation_Take_Sample(t0, time.Now().UTC())
	// pkg.Json("(demo *DemoDeviceClient) StartDemoJob( ) -> Initial Sample ", demo.SMP)

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
	demo.WriteSmpToFlash(demo.Job.DESJobName, demo.SMP)
	demo.WriteEvtToFlash(demo.Job.DESJobName, demo.EVT)

	/* SEND CONFIRMATION */
	demo.MQTTPublication_DemoDeviceClient_SIGAdmin(&demo.ADM)
	demo.MQTTPublication_DemoDeviceClient_SIGHeader(&demo.HDR)
	demo.MQTTPublication_DemoDeviceClient_SIGConfig(&demo.CFG)
	smp := Demo_EncodeMQTTSampleMessage(demo.HDR.HdrJobName, 0, demo.SMP)
	demo.MQTTPublication_DemoDeviceClient_SIGSample(&smp)
	demo.MQTTPublication_DemoDeviceClient_SIGEvent(&demo.EVT)

	/* RUN JOB... */
	go demo.Demo_Simulation(t0)
	// pkg.Json("(demo *DemoDeviceClient) StartDemoJob( ) -> demo.EVT:", demo.EVT)
	fmt.Printf("\n(demo *DemoDeviceClient) StartDemoJob( ) -> RUNNING %s...\n", demo.HDR.HdrJobName)
}

func (demo *DemoDeviceClient) EndDemoJob(evt Event) {
	// fmt.Printf("\n(demo *DemoDeviceClient) EndDemoJob( )...\n")

	/* CAPTURE TIME VALUE FOR JOB TERMINATION: HDR, EVT */
	t0 := time.Now().UTC()
	endTime := t0.UnixMilli()


	// demo.GetHdrFromFlash(demo.ZeroJobName(), &demo.HDR)
	demo.HDR.HdrTime = endTime
	demo.HDR.HdrAddr = demo.DESDevSerial
	demo.HDR.HdrUserID = evt.EvtUserID
	demo.HDR.HdrApp = evt.EvtApp
	demo.HDR.HdrJobEnd = endTime

	demo.EVT = evt
	demo.EVT.EvtTime = endTime
	demo.EVT.EvtAddr = demo.DESDevSerial
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

	// pkg.Json("(demo *DemoDeviceClient) EndDemoJob( ) -> demo.EVT:", demo.EVT)
	fmt.Printf("\n(demo *DemoDeviceClient) EndDemoJob( ) -> ENDED: %s\n", demo.HDR.HdrJobName)
	
	// demo.HDR = demo.Job.RegisterJob_Default_JobHeader()
	// demo.MQTTPublication_DemoDeviceClient_SIGHeader(&demo.HDR)

}


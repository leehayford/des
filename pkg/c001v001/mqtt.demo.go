package c001v001

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"

	"sync"

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
	MTxCh4    DemoModeTransition `json:"mtx_ch4"`
	MTxHiFlow DemoModeTransition `json:"mtx_hi_flow"`
	MTxLoFlow DemoModeTransition `json:"mtx_lo_flow"`
	MTxBuild  DemoModeTransition `json:"mtx_build"`
	pkg.DESMQTTClient
	Stop  chan struct{}
	Rate  chan int32
	Mode  chan int32
	TZero chan time.Time
	GPS   chan bool
	Live  bool
}

type DemoDeviceClientsMap map[string]DemoDeviceClient

var DemoDeviceClients = make(DemoDeviceClientsMap)
var DemoDeviceClientsRWMutex = sync.RWMutex{}

func DemoDeviceClientsMapWrite(serial string, d DemoDeviceClient) {
	DemoDeviceClientsRWMutex.Lock()
	DemoDeviceClients[serial] = d
	DemoDeviceClientsRWMutex.Unlock()
}
func DemoDeviceClientsMapReadAll() (demos DemoDeviceClientsMap) {
	DemoDeviceClientsRWMutex.Lock()
	demos = DemoDeviceClients
	DemoDeviceClientsRWMutex.Unlock()
	return
}
func DemoDeviceClientsMapRead(serial string, d DemoDeviceClient) {
	DemoDeviceClientsRWMutex.Lock()
	d = DemoDeviceClients[serial]
	DemoDeviceClientsRWMutex.Unlock()
}
func DemoDeviceClientsMapRemove(serial string) {
	DemoDeviceClientsRWMutex.Lock()
	delete(DemoDeviceClients, serial)
	DemoDeviceClientsRWMutex.Unlock()
	fmt.Printf("\n\nRemoveFromDemoDeviceClientsMap( %s ) Removed... \n", serial)
}

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

/* CALLED ON SERVER STARTUP */
func DemoDeviceClient_ConnectAll() {

	regs, err := GetDemoDeviceList()
	if err != nil {
		pkg.LogErr(err)
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
	demos := DemoDeviceClientsMapReadAll()
	for _, d := range demos {
		d.DemoDeviceClient_Disconnect()
	}
}

func (demo *DemoDeviceClient) DemoDeviceClient_Connect() (err error) {

	fmt.Printf("\n\n(*DemoDeviceClient) DemoDeviceClient_Connect() -> %s -> connecting... \n", demo.DESDevSerial)

	dir := demo.CmdArchiveName()
	demo.ReadLastADMFromJSONFile(dir)
	demo.ReadLastSTAFromJSONFile(dir)
	demo.ReadLastHDRFromJSONFile(dir)
	demo.ReadLastCFGFromJSONFile(dir)
	demo.ReadLastEVTFromJSONFile(dir)

	/* ENSURE DEMO DEVICE HAS CORRECT ID VALUES
	DEVICE USER ID IS USED WHEN CREATING AUTOMATED / ALARM Event OR Config STRUCTS
	- WE DON'T WANT TO ATTRIBUTE THEM TO ANOTHER USER */
	if err = demo.GetDeviceDESU(); err != nil {
		return pkg.LogErr(err)
	}
	demo.STA.StaAddr = demo.DESDevSerial
	demo.STA.StaUserID = demo.DESU.ID.String()
	demo.STA.StaApp = demo.DESDevSerial

	// demo.EVT.WG = &sync.WaitGroup{}
	demo.Stop = make(chan struct{})
	demo.Rate = make(chan int32)
	demo.Mode = make(chan int32)
	demo.TZero = make(chan time.Time)
	demo.GPS = make(chan bool)
	demo.Live = true

	if err := demo.MQTTDemoDeviceClient_Connect(); err != nil {
		demo.Live = false
		return pkg.LogErr(err)
	}

	/* ADD TO DemoDeviceClients MAP */
	DemoDeviceClientsMapWrite(demo.DESDevSerial, *demo)

	/* RUN THE SIMULATION IF LAST KNOWN STATUS WAS LOGGING */
	if demo.STA.StaLogging == OP_CODE_JOB_STARTED {
		go demo.Demo_Simulation(demo.STA.StaJobName, demo.CFG.CfgVlvTgt, demo.CFG.CfgOpSample)
	}

	gps := false
	go func() {
		for demo.Live {
			select {

			case gps = <-demo.GPS: /* IF GPS IS ON, LTE IS OFF */

			default:
				if !gps {
					demo.MQTTPublication_DemoDeviceClient_SIGPing()
					time.Sleep(time.Millisecond * DEVICE_PING_TIMEOUT)
				}
			}
		}
	}()

	fmt.Printf("\n(*DemoDeviceClient) DemoDeviceClient_Connect() -> %s -> connected... \n\n", demo.DESDevSerial)
	return
}
func (demo *DemoDeviceClient) DemoDeviceClient_Disconnect() {
	/* TODO: TEST WHEN IMPLEMENTING
	- UNREGISTER DEVICE
	- GRACEFUL SHUTDOWN
	*/
	fmt.Printf("\n\n(*DemoDeviceClient) DemoDeviceClient_Disconnect() -> %s -> disconnecting... \n", demo.DESDevSerial)

	if err := demo.MQTTDeviceClient_Disconnect(); err != nil {
		pkg.LogErr(err)
	}

	close(demo.GPS)
	demo.GPS = nil

	close(demo.Stop)
	demo.Stop = nil

	close(demo.Rate)
	demo.Rate = nil

	close(demo.Mode)
	demo.Mode = nil

	close(demo.TZero)
	demo.TZero = nil

	demo.Live = false

	/* REMOVE FROM DemoDeviceClients MAP */
	DemoDeviceClientsMapRemove(demo.DESDevSerial)
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
	demo.MQTTSubscription_DemoDeviceClient_CMDStartJob().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDEndJob().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDReport().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDAdmin().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDState().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDHeader().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDConfig().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDEvent().Sub(demo.DESMQTTClient)

	/* MESSAGE LIMIT TEST ***TODO: REMOVE AFTER DEVELOPMENT*** */
	demo.MQTTSubscription_DemoDeviceClient_CMDMsgLimit().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDTestOLS().Sub(demo.DESMQTTClient)

	return err
}
func (demo *DemoDeviceClient) MQTTDemoDeviceClient_Disconnect() (err error) {

	if demo.DESMQTTClient.Client != nil {
		/* UNSUBSCRIBE FROM ALL MQTTSubscriptions */
		demo.MQTTSubscription_DemoDeviceClient_CMDStartJob().UnSub(demo.DESMQTTClient)
		demo.MQTTSubscription_DemoDeviceClient_CMDEndJob().UnSub(demo.DESMQTTClient)
		demo.MQTTSubscription_DemoDeviceClient_CMDReport().UnSub(demo.DESMQTTClient)
		demo.MQTTSubscription_DemoDeviceClient_CMDAdmin().UnSub(demo.DESMQTTClient)
		demo.MQTTSubscription_DemoDeviceClient_CMDState().UnSub(demo.DESMQTTClient)
		demo.MQTTSubscription_DemoDeviceClient_CMDHeader().UnSub(demo.DESMQTTClient)
		demo.MQTTSubscription_DemoDeviceClient_CMDConfig().UnSub(demo.DESMQTTClient)
		demo.MQTTSubscription_DemoDeviceClient_CMDEvent().UnSub(demo.DESMQTTClient)

		/* MESSAGE LIMIT TEST ***TODO: REMOVE AFTER DEVELOPMENT*** */
		demo.MQTTSubscription_DemoDeviceClient_CMDMsgLimit().UnSub(demo.DESMQTTClient)
		demo.MQTTSubscription_DemoDeviceClient_CMDTestOLS().UnSub(demo.DESMQTTClient)
	}
	/* DISCONNECT THE DESMQTTCLient */
	demo.DESMQTTClient_Disconnect()

	fmt.Printf("\n(*DemoDeviceClient) MQTTDemoDeviceClient_Dicconnect( ) -> %s -> disconnected.\n", demo.ClientID)
	return
}

/* SUBSCRIPTIONS ****************************************************************************************/

/* SUBSCRIPTION -> START JOB -> UPON RECEIPT, LOG, START JOB & REPLY TO .../sig/start */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDStartJob() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDStartJob(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			start := StartJob{}
			if err := json.Unmarshal(msg.Payload(), &start); err != nil {
				pkg.LogErr(err)
			}

			demo.StartDemoJob(start, false)
		},
	}
}

/* SUBSCRIPTION -> END JOB -> UPON RECEIPT, LOG, END JOB & REPLY TO .../sig/end */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDEndJob() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDEndJob(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			evt := Event{}
			if err := json.Unmarshal(msg.Payload(), &evt); err != nil {
				pkg.LogErr(err)
			}

			demo.EndDemoJob(evt)
		},
	}
}

/*
	SUBSCRIPTION -> REPORT ALL MODELS -> UPON RECEIPT REPLY TO EACH SIG TOPIC WITH THE CORRESPONDING MODEL

- Admin .../sig/admin
- State .../sig/State
- Header .../sig/header
- Config .../sig/config
- Event .../sig/event
*/
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDReport() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDReport(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* MAKE A COPY OF EACH MODEL - AS IS, NO MODIFICATION */
			adm := demo.ADM
			sta := demo.STA
			hdr := demo.HDR
			cfg := demo.CFG
			evt := demo.EVT

			fmt.Printf("\n%s Publishing Report.\n", demo.DESDevSerial)
			/* PUBLISH EACH LOCAL MODEL IN A GO ROUTINE  */
			go demo.MQTTPublication_DemoDeviceClient_SIGAdmin(adm)
			go demo.MQTTPublication_DemoDeviceClient_SIGState(sta)
			go demo.MQTTPublication_DemoDeviceClient_SIGHeader(hdr)
			go demo.MQTTPublication_DemoDeviceClient_SIGConfig(cfg)
			go demo.MQTTPublication_DemoDeviceClient_SIGEvent(evt)
		},
	}
}

/* SUBSCRIPTION -> ADMIN -> UPON RECEIPT, LOG & REPLY TO .../sig/admin */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDAdmin() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDAdmin(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			adm := Admin{}
			if err := json.Unmarshal(msg.Payload(), &adm); err != nil {
				pkg.LogErr(err)
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> CMDARCHIVE */
			adm_rec := adm
			demo.WriteAdmToJSONFile(demo.CmdArchiveName(), adm_rec)

			/* UPDATE SOURCE ADDRESS AND TIME */
			adm.AdmAddr = demo.DESDevSerial
			adm.AdmTime = time.Now().UTC().UnixMilli()

			if demo.STA.StaLogging > OP_CODE_JOB_START_REQ {

				/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB */
				demo.WriteAdmToJSONFile(demo.DESJobName, adm_rec)

				/* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB */
				demo.WriteAdmToJSONFile(demo.DESJobName, adm)
			}

			/* WRITE (AS LOADED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteAdmToJSONFile(demo.CmdArchiveName(), adm)

			/* LOAD VALUE INTO SIM 'RAM' */
			demo.ADM = adm

			/* SEND CONFIRMATION */
			go demo.MQTTPublication_DemoDeviceClient_SIGAdmin(adm)
		},
	}
}

/* SUBSCRIPTION -> STATE -> UPON RECEIPT, LOG & REPLY TO .../sig/state */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDState() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDState(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			sta := State{}
			if err := json.Unmarshal(msg.Payload(), &sta); err != nil {
				pkg.LogErr(err)
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> CMDARCHIVE */
			sta_rec := sta
			demo.WriteStateToJSONFile(demo.CmdArchiveName(), sta_rec)

			/* TEMPORARY OUT: UNCOMMENT THIS WHEN STATE WRITES ARE ENABLED
			// UPDATE SOURCE ADDRESS ONLY
			sta.StaAddr = demo.DESDevSerial
			**********************************************************************************/

			/* TEMPORARY USE:
			COMMENT THIS OUT WHEN STATE WRITES ARE ENABLED **********/
			sta = demo.STA
			sta.StaTime = time.Now().UTC().UnixMilli()
			/* END TEMPORARY USE ********************************************************************/

			if demo.STA.StaLogging > OP_CODE_JOB_START_REQ {

				/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB */
				demo.WriteStateToJSONFile(demo.DESJobName, sta_rec)

				/* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB */
				demo.WriteStateToJSONFile(demo.DESJobName, sta)
			}

			/* WRITE (AS LOADED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteStateToJSONFile(demo.CmdArchiveName(), sta)

			/* TEMPORARY OUT : UNCOMMENT THIS WHEN STATE WRITES ARE ENABLED
			// LOAD VALUE INTO SIM 'RAM'
			demo.STA = sta
			**********************************************************************************/

			/* SEND CONFIRMATION */
			go demo.MQTTPublication_DemoDeviceClient_SIGState(sta)
		},
	}
}

/* SUBSCRIPTIONS -> HEADER -> UPON RECEIPT, LOG & REPLY TO .../sig/header */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDHeader() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDHeader(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			hdr := Header{}
			if err := json.Unmarshal(msg.Payload(), &hdr); err != nil {
				pkg.LogErr(err)
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> CMDARCHIVE */
			hdr_rec := hdr
			demo.WriteHdrToJSONFile(demo.CmdArchiveName(), hdr_rec)

			/* UPDATE SOURCE ADDRESS AND TIME */
			hdr.HdrAddr = demo.DESDevSerial
			hdr.HdrTime = time.Now().UTC().UnixMilli()

			if demo.STA.StaLogging > OP_CODE_JOB_START_REQ {

				/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB */
				demo.WriteHdrToJSONFile(demo.DESJobName, hdr_rec)

				/* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB */
				demo.WriteHdrToJSONFile(demo.DESJobName, hdr)
			}

			/* WRITE (AS LOADED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteHdrToJSONFile(demo.CmdArchiveName(), hdr)

			/* LOAD VALUE INTO SIM 'RAM' */
			demo.HDR = hdr

			/* SEND CONFIRMATION */
			go demo.MQTTPublication_DemoDeviceClient_SIGHeader(hdr)
		},
	}
}

/* SUBSCRIPTIONS -> CONFIGURATION -> UPON RECEIPT, LOG & REPLY TO .../sig/config */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDConfig() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDConfig(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* CAPTURE EXISTING CFG */
			exCFG := demo.CFG

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			cfg := Config{}
			if err := json.Unmarshal(msg.Payload(), &cfg); err != nil {
				pkg.LogErr(err)
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> CMDARCHIVE */
			cfg_rec := cfg
			demo.WriteCfgToJSONFile(demo.CmdArchiveName(), cfg_rec)

			/* UPDATE SOURCE ADDRESS AND TIME */
			cfg.CfgAddr = demo.DESDevSerial
			cfg.CfgTime = time.Now().UTC().UnixMilli()

			if demo.STA.StaLogging > OP_CODE_JOB_START_REQ {

				/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB */
				demo.WriteCfgToJSONFile(demo.DESJobName, cfg_rec)

				/* IF VALVE TARGET HAS CHANGED, START A NEW MODE TRANSITION */
				if exCFG.CfgVlvTgt != cfg.CfgVlvTgt {
					demo.Mode <- cfg.CfgVlvTgt
					demo.TZero <- time.Now().UTC()
				}

				/* IF SAMPLE DATE HAS CHANGED, SEND UPDATE THE SIMULATION */
				if exCFG.CfgOpSample != cfg.CfgOpSample {
					demo.Rate <- cfg.CfgOpSample
				}

				/* WRITE (AS LOADED) TO SIM 'FLASH' */
				demo.WriteCfgToJSONFile(demo.DESJobName, cfg)
			}

			/* WRITE (AS LOADED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteCfgToJSONFile(demo.CmdArchiveName(), cfg)

			/* LOAD VALUE INTO SIM 'RAM' */
			demo.CFG = cfg

			/* SEND CONFIRMATION */
			go demo.MQTTPublication_DemoDeviceClient_SIGConfig(cfg)
		},
	}
}

/* SUBSCRIPTIONS -> EVENT -> UPON RECEIPT, LOG, HANDLE, & REPLY TO .../sig/event */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDEvent() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDEvent(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE EVENT IN CMDARCHIVE */
			evt := Event{}
			if err := json.Unmarshal(msg.Payload(), &evt); err != nil {
				pkg.LogErr(err)
			}

			/* WRITE (AS REVEICED) TO SIM 'FLASH' -> CMDARCHIVE */
			evt_rec := evt
			demo.WriteEvtToJSONFile(demo.CmdArchiveName(), evt_rec)

			/* UPDATE SOURCE ADDRESS AND TIME */
			evt.EvtAddr = demo.DESDevSerial
			evt.EvtTime = time.Now().UTC().UnixMilli()

			if demo.STA.StaLogging > OP_CODE_JOB_START_REQ {

				/* WRITE (AS REVEICED) TO SIM 'FLASH' -> JOB */
				demo.WriteEvtToJSONFile(demo.DESJobName, evt_rec)

				/* WRITE (AS LOADED) TO SIM 'FLASH' -> JOB */
				demo.WriteEvtToJSONFile(demo.DESJobName, evt)
			}

			/* WRITE (AS LOADED) TO SIM 'FLASH' -> CMDARCHIVE */
			demo.WriteEvtToJSONFile(demo.CmdArchiveName(), evt)

			/* LOAD VALUE INTO SIM 'RAM' */
			demo.EVT = evt

			/* SEND CONFIRMATION */
			go demo.MQTTPublication_DemoDeviceClient_SIGEvent(evt)
		},
	}
}

/* SUBSCRIPTIONS -> MESSAGE LIMIT TEST ***TODO: REMOVE AFTER DEVELOPMENT*** */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDMsgLimit() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDMsgLimit(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE MsgLimit IN CMDARCHIVE */
			kafka := MsgLimit{}
			if err := json.Unmarshal(msg.Payload(), &kafka); err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DemoDeviceClient_CMDMsgLimit(): -> kafka", kafka)

			/* SEND CONFIRMATION */
			go demo.MQTTPublication_DemoDeviceClient_SIGMsgLimit(kafka)

		},
	}
}

/* SUBSCRIPTIONS -> OFFLINE JOB START TEST ***TODO: REMOVE AFTER DEVELOPMENT*** */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDTestOLS() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDTestOLS(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* OFFLINE JOB START */
			demo.SimOfflineStart()

		},
	}
}

/* PUBLICATIONS ******************************************************************************************/

/* MQTTPublication_DemoDeviceClient_SIGStartJob */

/* PUBLICATION -> START JOB -> SIMULATED JOB STARTED RESPONSE */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGStartJob(start StartJob) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/

	json, err := pkg.ModelToJSONString(start)
	if err != nil {
		pkg.LogErr(err)
	}

	sig := pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGStartJob(),
		Message:  json,
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	sig.Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> END JOB -> SIMULATED JOB STARTED RESPONSE */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGEndJob(sta State) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/

	json, err := pkg.ModelToJSONString(sta)
	if err != nil {
		pkg.LogErr(err)
	}

	sig := pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGEndJob(),
		Message:  json,
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	sig.Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> PING -> SIMULATED ADMINS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGPing() {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/

	json, err := pkg.ModelToJSONString(pkg.Ping{Time: time.Now().UTC().UnixMilli(), OK: true})
	if err != nil {
		pkg.LogErr(err)
	}

	sig := pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGDevicePing(),
		Message:  json,
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	sig.Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> ADMIN -> SIMULATED ADMINS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGAdmin(adm Admin) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/

	json, err := pkg.ModelToJSONString(adm)
	if err != nil {
		pkg.LogErr(err)
	}

	sig := pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGAdmin(),
		Message:  json,
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	sig.Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> STATE  -> SIMULATED STATE */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGState(sta State) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/

	json, err := pkg.ModelToJSONString(sta)
	if err != nil {
		pkg.LogErr(err)
	}

	sig := pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGState(),
		Message:  json,
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

	json, err := pkg.ModelToJSONString(hdr)
	if err != nil {
		pkg.LogErr(err)
	}

	sig := pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGHeader(),
		Message:  json,
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

	json, err := pkg.ModelToJSONString(cfg)
	if err != nil {
		pkg.LogErr(err)
	}

	sig := pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGConfig(),
		Message:  json,
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

	json, err := pkg.ModelToJSONString(evt)
	if err != nil {
		pkg.LogErr(err)
	}

	sig := pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGEvent(),
		Message:  json,
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
		pkg.LogErr(err)
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

/* PUBLICATION -> MESSAGE LIMIT TEST ***TODO: REMOVE AFTER DEVELOPMENT***  */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGMsgLimit(msg MsgLimit) {
	/* RUN IN A GO ROUTINE (SEPARATE THREAD) TO
	PREVENT BLOCKING WHEN PUBLISH IS CALLED IN A MESSAGE HANDLER
	*/

	json, err := pkg.ModelToJSONString(msg)
	if err != nil {
		pkg.LogErr(err)
	}

	sig := pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGMsgLimit(),
		Message:  json,
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	sig.Pub(demo.DESMQTTClient)
}

/* SIMULATIONS *******************************************************************************************/

func (demo *DemoDeviceClient) StartDemoJob(start StartJob, offline bool) {
	fmt.Printf("\n(*DemoDeviceClient) StartDemoJob( %s )...\n", demo.DESDevSerial)

	/* TELL USERS WE ARE SWITCHING FROM LTE TO GPS */
	evt := start.EVT
	evt.EvtAddr = demo.DESDevSerial
	evt.EvtCode = OP_CODE_GPS_ACQ
	evt.EvtTitle = GetEventTypeByCode(evt.EvtCode)
	go demo.MQTTPublication_DemoDeviceClient_SIGEvent(evt)

	/* DISCONNECT LTE AND SIMULATE GPS AQUISITION */
	demo.GPS <- true
	fmt.Printf("\n(*DemoDeviceClient) StartDemoJob( %s ) -> LTE OFF; GPS ON...\n", demo.DESDevSerial)
	time.Sleep(time.Millisecond * (DES_PING_TIMEOUT / 2))

	/* CAPTURE TIME VALUE FOR JOB INTITALIZATION: DB/JOB NAME, ADM, HDR, CFG, EVT */
	startTime := time.Now().UTC().UnixMilli()

	/* USED INCASE WE NEED TO CREATE DEFAULT SETTINGS */
	demo.DESJob = pkg.DESJob{
		DESJobRegTime:   startTime,
		DESJobRegAddr:   demo.DESDevSerial,
		DESJobRegUserID: start.STA.StaUserID,
		DESJobRegApp:    start.STA.StaApp,

		DESJobName:  fmt.Sprintf("%s_%d", demo.DESDevSerial, startTime),
		DESJobStart: startTime,
		DESJobEnd:   0,
		DESJobLng:   -114.75 + rand.Float64()*(-110.15+114.75),
		DESJobLat:   51.85 + rand.Float64()*(54.35-51.85),
		DESJobDevID: demo.DESDevID,
	}

	/* RECONNECT LTE AFTER SIMULATED GPS AQUIRE */
	demo.GPS <- false
	fmt.Printf("\n(*DemoDeviceClient) StartDemoJob( %s ) -> LTE ON GPS OFF...\n", demo.DESDevSerial)

	demo.ADM = start.ADM
	demo.ADM.AdmTime = startTime
	demo.ADM.AdmAddr = demo.DESDevSerial

	/* CREATE A LOCAL STATE VARIABLE TO AVOID ALTERING LOGGING MODE PREMATURELY */
	sta := demo.STA
	sta.StaTime = startTime
	sta.StaLogging = OP_CODE_JOB_STARTED
	sta.StaJobName = demo.DESJobName

	demo.HDR = start.HDR
	demo.HDR.HdrTime = startTime
	demo.HDR.HdrAddr = demo.DESDevSerial
	demo.HDR.HdrJobStart = startTime
	demo.HDR.HdrJobEnd = 0
	demo.HDR.HdrGeoLng = demo.DESJobLng
	demo.HDR.HdrGeoLat = demo.DESJobLat

	demo.CFG = start.CFG
	demo.CFG.CfgTime = startTime
	demo.CFG.CfgAddr = demo.DESDevSerial
	demo.CFG.CfgVlvTgt = MODE_VENT
	demo.CFG.Validate()

	demo.EVT = start.EVT
	demo.EVT.EvtTime = startTime
	demo.EVT.EvtAddr = demo.DESDevSerial
	demo.EVT.EvtCode = OP_CODE_JOB_STARTED
	demo.EVT.EvtTitle = "JOB STARTED"
	demo.EVT.EvtMsg = demo.DESJobName

	/* WRITE TO FLASH - CMDARCHIVE */
	demo.WriteAdmToJSONFile(demo.CmdArchiveName(), demo.ADM)
	demo.WriteStateToJSONFile(demo.CmdArchiveName(), sta)
	demo.WriteHdrToJSONFile(demo.CmdArchiveName(), demo.HDR)
	demo.WriteCfgToJSONFile(demo.CmdArchiveName(), demo.CFG)
	demo.WriteEvtToJSONFile(demo.CmdArchiveName(), demo.EVT)

	/* WRITE TO FLASH - JOB */
	demo.WriteAdmToJSONFile(demo.DESJobName, demo.ADM)
	demo.WriteStateToJSONFile(demo.DESJobName, sta)
	demo.WriteHdrToJSONFile(demo.DESJobName, demo.HDR)
	demo.WriteCfgToJSONFile(demo.DESJobName, demo.CFG)
	demo.WriteEvtToJSONFile(demo.DESJobName, demo.EVT)

	/* LOAD VALUE INTO SIM 'RAM'
	UPDATE THE DEVICE STATE ENABLING MQTT MESSAGE WRITES TO ACTIVE JOB
	AFTER WE HAVE WRITTEN THE INITIAL JOB RECORDS
	AND BEFORE WE SEND THE RESPONSE
	*/
	demo.STA = sta

	if !offline {
		/* SEND CONFIRMATION */
		go demo.MQTTPublication_DemoDeviceClient_SIGStartJob(
			StartJob{
				ADM: demo.ADM,
				STA: demo.STA,
				HDR: demo.HDR,
				CFG: demo.CFG,
				EVT: demo.EVT,
			})
	}

	/* RUN JOB... */
	go demo.Demo_Simulation(demo.STA.StaJobName, demo.CFG.CfgVlvTgt, demo.CFG.CfgOpSample)

	fmt.Printf("\n(*DemoDeviceClient) StartDemoJob( ) -> RUNNING %s...\n", demo.STA.StaJobName)
}

func (demo *DemoDeviceClient) EndDemoJob(evt Event) {
	// fmt.Printf("\n%s (*DemoDeviceClient) EndDemoJob( )...\n", demo.DESDevSerial)

	// demo.DESMQTTClient.WG.Wait()
	// demo.DESMQTTClient.WG.Add(1)
	if demo.Stop != nil {
		demo.Stop <- struct{}{}
	}

	/* CAPTURE TIME VALUE FOR JOB TERMINATION: HDR, EVT */
	endTime := time.Now().UTC().UnixMilli()

	fmt.Printf("\n%s (*DemoDeviceClient) EndDemoJob( ) at:\t%d\n", demo.DESDevSerial, endTime)
	// demo.GetHdrFromFlash(demo.CmdArchiveName(), &demo.HDR)
	hdr := demo.HDR
	hdr.HdrTime = endTime
	hdr.HdrAddr = demo.DESDevSerial
	hdr.HdrUserID = evt.EvtUserID
	hdr.HdrApp = evt.EvtApp
	hdr.HdrJobEnd = endTime

	evt.EvtTime = endTime
	evt.EvtAddr = demo.DESDevSerial
	evt.EvtCode = OP_CODE_JOB_ENDED
	evt.EvtTitle = "JOB ENDED"
	evt.EvtMsg = demo.STA.StaJobName // BEFORE WE CHANGE IT

	sta := demo.STA
	sta.StaTime = endTime
	sta.StaAddr = demo.DESDevSerial
	sta.StaLogging = OP_CODE_JOB_ENDED
	sta.StaJobName = demo.CmdArchiveName()

	/* LOAD VALUE INTO SIM 'RAM'
	UPDATE THE DEVICE EVENT CODE, AND STATE DISABLING MQTT MESSAGE WRITES TO ACTIVE JOB
	BEFORE WE HAVE WRITTEN THE FINAL JOB RECORDS
	*/
	demo.HDR = hdr
	demo.EVT = evt
	demo.STA = sta

	/* WRITE TO FLASH - CMDARCHIVE */
	demo.WriteHdrToJSONFile(demo.CmdArchiveName(), hdr)
	demo.WriteEvtToJSONFile(demo.CmdArchiveName(), evt)
	demo.WriteStateToJSONFile(demo.CmdArchiveName(), sta)

	/* WRITE TO FLASH - JOB */
	demo.WriteHdrToJSONFile(demo.DESJobName, hdr)
	demo.WriteEvtToJSONFile(demo.DESJobName, evt)
	demo.WriteStateToJSONFile(demo.DESJobName, sta)

	/* SEND FINAL DATA MODELS */
	demo.MQTTPublication_DemoDeviceClient_SIGHeader(hdr)
	demo.MQTTPublication_DemoDeviceClient_SIGEvent(evt)
	demo.MQTTPublication_DemoDeviceClient_SIGState(sta)

	/* SEND END JOB CONFIRMATION */
	demo.MQTTPublication_DemoDeviceClient_SIGEndJob(sta)

	/* GET DEFAULT MODELS AND UPDATE TIMES */
	adm := demo.ADM
	adm.DefaultSettings_Admin(demo.DESRegistration)
	adm.AdmTime = time.Now().UTC().UnixMilli()

	hdr.DefaultSettings_Header(demo.DESRegistration)
	hdr.HdrTime = time.Now().UTC().UnixMilli()

	cfg := demo.CFG
	cfg.DefaultSettings_Config(demo.DESRegistration)
	cfg.CfgTime = time.Now().UTC().UnixMilli()

	/* TRANSMIT DEFAULT MODELS */
	demo.MQTTPublication_DemoDeviceClient_SIGAdmin(adm)
	demo.MQTTPublication_DemoDeviceClient_SIGHeader(hdr)
	demo.MQTTPublication_DemoDeviceClient_SIGConfig(cfg)

	/* LOAD DEFAULT MODELS INTO RAM */
	demo.ADM = adm
	demo.HDR = hdr
	demo.CFG = cfg

	// demo.DESMQTTClient.WG.Done()

	fmt.Printf("\n(*DemoDeviceClient) EndDemoJob( ) -> ENDED: %s\n", demo.STA.StaJobName)
}

func (demo *DemoDeviceClient) SimOfflineStart() {
	fmt.Printf("\n(*DemoDeviceClient) SimOfflineStart( %s )...\n", demo.DESDevSerial)

	demo.DESJobRegAddr = demo.DESDevSerial
	demo.DESJobRegUserID = demo.DESU.GetUUIDString()
	demo.DESJobRegApp = demo.DESDevSerial

	adm := Admin{}
	adm.DefaultSettings_Admin(demo.DESRegistration)

	sta := demo.STA
	sta.DefaultSettings_State(demo.DESRegistration)

	hdr := Header{}
	hdr.DefaultSettings_Header(demo.DESRegistration)
	hdr.HdrWellCo = "Some Offline Co."

	cfg := Config{}
	cfg.DefaultSettings_Config(demo.DESRegistration)
	// cfg.CfgOpSample = 200

	evt := Event{}
	evt.DefaultSettings_Event(demo.DESRegistration)

	demo.StartDemoJob(
		StartJob{
			ADM: adm,
			STA: sta,
			HDR: hdr,
			CFG: cfg,
			EVT: evt,
		}, true,
	)

}

/*DEMO SIM -> PUBLISH TO MQTT */
func (demo *DemoDeviceClient) Demo_Simulation(job string, mode, rate int32) {
	// fmt.Printf("\n(*DemoDeviceClient) Demo_Simulation( ) %s -> Starting simulation...\n", demo.DESDevSerial)

	// demo.DESMQTTClient.WG.Wait()
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
			demo.WriteSMPToHEXFile(job, smp)
			smpMQTT := Demo_EncodeMQTTSampleMessage(job, 0, smp)
			demo.MQTTPublication_DemoDeviceClient_SIGSample(smpMQTT)
		}
	}

	fmt.Printf("\n(*DemoDeviceClient) Demo_Simulation( ) -> SIMULATION STOPPED: %s\n", demo.DESDevSerial)
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

	fmt.Printf("\n(*DemoDeviceClient) Set_MTx() -> %s:, %f, %f, %f, %f\n", demo.DESDevSerial, demo.MTxCh4.VMax, demo.MTxHiFlow.VMax, demo.MTxLoFlow.VMax, demo.MTxBuild.VMax)
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
	hex = append(hex, pkg.GetBytes_L(x)...)
	hex = append(hex, pkg.GetBytes_L(ch)...)
	hex = append(hex, pkg.GetBytes_L(hf)...)
	hex = append(hex, pkg.GetBytes_L(lf)...)
	hex = append(hex, pkg.GetBytes_L(p)...)
	hex = append(hex, pkg.GetBytes_L(bc)...)
	hex = append(hex, pkg.GetBytes_L(bv)...)
	hex = append(hex, pkg.GetBytes_L(mv)...)
	hex = append(hex, pkg.GetBytes_L(vt)...)
	hex = append(hex, pkg.GetBytes_L(vp)...)
	// fmt.Printf("Hex:\t%X\n", hex)

	b64url := pkg.BytesToBase64URL(hex)
	// fmt.Printf("Base64URL:\t%s\n\n", b64url)

	msg := MQTT_Sample{
		DesJobName: job,
		Data:       b64url,
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

package c001v001

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"os"
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
	Qty     int   `json:"qty"`
	Dur     int64 `json:"dur"`
	FillQty int64 `json:"fill_qty"`
	MTxCh4 DemoModeTransition `json:"mtx_ch4"`
	MTxFlow DemoModeTransition `json:"mtx_flow"`
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
	
	ADM Admin // RAM Value
	ADMFile *os.File  // Flash
	
	HDR Header // RAM Value
	HDRFile *os.File  // Flash
	
	CFG Config // RAM Value
	CFGFile *os.File // Flash
	
	EVT Event // RAM Value
	EVTFile *os.File  // Flash
	
	Sim
	sizeChan   chan int
	sentChan   chan int
	WSClientID string
	CTX        context.Context
	Cancel     context.CancelFunc
	pkg.DESMQTTClient
}

func (demo DemoDeviceClient) WSDemoDeviceClient_Connect(c *websocket.Conn) {

	simStr, _ := url.QueryUnescape(c.Query("sim"))
	des_regStr, _ := url.QueryUnescape(c.Query("des_reg"))

	des_reg := pkg.DESRegistration{}
	if err := json.Unmarshal([]byte(des_regStr), &des_reg); err != nil {
		pkg.Trace(err)
	}
	des_reg.DESDevRegAddr = c.RemoteAddr().String()
	des_reg.DESJobRegAddr = c.RemoteAddr().String()

	sim := Sim{}
	if err := json.Unmarshal([]byte(simStr), &sim); err != nil {
		pkg.Trace(err)
	}

	wscid := fmt.Sprintf("%s-DEMO-%s-%s",
		c.RemoteAddr().String(),
		des_reg.DESDevRegUserID,
		des_reg.DESJobName,
	)

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

	demo.MQTTDemoDeviceClient_Connect()

	evt := demo.GetLastEvent()

	if evt.EvtCode == 2 || evt.EvtCode == 0 {
		fmt.Printf("\n%s: waiting for job start event...\n", demo.DESDevSerial)
	} else {
		go demo.Demo_Run_Sim()
		fmt.Printf("\n%s: simulation running...\n", demo.DESDevSerial)
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
		"DMODevice-%s-%s-%s",
		demo.DESDevClass,
		demo.DESDevVersion,
		demo.DESDevSerial,
	)
	if err = demo.DESMQTTClient.DESMQTTClient_Connect(); err != nil {
		return err
	}
	pkg.Json(`(demo *DemoDeviceClient) MQTTDemoDeviceClient_Connect()(...) -> ClientID:`, demo.DESMQTTClient.ClientOptions.ClientID)

	demo.MQTTSubscription_DemoDeviceClient_CMDAdmin().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDHeader().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDConfig().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDEvent().Sub(demo.DESMQTTClient)

	pkg.MQTTDemoClients[demo.DESDevSerial] = demo.DESMQTTClient
	demoClient := pkg.MQTTDemoClients[demo.DESDevSerial]
	fmt.Printf("\n%s client ID: %s\n", demo.DESDevSerial, demoClient.MQTTClientID)

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

	delete(pkg.MQTTDemoClients, demo.DESDevSerial)

	fmt.Printf("(demo *DemoDeviceClient) MQTTDemoDeviceClient_Disconnect( ... ): Complete.\n")
}

/*
SUBSCRIPTIONS
*/

/* SUBSCRIPTION -> ADMINISTRATION -> UPON RECEIPT, REPLY TO .../cmd/admin */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDAdmin() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDAdmin(),
		Handler: func(c phao.Client, msg phao.Message) {

			// adm := &Admin{}
			if err := json.Unmarshal(msg.Payload(), &demo.ADM); err != nil {
				pkg.Trace(err)
			}
			evt := demo.GetLastEvent()
			if evt.EvtCode > 2 {
				/* SIMULATE HAVING DONE SOMETHING */
				time.Sleep(time.Millisecond * 200)
				demo.ADM.AdmTime = time.Now().UTC().UnixMilli()
				demo.ADM.AdmApp = demo.DESDevSerial
				demo.MQTTPublication_DemoDeviceClient_SIGAdmin(&demo.ADM)
			}
		},
	}
}

/* SUBSCRIPTIONS -> HEADER -> UPON RECEIPT, REPLY TO .../cmd/header */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDHeader() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDHeader(),
		Handler: func(c phao.Client, msg phao.Message) {

			// hdr := &Header{}
			if err := json.Unmarshal(msg.Payload(), &demo.HDR); err != nil {
				pkg.Trace(err)
			}
			evt := demo.GetLastEvent()
			if evt.EvtCode > 2 {
				/* SIMULATE HAVING DONE SOMETHING */
				time.Sleep(time.Millisecond * 200)
				demo.HDR.HdrTime = time.Now().UTC().UnixMilli()
				demo.HDR.HdrApp = demo.DESDevSerial
				demo.MQTTPublication_DemoDeviceClient_SIGHeader(&demo.HDR)
			}
		},
	}
}

/* SUBSCRIPTIONS -> CONFIGURATION -> UPON RECEIPT, REPLY TO .../cmd/config */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDConfig() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDConfig(),
		Handler: func(c phao.Client, msg phao.Message) {

			// cfg := &Config{}
			if err := json.Unmarshal(msg.Payload(), &demo.CFG); err != nil {
				pkg.Trace(err)
			}
			evt := demo.GetLastEvent()
			if evt.EvtCode > 2 {
				/* SIMULATE HAVING DONE SOMETHING */
				time.Sleep(time.Millisecond * 200)
				demo.CFG.CfgTime = time.Now().UTC().UnixMilli()
				demo.CFG.CfgApp = demo.DESDevSerial
				demo.MQTTPublication_DemoDeviceClient_SIGConfig(&demo.CFG)
			}
		},
	}
}

/* SUBSCRIPTIONS -> CONFIGURATION -> UPON RECEIPT, REPLY TO .../cmd/config */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDEvent() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDEvent(),
		Handler: func(c phao.Client, msg phao.Message) {

			// evt := &Event{}
			if err := json.Unmarshal(msg.Payload(), &demo.EVT); err != nil {
				pkg.Trace(err)
			}

			/* LOG EVENT TO Job_0 & Job_X */

			/* SIMULATE EVENT RESPONSE */
			time.Sleep(time.Millisecond * 200)
			demo.EVT.EvtTime = time.Now().UTC().UnixMilli()
			demo.EVT.EvtApp = demo.DESDevSerial

			fmt.Printf("\n(demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDEvent() -> %s\n", demo.MQTTTopic_CMDEvent())
			switch demo.EVT.EvtCode {

			case 1: // Start Job
				demo.StartDemoJob()

			case 2: // End Job
				demo.EndDemoJob()

			case 10: // Mode Vent
			case 11: // Mode Build
			case 12: // Mode Hi Flow
			case 13: // Mode Lo Flow
				// EDIT / SEND CONFIG -> MOVE VALVE
				demo.MoveValve()

			default:
				demo.MQTTPublication_DemoDeviceClient_SIGEvent(&demo.EVT)

			}
		},
	}
}

/*
PUBLICATIONS
*/
/* PUBLICATION -> ADMIN -> SIMULATED ADMINS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGAdmin(adm *Admin) bool {
	return (pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGAdmin(),
		Message:  pkg.MakeMQTTMessage(adm.FilterAdmRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> HEADER -> SIMULATED HEADERS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGHeader(hdr *Header) bool {
	return (pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGHeader(),
		Message:  pkg.MakeMQTTMessage(hdr.FilterHdrRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> CONFIG -> SIMULATED CONFIGS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGConfig(cfg *Config) bool {
	return (pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGConfig(),
		Message:  pkg.MakeMQTTMessage(cfg.FilterCfgRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> EVENT -> SIMULATED EVENTS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGEvent(evt *Event) bool {
	return (pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGEvent(),
		Message:  pkg.MakeMQTTMessage(evt.FilterEvtRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}

/* PUBLICATION -> SAMPLE -> SIMULATED SAMPLES */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGSample(mqtts *MQTT_Sample) bool {

	b64, err := json.Marshal(mqtts)
	if err != nil {
		pkg.Trace(err)
	} // pkg.Json("MQTT_Sample:", b64)
	return (pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGSample(),
		Message:  string(b64),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}).Pub(demo.DESMQTTClient)
}

/*DEMO SIM -> PUBLISH TO MQTT */
func (demo *DemoDeviceClient) Demo_Siulation(  ) {
	// t0 := time.Now()

	// for {



	// }

}

func (demo *DemoDeviceClient) Demo_Run_Sim() {
	fmt.Printf("\n (demo *DemoDeviceClient) Demo_Run_Sim( ): demo.Sim.Dur %d\n", demo.Sim.Dur)
	fmt.Printf("\n (demo *DemoDeviceClient) Demo_Run_Sim( ): demo.Sim.Qty %d\n", demo.Sim.Qty)
	t0 := time.Now()
	i := 1
	for i < demo.Sim.Qty {

		mqtts := Demo_Make_Sim_Sample(t0, time.Now(), demo.DESJob.DESJobName)

		demo.MQTTPublication_DemoDeviceClient_SIGSample(&mqtts)

		time.Sleep(time.Millisecond * time.Duration(demo.Job.Configs[0].CfgOpSample))
		i++

	}

	demo.MQTTDemoDeviceClient_Disconnect()
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
func Demo_Make_Sim_Sample(t0, ti time.Time, job string) MQTT_Sample {

	tumic := ti.UnixMilli()
	data := []pkg.TimeSeriesData{
		/* "AAABgss3rYBCxs2nO2VgQj6qrwk/JpeNPv6JZUFWw+1BUWVuAAQABA==" */
		{ // methane
			Data: []pkg.TSDPoint{{
				X: tumic,
				Y: Demo_Mode_Transition(t0, ti, time.Duration(time.Second*250), 97.99999, 0.01),
			}},
			Min: 0,
			Max: 100,
		},
		{ // high_flow
			Data: []pkg.TSDPoint{{
				X: tumic,
				Y: Demo_Mode_Transition(t0, ti, time.Duration(time.Second*30), 1.79999, 0.01),
			}},
			Min: 0,
			Max: 250,
		},
		{ // low_flow
			Data: []pkg.TSDPoint{{
				X: tumic,
				Y: Demo_Mode_Transition(t0, ti, time.Duration(time.Second*30), 1.79999, 0.01),
			}},
			Min: 0,
			Max: 2,
		},
		{ // pressure
			Data: []pkg.TSDPoint{{
				X: tumic,
				Y: Demo_Mode_Transition(t0, ti, time.Duration(time.Second*600), 18.99999, 699.99999),
			}},
			Min: 0,
			Max: 1500,
		},
		{ // battery_current
			// Data: []pkg.TSDPoint{{X: tumic, Y: YSinX(t0, ti, 0.249, 0.09)}},
			Data: []pkg.TSDPoint{{X: tumic, Y: 0.049 + rand.Float32()*0.023}},
			Min:  0,
			Max:  1.5,
		},
		{ // battery_voltage
			// Data: []pkg.TSDPoint{{X: tumic, Y: YCosX(t0, ti, 13.9, 0.8)}},
			Data: []pkg.TSDPoint{{X: tumic, Y: 12.733 + rand.Float32()*0.072}},
			Min:  0,
			Max:  15,
		},
		{ // motor_voltage
			// Data: []pkg.TSDPoint{{X: tumic, Y: YSinX(t0, ti, 12.9, 0.9)}},
			Data: []pkg.TSDPoint{{X: tumic, Y: 11.9 + rand.Float32()*0.033}},
			Min:  0,
			Max:  15,
		},
		{ // valve_target
			Data: []pkg.TSDPoint{{X: tumic, Y: 0}},
			Min:  0,
			Max:  10,
		},
		{ // valve_position
			Data: []pkg.TSDPoint{{X: tumic, Y: 0}},
			Min:  0,
			Max:  10,
		},
	}

	return Demo_EncodeMQTTSampleMessage(job, 0, data)
}
func Demo_EncodeMQTTSampleMessage(job string, i int, data []pkg.TimeSeriesData) MQTT_Sample {
	// fmt.Println("\nDemo_EncodeMQTTSampleMessage()...")

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


func (demo *DemoDeviceClient) StartDemoJob() {
	fmt.Printf("(demo *DemoDeviceClient) StartDemoJob( )...\n")
	
	zero := Job{DESRegistration: demo.GetZeroJob()}
	db := zero.JDB()
	db.Connect()
	defer db.Close()

	// Get last ADM from Job_0 -> if null { use default }
	// adm := Admin{}
	res := db.Where("adm_time = ?", demo.EVT.EvtTime).Last(&demo.ADM)
	if res.Error != nil {
		pkg.Trace(res.Error)
	} 
	if demo.ADM.AdmID == 0 {
		demo.ADM = zero.RegisterJob_Default_JobAdmin()
	}
	demo.ADM.AdmTime = demo.EVT.EvtTime
	demo.ADM.AdmAddr = demo.DESDevSerial
	demo.ADM.AdmUserID = demo.EVT.EvtUserID
	demo.ADM.AdmApp = demo.EVT.EvtApp
	
	// Get last HDR from Job_0 -> if null { use default }
	// hdr := Header{}
	res = db.Where("hdr_time = ?", demo.EVT.EvtTime).Last(&demo.HDR)
	if res.Error != nil {
		pkg.Trace(res.Error)
	} 
	if demo.HDR.Hdr_ID == 0 {
		demo.HDR = zero.RegisterJob_Default_JobHeader()
	}
	demo.HDR.HdrTime = demo.EVT.EvtTime
	demo.HDR.HdrAddr = demo.DESDevSerial
	demo.HDR.HdrUserID = demo.EVT.EvtUserID
	demo.HDR.HdrApp = demo.EVT.EvtApp
	demo.HDR.HdrJobName = fmt.Sprintf("%s_%d", demo.DESDevSerial, demo.EVT.EvtTime)
	demo.HDR.HdrJobStart = demo.EVT.EvtTime
	demo.HDR.HdrGeoLng = -114.75 + rand.Float32() * ( -110.15 + 114.75 )
	demo.HDR.HdrGeoLat = 51.85 + rand.Float32() * ( 54.35 - 51.85 )
	
	// Get last CFG from Job_0 -> if null { use default }
	// cfg := Config{}
	res = db.Where("cfg_time = ?", demo.EVT.EvtTime).Last(&demo.CFG)
	if res.Error != nil {
		pkg.Trace(res.Error)
	} 
	if demo.CFG.CfgID == 0 {
		demo.CFG = zero.RegisterJob_Default_JobConfig()
	}
	demo.CFG.CfgTime = demo.EVT.EvtTime
	demo.CFG.CfgAddr = demo.DESDevSerial
	demo.CFG.CfgUserID = demo.EVT.EvtUserID
	demo.CFG.CfgApp = demo.EVT.EvtApp
	
	db.Close()

	// create /job/files
	dir := fmt.Sprintf("demo/%s", demo.HDR.HdrJobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.Trace(err)
	}
	adm_txt, err := os.Create(fmt.Sprintf("%s/adm.txt", dir))
	if err != nil {
		pkg.Trace(err)
	}
	defer adm_txt.Close()
	hdr_txt, _ := os.Create(fmt.Sprintf("%s/hdr.txt", dir))
	defer hdr_txt.Close()
	cfg_txt, _ := os.Create(fmt.Sprintf("%s/cfg.txt", dir))
	defer cfg_txt.Close()
	evt_txt, _ := os.Create(fmt.Sprintf("%s/evt.txt", dir))
	defer evt_txt.Close()
	smp_txt, _ := os.Create(fmt.Sprintf("%s/smp.txt", dir))
	defer smp_txt.Close()

	// each[ ADM, HDR, CFG, EVT ] { MQTT out ( avec time.Now() ) }
	demo.MQTTPublication_DemoDeviceClient_SIGAdmin(&demo.ADM)
	demo.MQTTPublication_DemoDeviceClient_SIGHeader(&demo.HDR)
	demo.MQTTPublication_DemoDeviceClient_SIGConfig(&demo.CFG)
	demo.MQTTPublication_DemoDeviceClient_SIGEvent(&demo.EVT)
}

func (demo *DemoDeviceClient) EndDemoJob() {
	fmt.Printf(" END JOB\n")

	demo.Device.Job.DESRegistration = demo.GetCurrentJob()

	// hdr := &Header{}
	demo.HDR.HdrTime = demo.EVT.EvtTime
	demo.HDR.HdrAddr = demo.DESDevSerial
	demo.HDR.HdrUserID = demo.EVT.EvtUserID
	demo.HDR.HdrApp = demo.EVT.EvtApp
	demo.HDR.HdrJobEnd = demo.EVT.EvtTime
	demo.MQTTPublication_DemoDeviceClient_SIGHeader(&demo.HDR)
	demo.MQTTPublication_DemoDeviceClient_SIGEvent(&demo.EVT)
}

func (demo *DemoDeviceClient) MoveValve() {
}
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

	ADM     Admin    // RAM Value
	ADMFile *os.File // Flash

	HDR     Header   // RAM Value
	HDRFile *os.File // Flash

	CFG     Config   // RAM Value
	CFGFile *os.File // Flash

	EVT     Event    // RAM Value
	EVTFile *os.File // Flash

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

	// if no demo client ...
	// demo.MQTTDemoDeviceClient_Connect()

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

func GetDemoDeviceList() (demos []pkg.DESRegistration, err error) {

	demoSubQry := pkg.DES.DB.
		Table("des_devs").
		Where("des_dev_class = '001' AND des_dev_version = '001' AND des_dev_serial LIKE 'DEMO%' ").
		Select("des_dev_serial, MAX(des_dev_reg_time) AS max_time").
		Group("des_dev_serial")

	jobSubQry := pkg.DES.DB.
		Table("des_jobs").
		Where("des_jobs.des_job_end = 0").
		Select("des_job_id, MAX(des_job_reg_time) AS max_reg_time").
		Group("des_job_id")

	jobQry := pkg.DES.DB.
		Table("des_jobs").
		Select("*").
		Joins(`JOIN ( ? ) jobs
			ON des_jobs.des_job_id = jobs.des_job_id
			AND des_jobs.des_job_reg_time = jobs.max_reg_time`,
			jobSubQry)

	qry := pkg.DES.DB.
		Table("des_devs").
		Select(" des_devs.*, j.* ").
		Joins(`JOIN ( ? ) d 
			ON des_devs.des_dev_serial = d.des_dev_serial 
			AND des_devs.des_dev_reg_time = d.max_time`,
			demoSubQry).
		Joins(`JOIN ( ? ) j
			ON des_devs.des_dev_id = j.des_job_dev_id`,
			jobQry).
		Order("des_devs.des_dev_serial DESC")

	res := qry.Scan(&demos)
	err = res.Error
	return

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

			/* LOAD VALUE INTO SIM 'RAM' */
			if err := json.Unmarshal(msg.Payload(), &demo.ADM); err != nil {
				pkg.Trace(err)
			}
			evt := demo.GetLastEvent()
			if evt.EvtCode > 2 {
				/* SIMULATE HAVING DONE SOMETHING */
				time.Sleep(time.Millisecond * 200)
				/* UPDATE TIME / SOURCE */
				demo.ADM.AdmTime = time.Now().UTC().UnixMilli()
				demo.ADM.AdmApp = demo.DESDevSerial
				/* WRITE TO SIM 'FLASH' */
				demo.ADMFile.Write(msg.Payload())
				/* SEND CONFIRMATION */
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
func (demo *DemoDeviceClient) Demo_Siulation() {
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


func (demo *DemoDeviceClient) WriteAdmToFlash(job Job, adm Admin) (err error) {

	admBytes := adm.FilterAdmBytes()
	fmt.Printf("\nadmBytes ( %d ) : %v\n", len(admBytes), admBytes)

	dir := fmt.Sprintf("demo/%s", job.DESJobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.Trace(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/adm.bin", dir), os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return pkg.Trace(err)
	}
	defer f.Close()

	_, err = f.Write(admBytes)
	if err != nil {
		return pkg.Trace(err)
	}

	f.Close()

	return
}
func (demo *DemoDeviceClient) ReadAdmFromFlash(job Job, time int64) (adm []byte, err error) {

	dir := fmt.Sprintf("demo/%s", job.DESJobName)
	f, err := os.OpenFile(fmt.Sprintf("%s/adm.bin", dir), os.O_RDONLY, 0600)
	if err != nil {
		return nil, pkg.Trace(err)
	}

	admFile, err := ioutil.ReadAll(f)
	if err != nil {
		pkg.Trace(err)
		return
	}
	eof := len(admFile)

	adm = admFile[eof-288 : eof]

	fmt.Printf("\nadmBytes ( %d ) : %v\n", len(adm), adm)

	defer f.Close()

	return
}
func (demo *DemoDeviceClient) MakeAdmFromBytes(b []byte) (adm Admin) {

	adm = Admin{

		AdmTime:   pkg.BytesToInt64_L(b[0:8]),
		AdmAddr:   pkg.FFStrBytesToString(b[8:44]),
		AdmUserID: pkg.FFStrBytesToString(b[44:80]),
		AdmApp:    pkg.FFStrBytesToString(b[80:116]),

		AdmDefHost: pkg.FFStrBytesToString(b[116:148]),
		AdmDefPort: pkg.BytesToInt32_L(b[148:152]),
		AdmOpHost: pkg.FFStrBytesToString(b[152:184]),
		AdmOpPort: pkg.BytesToInt32_L(b[184:188]),

		AdmSerial: pkg.FFStrBytesToString(b[188:198]),
		AdmVersion: pkg.FFStrBytesToString(b[198:201]),
		AdmClass: pkg.FFStrBytesToString(b[201:204]),

		AdmBatHiAmp:   pkg.BytesToFloat32_L(b[204:208]),
		AdmBatLoVolt:  pkg.BytesToFloat32_L(b[208:212]),

		AdmMotHiAmp: pkg.BytesToFloat32_L(b[212:216]),

		AdmHFSFlow: pkg.BytesToFloat32_L(b[216:220]),
		AdmHFSFlowMin: pkg.BytesToFloat32_L(b[220:224]),
		AdmHFSFlowMax: pkg.BytesToFloat32_L(b[224:228]),
		AdmHFSPress: pkg.BytesToFloat32_L(b[228:232]),
		AdmHFSPressMin: pkg.BytesToFloat32_L(b[232:236]),
		AdmHFSPressMax: pkg.BytesToFloat32_L(b[236:240]),
		AdmHFSDiff: pkg.BytesToFloat32_L(b[240:244]),
		AdmHFSDiffMin: pkg.BytesToFloat32_L(b[244:248]),
		AdmHFSDiffMax: pkg.BytesToFloat32_L(b[248:252]),

		AdmLFSFlow: pkg.BytesToFloat32_L(b[252:256]),
		AdmLFSFlowMin: pkg.BytesToFloat32_L(b[256:260]),
		AdmLFSFlowMax: pkg.BytesToFloat32_L(b[260:264]),
		AdmLFSPress: pkg.BytesToFloat32_L(b[264:268]),
		AdmLFSPressMin: pkg.BytesToFloat32_L(b[268:272]),
		AdmLFSPressMax: pkg.BytesToFloat32_L(b[272:276]),
		AdmLFSDiff: pkg.BytesToFloat32_L(b[276:280]),
		AdmLFSDiffMin: pkg.BytesToFloat32_L(b[280:284]),
		AdmLFSDiffMax: pkg.BytesToFloat32_L(b[284:288]),
		
	}

	return
}


func (demo *DemoDeviceClient) WriteHdrToFlash(job Job, hdr Header) (err error) {

	hdrBytes := hdr.FilterHdrBytes()
	fmt.Printf("\nhdrBytes ( %d ) : %v\n", len(hdrBytes), hdrBytes)

	dir := fmt.Sprintf("demo/%s", job.DESJobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.Trace(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/hdr.bin", dir), os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return pkg.Trace(err)
	}
	defer f.Close()

	_, err = f.Write(hdrBytes)
	if err != nil {
		return pkg.Trace(err)
	}

	f.Close()

	return
}
func (demo *DemoDeviceClient) ReadHdrFromFlash(job Job, time int64) (hdr []byte, err error) {

	dir := fmt.Sprintf("demo/%s", job.DESJobName)
	f, err := os.OpenFile(fmt.Sprintf("%s/hdr.bin", dir), os.O_RDONLY, 0600)
	if err != nil {
		return nil, pkg.Trace(err)
	}

	hdrFile, err := ioutil.ReadAll(f)
	if err != nil {
		pkg.Trace(err)
		return
	}
	eof := len(hdrFile)

	hdr = hdrFile[eof-324 : eof]

	fmt.Printf("\nhdrBytes ( %d ) : %v\n", len(hdr), hdr)

	defer f.Close()

	return
}
func (demo *DemoDeviceClient) MakeHdrFromBytes(b []byte) (hdr Header) {

	hdr = Header{

		HdrTime:   pkg.BytesToInt64_L(b[0:8]),
		HdrAddr:   pkg.FFStrBytesToString(b[8:44]),
		HdrUserID: pkg.FFStrBytesToString(b[44:80]),
		HdrApp:    pkg.FFStrBytesToString(b[80:116]),

		HdrJobName: pkg.FFStrBytesToString(b[116:140]),
		HdrJobStart: pkg.BytesToInt64_L(b[140:148]),
		HdrJobEnd: pkg.BytesToInt64_L(b[148:156]),

		HdrWellCo: pkg.FFStrBytesToString(b[156:188]),
		HdrWellName: pkg.FFStrBytesToString(b[188:220]),
		HdrWellSFLoc: pkg.FFStrBytesToString(b[220:252]),
		HdrWellBHLoc: pkg.FFStrBytesToString(b[252:284]),
		HdrWellLic: pkg.FFStrBytesToString(b[284:316]),

		HdrGeoLng: pkg.BytesToFloat32_L(b[316:320]),
		HdrGeoLat: pkg.BytesToFloat32_L(b[320:324]),
	}

	return
}


func (demo *DemoDeviceClient) WriteCfgToFlash(job Job, cfg Config) (err error) {

	cfgBytes := cfg.FilterCfgBytes()
	fmt.Printf("\ncfgBytes ( %d ) : %v\n", len(cfgBytes), cfgBytes)

	dir := fmt.Sprintf("demo/%s", job.DESJobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.Trace(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/cfg.bin", dir), os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return pkg.Trace(err)
	}
	defer f.Close()

	_, err = f.Write(cfgBytes)
	if err != nil {
		return pkg.Trace(err)
	}

	f.Close()

	return
}
func (demo *DemoDeviceClient) ReadCfgFromFlash(job Job, time int64) (cfg []byte, err error) {

	dir := fmt.Sprintf("demo/%s", job.DESJobName)
	f, err := os.OpenFile(fmt.Sprintf("%s/cfg.bin", dir), os.O_RDONLY, 0600)
	if err != nil {
		return nil, pkg.Trace(err)
	}

	cfgFile, err := ioutil.ReadAll(f)
	if err != nil {
		pkg.Trace(err)
		return
	}
	eof := len(cfgFile)

	cfg = cfgFile[eof-172 : eof]

	fmt.Printf("\ncfgBytes ( %d ) : %v\n", len(cfg), cfg)

	defer f.Close()

	return
}
func (demo *DemoDeviceClient) MakeCfgFromBytes(b []byte) (cfg Config) {

	cfg = Config{

		CfgTime:   pkg.BytesToInt64_L(b[0:8]),
		CfgAddr:   pkg.FFStrBytesToString(b[8:44]),
		CfgUserID: pkg.FFStrBytesToString(b[44:80]),
		CfgApp:    pkg.FFStrBytesToString(b[80:116]),

		CfgSCVD:     pkg.BytesToFloat32_L(b[116:120]),
		CfgSCVDMult: pkg.BytesToFloat32_L(b[120:124]),
		CfgSSPRate:  pkg.BytesToFloat32_L(b[124:128]),
		CfgSSPDur:   pkg.BytesToFloat32_L(b[128:132]),
		CfgHiSCVF:   pkg.BytesToFloat32_L(b[132:136]),
		CfgFlowTog:  pkg.BytesToFloat32_L(b[136:140]),

		CfgVlvTgt: pkg.BytesToInt32_L(b[140:144]),
		CfgVlvPos: pkg.BytesToInt32_L(b[144:148]),

		CfgOpSample: pkg.BytesToInt32_L(b[148:152]),
		CfgOpLog:    pkg.BytesToInt32_L(b[152:156]),
		CfgOpTrans:  pkg.BytesToInt32_L(b[156:160]),

		CfgDiagSample: pkg.BytesToInt32_L(b[160:164]),
		CfgDiagLog:    pkg.BytesToInt32_L(b[164:168]),
		CfgDiagTrans:  pkg.BytesToInt32_L(b[168:172]),
	}

	return
}


func (demo *DemoDeviceClient) WriteEvtToFlash(job Job, evt Event) (err error) {

	evtBytes := evt.FilterEvtBytes()
	fmt.Printf("\nevtBytes ( %d ) : %v\n", len(evtBytes), evtBytes)

	dir := fmt.Sprintf("demo/%s/events", job.DESJobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.Trace(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/%d.bin", dir, evt.EvtTime), os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return pkg.Trace(err)
	}
	defer f.Close()

	_, err = f.Write(evtBytes)
	if err != nil {
		return pkg.Trace(err)
	}

	f.Close()

	return
}
func (demo *DemoDeviceClient) ReadEvtFromFlash(job Job, time int64) {

}

func (demo *DemoDeviceClient) WriteSmpToFlash(job Job, smp Sample) (err error) {

	smpBytes := smp.FilterSmpBytes()
	fmt.Printf("\nsmpBytes ( %d ) : %v\n", len(smpBytes), smpBytes)

	dir := fmt.Sprintf("demo/%s", job.DESJobName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.Trace(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/smp.bin", dir), os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return pkg.Trace(err)
	}
	defer f.Close()

	_, err = f.Write(smpBytes)
	if err != nil {
		return pkg.Trace(err)
	}

	f.Close()

	return
}

func (demo *DemoDeviceClient) StartDemoJob() {
	fmt.Printf("(demo *DemoDeviceClient) StartDemoJob( )...\n")

	zero := Job{DESRegistration: demo.GetZeroJob()}
	db := zero.JDB()
	db.Connect()
	defer db.Close()

	// Get last ADM from Job_0 -> if null { use default }
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
	demo.HDR.HdrGeoLng = -114.75 + rand.Float32()*(-110.15+114.75)
	demo.HDR.HdrGeoLat = 51.85 + rand.Float32()*(54.35-51.85)

	// Get last CFG from Job_0 -> if null { use default }
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

	/* WRITE TO FLASH - JOB_0 */
	demo.WriteAdmToFlash(zero, demo.ADM)
	demo.WriteHdrToFlash(zero, demo.HDR)
	demo.WriteCfgToFlash(zero, demo.CFG)
	demo.WriteEvtToFlash(zero, demo.EVT)

	/* WRITE TO FLASH - JOB_X */
	demo.Job = Job{
		DESRegistration: pkg.DESRegistration{
			DESDev: demo.DESDev,
			DESJob: pkg.DESJob{
				DESJobRegTime:   demo.EVT.EvtTime,
				DESJobRegAddr:   demo.DESDevSerial,
				DESJobRegUserID: demo.EVT.EvtUserID,
				DESJobRegApp:    demo.EVT.EvtApp,

				DESJobName:  demo.HDR.HdrJobName,
				DESJobStart: demo.EVT.EvtTime,
				DESJobEnd:   0,
				DESJobLng:   demo.HDR.HdrGeoLng,
				DESJobLat:   demo.HDR.HdrGeoLat,
				DESJobDevID: demo.DESDevID,
			},
		},
	}
	demo.WriteAdmToFlash(demo.Job, demo.ADM)
	demo.WriteHdrToFlash(demo.Job, demo.HDR)
	demo.WriteCfgToFlash(demo.Job, demo.CFG)
	demo.WriteEvtToFlash(demo.Job, demo.EVT)

	// each[ ADM, HDR, CFG, EVT ] { MQTT out ( avec time.Now() ) }
	demo.MQTTPublication_DemoDeviceClient_SIGAdmin(&demo.ADM)
	demo.MQTTPublication_DemoDeviceClient_SIGHeader(&demo.HDR)
	demo.MQTTPublication_DemoDeviceClient_SIGConfig(&demo.CFG)
	demo.MQTTPublication_DemoDeviceClient_SIGEvent(&demo.EVT)

	/* RUN JOB... */
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

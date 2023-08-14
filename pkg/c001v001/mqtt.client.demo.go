package c001v001

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"net/url"
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
}
type DemoDeviceClient struct {
	Device
	Sim
	sizeChan   chan int
	sentChan   chan int
	WSClientID string
	CTX        context.Context
	Cancel     context.CancelFunc
	pkg.DESMQTTClient
}

func (demo DemoDeviceClient) WSDemoDeviceClient_Connect(c *websocket.Conn) {
	fmt.Println("WSDemoDeviceClient_Connect( ... ): So far, so good...")

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

	go demo.Demo_Run_Sim()

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
	pkg.Json(`(demo *DemoDeviceClient) MQTTDemoDeviceClient_Connect()(...) -> 
	demo.DESMQTTClientDESMQTTClient_Connect() -> demo.DESMQTTClient.ClientOptions.ClientID:`,
		demo.DESMQTTClient.ClientOptions.ClientID)

	demo.MQTTSubscription_DemoDeviceClient_CMDAdmin().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDConfig().Sub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDEvent().Sub(demo.DESMQTTClient)

	return err
}
func (demo *DemoDeviceClient) MQTTDemoDeviceClient_Disconnect() {

	/* UNSUBSCRIBE FROM ALL MQTTSubscriptions */
	demo.MQTTSubscription_DemoDeviceClient_CMDAdmin().UnSub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDConfig().UnSub(demo.DESMQTTClient)
	demo.MQTTSubscription_DemoDeviceClient_CMDEvent().UnSub(demo.DESMQTTClient)

	/* DISCONNECT THE DESMQTTCLient */
	demo.DESMQTTClient_Disconnect()

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

			adm := &Admin{}
			if err := json.Unmarshal(msg.Payload(), adm); err != nil {
				pkg.Trace(err)
			}

			/* SIMULATE HAVING DONE SOMETHING */
			time.Sleep(time.Millisecond * 500)
			adm.AdmTime = time.Now().UTC().UnixMilli()

			demo.MQTTPublication_DemoDeviceClient_SIGAdmin(adm)
		},
	}
}

/* SUBSCRIPTIONS -> CONFIGURATION -> UPON RECEIPT, REPLY TO .../cmd/config */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDConfig() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDConfig(),
		Handler: func(c phao.Client, msg phao.Message) {

			mqtt := &Config{}
			if err := json.Unmarshal(msg.Payload(), mqtt); err != nil {
				pkg.Trace(err)
			}

			/* SIMULATE HAVING DONE SOMETHING */
			time.Sleep(time.Millisecond * 500)
			mqtt.CfgTime = time.Now().UTC().UnixMilli()

			demo.MQTTPublication_DemoDeviceClient_SIGConfig(mqtt)
		},
	}
}

/* SUBSCRIPTIONS -> CONFIGURATION -> UPON RECEIPT, REPLY TO .../cmd/config */
func (demo *DemoDeviceClient) MQTTSubscription_DemoDeviceClient_CMDEvent() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: demo.MQTTTopic_CMDEvent(),
		Handler: func(c phao.Client, msg phao.Message) {

			mqtt := &Event{}
			if err := json.Unmarshal(msg.Payload(), mqtt); err != nil {
				pkg.Trace(err)
			}

			/* SIMULATE HAVING DONE SOMETHING */
			time.Sleep(time.Millisecond * 500)
			mqtt.EvtTime = time.Now().UTC().UnixMilli()

			demo.MQTTPublication_DemoDeviceClient_SIGEvent(mqtt)
		},
	}
}

/*
PUBLICATIONS
*/
/* PUBLICATION -> CONFIG -> SIMULATED CONFIGS */
func (demo *DemoDeviceClient) MQTTPublication_DemoDeviceClient_SIGAdmin(adm *Admin) bool {
	return (pkg.MQTTPublication{

		Topic:    demo.MQTTTopic_SIGAdmin(),
		Message:  pkg.MakeMQTTMessage(adm.FilterAdmRecord()),
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
func (demo *DemoDeviceClient) Demo_Run_Sim() {
	fmt.Printf("\n (demo *DemoDeviceClient) Demo_Run_Sim( ): demo.Sim.Dur %d\n", demo.Sim.Dur)
	fmt.Printf("\n (demo *DemoDeviceClient) Demo_Run_Sim( ): demo.Sim.Qty %d\n", demo.Sim.Qty)
	t0 := time.Now()
	i := 1
	for i < demo.Sim.Qty {

		mqtts := Demo_Make_Sim_Sample(t0, time.Now(), demo.DESJob.DESJobName)

		demo.MQTTPublication_DemoDeviceClient_SIGSample(&mqtts)

		time.Sleep(time.Millisecond * time.Duration(demo.Sim.Dur))
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
			Data: []pkg.TSDPoint{{X: tumic, Y: YSinX(t0, ti, 99.9, 0.1)}},
			Min:  90,
			Max:  110,
		},
		{ // high_flow
			Data: []pkg.TSDPoint{{X: tumic, Y: YCosX(t0, ti, 2.1, 0.3)}},
			Min:  0,
			Max:  1,
		},
		{ // low_flow
			Data: []pkg.TSDPoint{{X: tumic, Y: YSinX(t0, ti, 1.9, 0.5)}},
			Min:  0,
			Max:  1,
		},
		{ // pressure
			Data: []pkg.TSDPoint{{X: tumic, Y: YCosX(t0, ti, 599.9, 0.7)}},
			Min:  0,
			Max:  1,
		},
		{ // battery_current
			Data: []pkg.TSDPoint{{X: tumic, Y: YSinX(t0, ti, 0.349, 0.9)}},
			Min:  0,
			Max:  1.5,
		},
		{ // battery_voltage
			Data: []pkg.TSDPoint{{X: tumic, Y: YCosX(t0, ti, 13.9, 1.1)}},
			Min:  0,
			Max:  15,
		},
		{ // motor_voltage
			Data: []pkg.TSDPoint{{X: tumic, Y: YSinX(t0, ti, 12.9, 1.3)}},
			Min:  0,
			Max:  15,
		},
		{ // valve_target
			Data: []pkg.TSDPoint{{X: tumic, Y: 4}},
			Min:  0,
			Max:  10,
		},
		{ // valve_position
			Data: []pkg.TSDPoint{{X: tumic, Y: 4}},
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

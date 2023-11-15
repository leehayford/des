package c001v001

import (
	"encoding/json"
	"fmt"
	"time"

	"net/url"

	phao "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofiber/websocket/v2"

	"github.com/leehayford/des/pkg"
)

/*
	MQTT DEVICE USER CLIENT

SUBSCRIBES TO ALL SIGNALS FOR A SINGLE DEVICE
  - SENDS LIVE DATA TO A SINGLE USER WSMessage
*/
type DeviceUserClient struct {
	Device
	WSClientID string
	pkg.DESMQTTClient
	DataOut chan string
	Close   chan struct{}
	Kill    chan struct{}
}

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type AuthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (duc DeviceUserClient) WSDeviceUserClient_Connect(c *websocket.Conn) {
	fmt.Printf("\nWSDeviceUserClient_Connect( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" && role != "viewer" {
		/* CREATE JSON WSMessage STRUCT */
		res := AuthResponse{
			Status:  "fail",
			Message: "You need permission to watch a live feed.",
		}
		js, err := json.Marshal(&WSMessage{Type: "auth", Data: res})
		if err != nil {
			pkg.LogErr(err)
			return
		}
		c.Conn.WriteJSON(string(js))
		return
	}

	des_regStr, _ := url.QueryUnescape(c.Query("des_reg"))

	des_reg := pkg.DESRegistration{}
	if err := json.Unmarshal([]byte(des_regStr), &des_reg); err != nil {
		pkg.LogErr(err)
	}
	des_reg.DESDevRegAddr = c.RemoteAddr().String()
	des_reg.DESJobRegAddr = c.RemoteAddr().String()

	wscid := fmt.Sprintf("%d-%s",
		time.Now().UTC().UnixMilli()/10,
		des_reg.DESDevSerial,
	) // fmt.Printf("WSDeviceUserClient_Connect -> wscid: %s\n", wscid)

	duc = DeviceUserClient{
		Device: Device{
			DESRegistration: des_reg,
		},
		WSClientID: wscid,
	} // fmt.Printf("\nHandle_ConnectDeviceUser(...) -> duc: %v\n\n", duc)

	duc.DataOut = make(chan string)
	duc.Close = make(chan struct{})
	duc.Kill = make(chan struct{})

	duc.MQTTDeviceUserClient_Connect()

	/* LISTEN FOR MESSAGES FROM CONNECTED USER */
	go func() {
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				fmt.Printf("WSDeviceUserClient_Connect -> c.ReadMessage() %s\n ERROR:\n%s\n", duc.DESDevSerial, err.Error())
				pkg.LogErr(err)
				break
			}
			if string(msg) == "close" {
				/* USER HAS CLOSED THE CONNECTION */
				fmt.Printf("WSDeviceUserClient_Connect -> go func() -> c.ReadMessage(): %s\n", string(msg))
				duc.MQTTDeviceUserClient_Disconnect()
				duc.Close <- struct{}{}
				break
			}
		}
		fmt.Printf("WSDeviceUserClient_Connect -> go func() done\n")
	}()

	/* KEEP ALIVE GO ROUTINE SEND "live" EVERY 30 SECONDS TO PREVENT DISCONNECT */
	live := true
	go func() {
		for live {
			select {

			case <-duc.Kill:
				live = false

			default:
				time.Sleep(time.Second * 30)
				js, err := json.Marshal(&WSMessage{Type: "live", Data: ""})
				if err != nil {
					pkg.LogErr(err)
				}
				duc.DataOut <- string(js)
				// fmt.Printf("WSDeviceUserClient_Connect -> go func() KEEP ALIVE... \n")
			}
		}
	}()

	/* SEND MESSAGES TO CONNECTED USER */
	open := true
	for open {
		select {

		case <-duc.Close:
			duc.Kill <- struct{}{}
			open = false

		case data := <-duc.DataOut:
			if err := c.WriteJSON(data); err != nil {
				pkg.LogErr(err)
				duc.MQTTDeviceUserClient_Disconnect()
				duc.Close <- struct{}{}
			}

		}
	}
	close(duc.Close)
	duc.Close = nil

	close(duc.Kill)
	duc.Kill = nil

	close(duc.DataOut)
	duc.DataOut = nil

	return
}

func (duc *DeviceUserClient) MQTTDeviceUserClient_Connect( /*user, pw string*/ ) (err error) {

	/* TODO: replace with user specific credentials */
	user := pkg.MQTT_USER
	pw := pkg.MQTT_PW

	duc.MQTTUser = user
	duc.MQTTPW = pw
	duc.MQTTClientID = duc.WSClientID
	/* DEVICE USER CLIENTS ***DO NOT*** AUTOMATICALLY RESUBSCRIBE */
	if err = duc.DESMQTTClient.DESMQTTClient_Connect(true, false); err != nil {
		return err
	}
	// pkg.Json(`(duc *DeviceUserClient) MQTTDeviceUserClient_Connect(...) -> duc.DESMQTTClient.DESMQTTClient_Connect()
	// duc.DESMQTTClient.ClientOptions.ClientID:`,
	// duc.DESMQTTClient.ClientOptions.ClientID)

	duc.MQTTSubscription_DeviceUserClient_SIGPing().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGAdmin().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGState().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGHeader().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGConfig().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGEvent().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGSample().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGDiagSample().Sub(duc.DESMQTTClient)

	fmt.Printf("\n(duc) MQTTDeviceUserClient_Connect( ) -> ClientID: %s\n", duc.ClientID)
	return err
}
func (duc *DeviceUserClient) MQTTDeviceUserClient_Disconnect() {

	/* UNSUBSCRIBE FROM ALL MQTTSubscriptions */
	duc.MQTTSubscription_DeviceUserClient_SIGPing().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGAdmin().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGState().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGHeader().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGConfig().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGEvent().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGSample().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGDiagSample().UnSub(duc.DESMQTTClient)

	/* DISCONNECT THE DESMQTTCLient */
	duc.DESMQTTClient_Disconnect()

	fmt.Printf("\n(duc) MQTTDeviceUserClient_Disconnect( ): Complete -> ClientID: %s\n", duc.ClientID)
}

/* SUBSCRIPTIONS ****************************************************************************************/

/* SUBSCRIPTIONS -> PING  */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGPing() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGPing(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* DECODE MESSAGE PAYLOAD TO Ping STRUCT */
			ping := Ping{}
			if err := json.Unmarshal(msg.Payload(), &ping); err != nil {
				pkg.LogErr(err)
			}

			/* CREATE JSON WSMessage STRUCT */
			js, err := json.Marshal(&WSMessage{Type: "ping", Data: ping})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGPing(...) -> ping :", ping)

			/* SEND WSMessage AS JSON STRING */
			duc.DataOut <- string(js)

		},
	}
}

/* SUBSCRIPTIONS -> ADMIN  */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGAdmin() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGAdmin(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* DECODE MESSAGE PAYLOAD TO Admin STRUCT */
			adm := Admin{}
			if err := json.Unmarshal(msg.Payload(), &adm); err != nil {
				pkg.LogErr(err)
			}

			/* CREATE JSON WSMessage STRUCT */
			js, err := json.Marshal(&WSMessage{Type: "admin", Data: adm})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGAdmin(...) -> adm :", adm)

			/* SEND WSMessage AS JSON STRING */
			duc.DataOut <- string(js)

		},
	}
}

/* SUBSCRIPTIONS -> HARDWARE ID  */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGState() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGState(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* DECODE MESSAGE PAYLOAD TO State STRUCT */
			sta := State{}
			if err := json.Unmarshal(msg.Payload(), &sta); err != nil {
				pkg.LogErr(err)
			}

			/* CREATE JSON WSMessage STRUCT */
			js, err := json.Marshal(&WSMessage{Type: "state", Data: sta})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGState(...) -> sta :", sta)

			/* SEND WSMessage AS JSON STRING */
			duc.DataOut <- string(js)

		},
	}
}

/* SUBSCRIPTION -> HEADER   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGHeader() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGHeader(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* DECODE MESSAGE PAYLOAD TO Header STRUCT */
			hdr := Header{}
			if err := json.Unmarshal(msg.Payload(), &hdr); err != nil {
				pkg.LogErr(err)
			}

			/* CREATE JSON WSMessage STRUCT */
			js, err := json.Marshal(&WSMessage{Type: "header", Data: hdr})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGHeader(...) -> hdr :", hdr)

			/* SEND WSMessage AS JSON STRING */
			duc.DataOut <- string(js)

		},
	}
}

/* SUBSCRIPTION -> CONFIG */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGConfig() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGConfig(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* DECODE MESSAGE PAYLOAD TO Config STRUCT */
			cfg := Config{}
			if err := json.Unmarshal(msg.Payload(), &cfg); err != nil {
				pkg.LogErr(err)
			}

			/* CREATE JSON WSMessage STRUCT */
			js, err := json.Marshal(&WSMessage{Type: "config", Data: cfg})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGConfig(...) -> cfg :", cfg)

			/* SEND WSMessage AS JSON STRING */
			duc.DataOut <- string(js)

		},
	}
}

/* SUBSCRIPTION -> EVENT */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGEvent() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGEvent(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* DECODE MESSAGE PAYLOAD TO Event STRUCT */
			evt := Event{}
			if err := json.Unmarshal(msg.Payload(), &evt); err != nil {
				pkg.LogErr(err)
			}

			/* CREATE JSON WSMessage STRUCT */
			js, err := json.Marshal(&WSMessage{Type: "event", Data: evt})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGEvent(...) -> evt :", evt)

			/* SEND WSMessage AS JSON STRING */
			duc.DataOut <- string(js)

		},
	}
}

/* SUBSCRIPTION -> SAMPLE   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGSample() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGSample(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* DECODE THE PAYLOAD INTO AN MQTT_Sample */
			mqtts := MQTT_Sample{}
			if err := json.Unmarshal(msg.Payload(), &mqtts); err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGSample(...) ->  mqtts :", mqtts)

			/* CREATE Sample STRUCT INTO WHICH WE'LL DECODE THE MQTT_Sample  */
			smp := &Sample{SmpJobName: mqtts.DesJobName}

			/* DECODE BASE64URL STRING ( DATA ) */
			if err := smp.DecodeMQTTSample(mqtts.Data); err != nil {
				pkg.LogErr(err)
			}

			/* CREATE JSON WSMessage STRUCT */
			js, err := json.Marshal(&WSMessage{Type: "sample", Data: smp})
			if err != nil {
				pkg.LogErr(err)
			} else {
				// pkg.Json("MQTTSubscription_DeviceUserClient_SIGSample:", js)
				/* SEND WSMessage AS JSON STRING */
				duc.DataOut <- string(js)
			}

		},
	}
}

/* SUBSCRIPTION -> DIAG SAMPLE   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGDiagSample() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGDiagSample(),
		Handler: func(c phao.Client, msg phao.Message) {
			/* WRANGLE WS DATA */
			/* SEND WS DATA */
			duc.DataOut <- "diag_sample data..."
		},
	}
}

/* PUBLICATIONS ******************************************************************************************/
/* NONE; WE DON'T DO THAT;
ALL COMMANDS SENT TO THE DEVICE GO THROUGH HTTP HANDLERS
*/

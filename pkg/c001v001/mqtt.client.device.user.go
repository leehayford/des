package c001v001

import (
	"context"
	"encoding/json"
	"fmt"
	// "strings"
	"time"

	"net/url"

	phao "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofiber/contrib/websocket"

	"github.com/leehayford/des/pkg"
)

type DeviceUserClient struct {
	Device
	outChan chan string
	WSClientID string
	CTX        context.Context
	Cancel     context.CancelFunc
	pkg.DESMQTTClient
}

func (duc DeviceUserClient) WSDeviceUserClient_Connect(c *websocket.Conn) {

	fmt.Println("\nWSDeviceUserClient_Connect( )...")

	des_regStr, _ := url.QueryUnescape(c.Query("des_reg"))

	des_reg := pkg.DESRegistration{}
	if err := json.Unmarshal([]byte(des_regStr), &des_reg); err != nil {
		pkg.TraceErr(err)
	}
	des_reg.DESDevRegAddr = c.RemoteAddr().String()
	des_reg.DESJobRegAddr = c.RemoteAddr().String()

	wscid := fmt.Sprintf("%d-%s",
		// strings.Split(des_reg.DESJobRegUserID, "-")[4],
		time.Now().UTC().UnixMilli() / 10,
		des_reg.DESDevSerial,
	) // fmt.Printf("WSDeviceUserClient_Connect -> wscid: %s\n", wscid)

	duc = DeviceUserClient{
		Device: Device{
			DESRegistration: des_reg,
			Job:             Job{DESRegistration: des_reg},
		},
		WSClientID: wscid,
	} // fmt.Printf("\nHandle_ConnectDeviceUser(...) -> duc: %v\n\n", duc)

	duc.outChan = make(chan string)
	defer func() {
		close(duc.outChan)
		duc.outChan = nil
	}()

	duc.MQTTDeviceUserClient_Connect()

	open := true
	go func() {
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				fmt.Printf("WSDeviceUserClient_Connect -> c.ReadMessage() %s\n ERROR:\n%s\n", duc.DESDevSerial, err.Error())
				break
			}
			if string(msg) == "close" {
				fmt.Printf("WSDeviceUserClient_Connect -> go func() -> c.ReadMessage(): %s\n", string(msg))
				duc.MQTTDeviceUserClient_Disconnect()
				open = false
				break
			}
		}
		fmt.Printf("WSDeviceUserClient_Connect -> go func() done\n")
	}()

	go func() {
		for open {
			time.Sleep(time.Second * 30)
			js, err := json.Marshal(&WSMessage{Type: "live", Data: ""})
			if err != nil {
				pkg.TraceErr(err)
			}
			duc.outChan <- string(js)
		}
	}()

	for open {
		select {

		case data := <-duc.outChan:
			if err := c.WriteJSON(data); err != nil {
				pkg.TraceErr(err)
				duc.MQTTDeviceUserClient_Disconnect()
			}

		}
	}
	return
}


/*
	MQTT DEVICE USER CLIENT

SUBSCRIBES TO ALL SIGNALS FOR A SINGLE DEVICE
  - SENDS LIVE DATA TO A SINGLE USER UI VIA SSE
*/
func (duc *DeviceUserClient) MQTTDeviceUserClient_Connect( /*user, pw string*/ ) (err error) {

	/* TODO: replace with user specific credentials */
	user := pkg.MQTT_USER
	pw := pkg.MQTT_PW

	duc.MQTTUser = user
	duc.MQTTPW = pw
	duc.MQTTClientID = duc.WSClientID
	if err = duc.DESMQTTClient.DESMQTTClient_Connect(true); err != nil {
		return err
	}
	// pkg.Json(`(duc *DeviceUserClient) MQTTDeviceUserClient_Connect(...) -> duc.DESMQTTClient.DESMQTTClient_Connect()
	// duc.DESMQTTClient.ClientOptions.ClientID:`,
	// duc.DESMQTTClient.ClientOptions.ClientID)

	duc.MQTTSubscription_DeviceUserClient_SIGAdmin().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGHeader().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGConfig().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGEvent().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGSample().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGDiagSample().Sub(duc.DESMQTTClient)

	pkg.MQTTUserClients[duc.WSClientID] = duc.DESMQTTClient
	// userClient := pkg.MQTTUserClients[duc.WSClientID]
	// fmt.Printf("\n%s client ID: %s\n", duc.WSClientID, userClient.MQTTClientID)

	fmt.Printf("\n(duc) MQTTDeviceUserClient_Connect( ) -> ClientID: %s\n", duc.ClientID)
	return err
}
func (duc *DeviceUserClient) MQTTDeviceUserClient_Disconnect() {

	/* UNSUBSCRIBE FROM ALL MQTTSubscriptions */
	duc.MQTTSubscription_DeviceUserClient_SIGAdmin().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGHeader().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGConfig().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGEvent().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGSample().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGDiagSample().UnSub(duc.DESMQTTClient)

	/* DISCONNECT THE DESMQTTCLient */
	duc.DESMQTTClient_Disconnect()

	delete(pkg.MQTTUserClients, duc.WSClientID)

	fmt.Printf("\n(duc) MQTTDeviceUserClient_Disconnect( ): Complete -> ClientID: %s\n", duc.ClientID)
}

/*
SUBSCRIPTIONS
*/

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

/* SUBSCRIPTIONS -> ADMINISTRATION   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGAdmin() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGAdmin(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* WRANGLE WS DATA */
			adm := Admin{}
			if err := json.Unmarshal(msg.Payload(), &adm); err != nil {
				pkg.TraceErr(err)
			}

			js, err := json.Marshal(&WSMessage{Type: "admin", Data: adm})
			if err != nil {
				pkg.TraceErr(err)
			}
			// pkg.Json("MQTTSubscription_DeviceUserClient_SIGAdmin(...) -> adm :", adm)

			/* SEND WS DATA */
			duc.outChan <- string(js)

		},
	}
}

/* SUBSCRIPTION -> HEADER   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGHeader() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGHeader(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* WRANGLE WS DATA */
			hdr := Header{}
			if err := json.Unmarshal(msg.Payload(), &hdr); err != nil {
				pkg.TraceErr(err)
			}

			js, err := json.Marshal(&WSMessage{Type: "header", Data: hdr})
			if err != nil {
				pkg.TraceErr(err)
			}
			// pkg.Json("MQTTSubscription_DeviceUserClient_SIGHeader(...) -> hdr :", hdr)

			/* SEND WS DATA */
			duc.outChan <- string(js)

		},
	}
}

/* SUBSCRIPTION -> CONFIGURATION   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGConfig() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGConfig(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* WRANGLE WS DATA */
			cfg := Config{}
			if err := json.Unmarshal(msg.Payload(), &cfg); err != nil {
				pkg.TraceErr(err)
			}

			js, err := json.Marshal(&WSMessage{Type: "config", Data: cfg})
			if err != nil {
				pkg.TraceErr(err)
			}
			// pkg.Json("MQTTSubscription_DeviceUserClient_SIGConfig(...) -> cfg :", cfg)

			/* SEND WS DATA */
			duc.outChan <- string(js)

		},
	}
}

/* SUBSCRIPTION -> EVENT   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGEvent() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGEvent(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* WRANGLE WS DATA */
			evt := Event{}
			if err := json.Unmarshal(msg.Payload(), &evt); err != nil {
				pkg.TraceErr(err)
			}

			js, err := json.Marshal(&WSMessage{Type: "event", Data: evt})
			if err != nil {
				pkg.TraceErr(err)
			}
			// pkg.Json("MQTTSubscription_DeviceUserClient_SIGEvent(...) -> evt :", evt)

			/* SEND WS DATA */
			duc.outChan <- string(js)

		},
	}
}

/* SUBSCRIPTION -> SAMPLE   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGSample() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGSample(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* WRANGLE AND SEND WS DATA */
			// Decode the payload into an MQTT_Sample
			mqtts := MQTT_Sample{}
			if err := json.Unmarshal(msg.Payload(), &mqtts); err != nil {
				pkg.TraceErr(err)
			} // pkg.Json("DecodeMQTTSampleMessage(...) ->  msg :", msg)

			for _, b64 := range mqtts.Data {

				// Decode base64 string
				sample := Sample{SmpJobName: mqtts.DesJobName}
				if err := duc.Job.DecodeMQTTSample(b64, &sample); err != nil {
					pkg.TraceErr(err)
				}

				// Create a JSON version thereof
				js, err := json.Marshal(&WSMessage{Type: "sample", Data: sample})
				if err != nil {
					pkg.TraceErr(err)
				} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGSample:", js)
				// Ship it
				duc.outChan <- string(js)

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
			duc.outChan <- "diag_sample data..."
		},
	}
}

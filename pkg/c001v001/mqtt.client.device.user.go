
package c001v001

import (
	"context"
	"encoding/json"
	"fmt"

	"net/url"
	"time"

	"github.com/gofiber/contrib/websocket"
	phao "github.com/eclipse/paho.mqtt.golang"

	"github.com/leehayford/des/pkg"
)

type DeviceUserClient struct {
	Device
	adminChan   chan string
	configChan  chan string
	eventChan   chan string
	sampleChan  chan string
	diagChan    chan string
	WSClientID string
	CTX    context.Context
	Cancel context.CancelFunc
	pkg.DESMQTTClient
}

func (duc DeviceUserClient) WSDeviceUserClient_Connect(c *websocket.Conn) {
	fmt.Println("WSDeviceUserClient_Connect( ... ): So far, so good...")

	des_regStr, _ := url.QueryUnescape(c.Query("des_reg"))

	des_reg := pkg.DESRegistration{}
	if err := json.Unmarshal([]byte(des_regStr), &des_reg); err != nil {
		pkg.Trace(err)
	}
	des_reg.DESDevRegAddr = c.RemoteAddr().String()
	des_reg.DESJobRegAddr = c.RemoteAddr().String()

	wscid := fmt.Sprintf("%s-%s-%s",
		c.RemoteAddr().String(),
		des_reg.DESDevRegUserID,
		des_reg.DESJobName,
	)  // fmt.Printf("WSDeviceUserClient_Connect -> wscid: %s\n", wscid)

	duc = DeviceUserClient{
		Device:      Device{ 
			DESRegistration: des_reg, 
			Job:	Job{ DESRegistration: des_reg },
		},
		WSClientID: wscid,
	}  // fmt.Printf("\nHandle_ConnectDeviceUser(...) -> duc: %v\n\n", duc)

	duc.adminChan = make(chan string)
	defer func() {
		close(duc.adminChan)
		duc.adminChan = nil
	}()
	duc.configChan = make(chan string)
	defer func() {
		close(duc.configChan)
		duc.configChan = nil
	}()
	duc.eventChan = make(chan string)
	defer func() {
		close(duc.eventChan)
		duc.eventChan = nil
	}()
	duc.sampleChan = make(chan string)
	defer func() {
		close(duc.sampleChan)
		duc.sampleChan = nil
	}()
	duc.diagChan = make(chan string)
	defer func() {
		close(duc.diagChan)
		duc.diagChan = nil
	}()

	duc.MQTTDeviceUserClient_Connect()

	open := true
	go func() {  
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				fmt.Printf("WSDeviceUserClient_Connect -> c.ReadMessage() ERROR:\n%s", err.Error())
				break
			}
			if (string(msg) == "close") {
				fmt.Printf("WSDeviceUserClient_Connect -> go func() -> c.ReadMessage(): %s\n", string(msg))
				duc.MQTTDeviceUserClient_Disconnect()
				open = false
				break
			}
		}
		fmt.Printf("WSDeviceUserClient_Connect -> go func() done\n")
	}()

	for open {
		select {

		case admin := <-duc.adminChan:
			c.WriteJSON(admin)

		case config := <-duc.configChan:
			c.WriteJSON(config)

		case event := <-duc.eventChan:
			c.WriteJSON(event)

		case sample := <-duc.sampleChan:
			if err := c.WriteJSON(sample); err != nil {
				pkg.Trace(err)
			}

		case diag := <-duc.diagChan:
			c.WriteJSON(diag)
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
	duc.MQTTClientID = fmt.Sprintf(
		"DeviceUser-%s-%s-%s",
		duc.DESDevClass,
		duc.DESDevVersion,
		duc.DESDevSerial,
	)
	if err = duc.DESMQTTClient.DESMQTTClient_Connect(); err != nil {
		return err
	}
	// pkg.Json(`(duc *DeviceUserClient) MQTTDeviceUserClient_Connect(...) -> duc.DESMQTTClient.DESMQTTClient_Connect()
	// duc.DESMQTTClient.ClientOptions.ClientID:`,
	// duc.DESMQTTClient.ClientOptions.ClientID)

	duc.MQTTSubscription_DeviceUserClient_SIGAdmin().Sub(duc.DESMQTTClient)

	duc.MQTTSubscription_DeviceUserClient_SIGConfig().Sub(duc.DESMQTTClient)

	duc.MQTTSubscription_DeviceUserClient_SIGEvent().Sub(duc.DESMQTTClient)

	duc.MQTTSubscription_DeviceUserClient_SIGSample().Sub(duc.DESMQTTClient)

	duc.MQTTSubscription_DeviceUserClient_SIGDiagSample().Sub(duc.DESMQTTClient)

	return err
}
func (duc *DeviceUserClient) MQTTDeviceUserClient_Disconnect() {

	/* UNSUBSCRIBE FROM ALL MQTTSubscriptions */
	duc.MQTTSubscription_DeviceUserClient_SIGAdmin().UnSub(duc.DESMQTTClient)

	duc.MQTTSubscription_DeviceUserClient_SIGConfig().UnSub(duc.DESMQTTClient)

	duc.MQTTSubscription_DeviceUserClient_SIGEvent().UnSub(duc.DESMQTTClient)

	duc.MQTTSubscription_DeviceUserClient_SIGSample().UnSub(duc.DESMQTTClient)

	duc.MQTTSubscription_DeviceUserClient_SIGDiagSample().UnSub(duc.DESMQTTClient)

	/* DISCONNECT THE DESMQTTCLient */
	duc.DESMQTTClient_Disconnect()

	/* ENSURE ALL SSE MESSAGES HAVE CLEARED BEFORE CLOSING CHANELS*/
	// time.Sleep(time.Second * 3 ) 

	// close(duc.adminChan)
	// duc.adminChan = nil

	// close(duc.configChan)
	// duc.configChan = nil

	// close(duc.eventChan)
	// duc.eventChan = nil
	
	// close(duc.sampleChan)
	// duc.sampleChan = nil

	// close(duc.diagChan)
	// duc.diagChan = nil

	fmt.Printf("(duc *DeviceUserClient) MQTTDeviceUserClient_Disconnect( ... ): Complete.\n")
}


/*
SUBSCRIPTIONS
*/

/* SUBSCRIPTIONS -> ADMINISTRATION   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGAdmin() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGAdmin(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* WRANGLE SSE DATA */
			adm := Admin{}
			if err := json.Unmarshal(msg.Payload(), &adm); err != nil {
				pkg.Trace(err)
			}
			time.Sleep(time.Millisecond * 300) // wait for DB write to complete

			db := duc.JDB()
			db.Connect()
			defer db.Close()
			db.Where("adm_time = ?", adm.AdmTime).First(&adm)
			db.Close()
			js, err := json.Marshal(adm)
			if err != nil {
				pkg.Trace(err)
			}
			pkg.Json("(adm *Admin) SendSSEAdmin(...) -> adm :", adm)

			/* SEND SSE DATA */
			duc.adminChan <- string(js)

		},
	}
}

/* SUBSCRIPTION -> CONFIGURATION   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGConfig() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGConfig(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* WRANGLE SSE DATA */
			cfg := Config{}
			if err := json.Unmarshal(msg.Payload(), &cfg); err != nil {
				pkg.Trace(err)
			}
			time.Sleep(time.Millisecond * 300) // wait for DB write to complete

			db := duc.JDB()
			db.Connect()
			defer db.Close()
			db.Where("cfg_time = ?", cfg.CfgTime).First(&cfg)
			db.Close()
			js, err := json.Marshal(cfg)
			if err != nil {
				pkg.Trace(err)
			}
			pkg.Json("(cfg *Config) SendSSEAdmin(...) -> cfg :", cfg)

			/* SEND SSE DATA */
			duc.configChan <- string(js)

		},
	}
}

/* SUBSCRIPTION -> EVENT   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGEvent() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGEvent(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* WRANGLE SSE DATA */
			evt := Event{}
			if err := json.Unmarshal(msg.Payload(), &evt); err != nil {
				pkg.Trace(err)
			}
			time.Sleep(time.Millisecond * 300) // wait for DB write to complete

			db := duc.JDB()
			db.Connect()
			defer db.Close()
			db.Where("evt_time = ?", evt.EvtTime).First(&evt)
			db.Close()
			js, err := json.Marshal(evt)
			if err != nil {
				pkg.Trace(err)
			}
			pkg.Json("(cfg *Event) SendSSEAdmin(...) -> evt :", evt)

			/* SEND SSE DATA */
			duc.eventChan <- string(js)

		},
	}
}

/* SUBSCRIPTION -> SAMPLE   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGSample() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGSample(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* WRANGLE AND SEND SSE DATA */
			// Decode the payload into an MQTT_Sample
			mqtts := MQTT_Sample{}
			if err := json.Unmarshal(msg.Payload(), &mqtts); err != nil {
				pkg.Trace(err)
			} // pkg.Json("DecodeMQTTSampleMessage(...) ->  msg :", msg)

			for _, b64 := range mqtts.Data {

				// Decode base64 string
				sample := &Sample{SmpJobName: mqtts.DesJobName}
				if err := duc.Job.DecodeMQTTSample(b64, sample); err != nil {
					pkg.Trace(err)
				}

				// Create a JSON version thereof
				js, err := json.Marshal(&SSESample{Type: "sample",Data: *sample,})
				if err != nil {
					pkg.Trace(err)
				} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGSample:", js)
				// Ship it
				duc.sampleChan <- string(js)

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
			/* WRANGLE SSE DATA */
			/* SEND SSE DATA */
			duc.diagChan <- "diag_sample data..."
		},
	}
}


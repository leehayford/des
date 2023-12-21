package c001v001

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

/* CONNECTED DEVICE USER CLIENT *** DO NOT RUN IN GO ROUTINE *** */
func (duc *DeviceUserClient) DeviceUserClient_Connect(c *websocket.Conn, sid string) {

	// duc.WSClientID = fmt.Sprintf("%d-%s", time.Now().UTC().UnixMilli()/10, duc.DESDevSerial)
	sid_node := strings.Split(sid, "-")[4]
	duc.WSClientID = fmt.Sprintf("%s-%s",sid_node, duc.DESDevSerial)
	duc.DataOut = make(chan string)
	duc.Close = make(chan struct{})
	duc.Kill = make(chan struct{})

	duc.MQTTDeviceUserClient_Connect( /* TODO: PASS IN USER ROLE */ )

	/* LISTEN FOR MESSAGES FROM CONNECTED USER */
	go duc.ListenForMessages(c)

	/* KEEP ALIVE GO ROUTINE SEND "live" EVERY 30 SECONDS TO PREVENT DISCONNECT */
	go duc.RunKeepAlive()

	/* UPDATE USER WITH DEVICE & DES CONNECTION STATUS */
	go duc.GetPingsOnConnect()

	/* *** DO NOT RUN IN GO ROUTINE *** SEND MESSAGES TO CONNECTED USER */
	duc.SendMessages(c)
}

/* SEND MESSAGES TO CONNECTED USER */
func (duc DeviceUserClient) SendMessages(c *websocket.Conn) {
	open := true
	for open {
		select {

		case <-duc.Close:
			duc.Kill <- struct{}{}
			open = false

		case data := <-duc.DataOut:
			if err := c.WriteJSON(data); err != nil {
				// pkg.LogErr(err)
				// fmt.Printf("(DeviceUserClient) SendMessages -> data := <-duc.DataOut: %s\n", string(data))
				duc.MQTTDeviceUserClient_Disconnect()
				duc.Close <- struct{}{}
			}
		}
	}

	if duc.Close != nil {
		close(duc.Close)
		duc.Close = nil
	}

	if duc.Kill != nil {
		close(duc.Kill)
		duc.Kill = nil
	}

	if duc.DataOut != nil {
		close(duc.DataOut)
		duc.DataOut = nil
	}
	return
}

/* LISTEN FOR MESSAGES FROM CONNECTED USER */
func (duc DeviceUserClient) ListenForMessages(c *websocket.Conn) {
	listen := true
	for listen {
		_, msg, err := c.ReadMessage()
		if err != nil {
			// fmt.Printf("WSDeviceUserClient_Connect -> c.ReadMessage() %s -> ERROR: %s\n", duc.DESDevSerial, err.Error())
			// pkg.LogErr(err)
			break
		}
		if string(msg) == "close" {
			/* USER HAS CLOSED THE CONNECTION */
			fmt.Printf("WSDeviceUserClient_Connect -> go func() -> c.ReadMessage(): %s\n", string(msg))
			duc.MQTTDeviceUserClient_Disconnect()
			duc.Close <- struct{}{}
			listen = false
		}
	}
	fmt.Printf("WSDeviceUserClient_Connect -> go func() done\n")
}

/* KEEP ALIVE GO ROUTINE SEND "live" EVERY 30 SECONDS TO PREVENT WS DISCONNECT */
func (duc DeviceUserClient) RunKeepAlive() {

	live := true
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
}

/* UPDATE USER WITH DEVICE & DES CONNECTION STATUS AS OD WS CONNECT */
func (duc DeviceUserClient) GetPingsOnConnect() {
	/* WHEN CALLED FROM DeviceUserClient_Connect, 
	WE WANT TO ENSURE duc.SendMessages HAS BEEN STARTED*/
	time.Sleep(time.Second * 2)

	/* GET LAST DES CLIENT DEVICE PING FROM MAP */
	des_ping := DESDeviceClientPingsMapRead(duc.DESDevSerial)
	// pkg.Json("(DeviceUserClient) GetPingsOnConnect(...) -> des_ping :", des_ping)

	/* CHECK PING TIME */
	if (des_ping.OK && des_ping.Time + DES_PING_LIMIT < time.Now().UTC().UnixMilli()) {
		des_ping.OK = false
		DESDeviceClientPingsMapWrite(duc.DESDevSerial, des_ping)
	}

	/* CREATE WSMessage */
	des_ping_js, err := json.Marshal(&WSMessage{Type: "des_ping", Data: des_ping})
	if err != nil {
		pkg.LogErr(err)
	}  // pkg.Json("(DeviceUserClient) GetPingsOnConnect(...) -> des_ping_js :", des_ping_js)
	
	/* SEND WSMessage AS JSON STRING */
	duc.DataOut <- string(des_ping_js)
	

	/* GET LAST DEVICE PING FROM MAP */
	device_ping := DevicePingsMapRead(duc.DESDevSerial)
	// pkg.Json("(DeviceUserClient) GetPingsOnConnect(...) -> device_ping :", device_ping)

	/* CHECK PING TIME */
	if (device_ping.OK && device_ping.Time  + DEVICE_PING_LIMIT < time.Now().UTC().UnixMilli()) {
		device_ping.OK = false
		DevicePingsMapWrite(duc.DESDevSerial, device_ping)
	}

	/* CREATE WSMessage */
	device_ping_js, err := json.Marshal(&WSMessage{Type: "ping", Data: device_ping})
	if err != nil {
		pkg.LogErr(err)
	} // pkg.Json("(DeviceUserClient) GetPingsOnConnect(...) -> device_ping_js :", device_ping_js)

	/* SEND WSMessage AS JSON STRING */
	duc.DataOut <- string(device_ping_js)
}

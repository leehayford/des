package c001v001

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"     // go get github.com/google/uuid

	"github.com/leehayford/des/pkg"
)

const DEVICE_USER_CLIENT_WS_KEEP_ALIVE_SEC = 30
/*
	MQTT DEVICE USER CLIENT

SUBSCRIBES TO ALL SIGNALS FOR A SINGLE DEVICE
  - SENDS LIVE DATA TO A SINGLE USER WSMessage
*/
type DeviceUserClient struct {
	Device `json:"device"`
	SID         uuid.UUID     `json:"sid"`
	MQTTClientID string `json:"mqtt_id"`
	pkg.DESMQTTClient `json:"-"`
	DataOut     chan string   `json:"-"`
	Close       chan struct{} `json:"-"`
	CloseSend   chan struct{} `json:"-"`
	CloseKeep   chan struct{} `json:"-"`
}

type DeviceUserClientMap map[string]DeviceUserClient

var DeviceUserClientsMap = make(DeviceUserClientMap)
var DeviceUserClientMapRWMutex = sync.RWMutex{}

func DeviceUserClientsMapWrite(duc DeviceUserClient) (err error) {

	sid := duc.SID.String()
	if !pkg.ValidateUUIDString(sid) {
		err = fmt.Errorf("Invalid user session ID.")
		return
	}

	DeviceUserClientMapRWMutex.Lock()
	DeviceUserClientsMap[duc.MQTTClientID] = duc
	DeviceUserClientMapRWMutex.Unlock()
	return
}
func DeviceUserClientsMapRead(mcid string) (duc DeviceUserClient, err error) {
	DeviceUserClientMapRWMutex.Lock()
	duc = DeviceUserClientsMap[mcid]
	DeviceUserClientMapRWMutex.Unlock()

	if !pkg.ValidateUUIDString(duc.SID.String()) {
		err = fmt.Errorf("User session not found. Please log in.")
	}
	return
}
func DeviceUserClientsMapCopy() (ducm DeviceUserClientMap) {
	DeviceUserClientMapRWMutex.Lock()
	ducm = DeviceUserClientsMap
	DeviceUserClientMapRWMutex.Unlock()
	return
}
func DeviceUserClientsMapRemove(mcid string) {
	DeviceUserClientMapRWMutex.Lock()
	delete(DeviceUserClientsMap, mcid)
	DeviceUserClientMapRWMutex.Unlock()
}

/* CONNECTED DEVICE USER CLIENT *** DO NOT RUN IN GO ROUTINE *** */
func (duc *DeviceUserClient) DeviceUserClient_Connect(ws *websocket.Conn, sid string) {

	start := time.Now().Unix()

	sid_node := strings.Split(sid, "-")[4]
	duc.MQTTClientID = fmt.Sprintf("%s-%s", sid_node, duc.DESDevSerial)

	duc.DataOut = make(chan string)
	duc.Close = make(chan struct{})
	duc.CloseSend = make(chan struct{})
	duc.CloseKeep = make(chan struct{})

	duc.MQTTDeviceUserClient_Connect( /* TODO: PASS IN USER ROLE */ )

	/* LISTEN FOR MESSAGES FROM CONNECTED DEVICE USER */
	go duc.ListenForMessages(ws, start)

	/* KEEP ALIVE GO ROUTINE SEND "live" EVERY 30 SECONDS TO PREVENT WS DISCONNECT */
	go duc.RunKeepAlive(start)

	/* SEND MESSAGES TO CONNECTED DEVICE USER */
	go duc.SendMessages(ws, start)

	/* UPDATE USER WITH DEVICE & DES  MQTT CONNECTION STATUS */
	go duc.GetPingsOnConnect()

	DeviceUserClientsMapWrite(*duc)

	// fmt.Printf("\n(*DeviceUserClient) DeviceUserClient_Connect() -> %s : %d -> OPEN.\n", duc.MQTTClientID, start)
	open := true
	for open {
		select {
		case <-duc.Close:
			duc.MQTTDeviceUserClient_Disconnect()
					
			/* WE WANT TO ENSURE  MQTT CLIENT IS DISCONNECTED
				BEFORE WE START CLOSING CHANELS */
			time.Sleep(time.Second * 1)
			if duc.CloseKeep != nil {
				duc.CloseKeep = nil
			}
			if duc.CloseSend != nil {
				duc.CloseSend = nil
			}
			if duc.DataOut != nil {
				duc.DataOut = nil
			}
			open = false
		}
	}
	DeviceUserClientsMapRemove(duc.MQTTClientID)
	// fmt.Printf("\n(*DeviceUserClient) DeviceUserClient_Connect() -> %s : %d -> CLOSED.\n", duc.MQTTClientID, start)
}

/* GO ROUTINE: LISTEN FOR MESSAGES FROM CONNECTED USER */
func (duc *DeviceUserClient) ListenForMessages(ws *websocket.Conn, start int64) {
	listen := true
	for listen {
		_, msg, err := ws.ReadMessage()
		if err != nil { 
			// fmt.Printf("\n(DeviceUserClient) ListenForMessages() %s -> ERROR: %s\n", duc.MQTTClientID, err.Error())
			if strings.Contains(err.Error(), "close") {
				msg = []byte("close")
			}
		}
		/* CHECK IF USER HAS CLOSED THE CONNECTION */
		if string(msg) == "close" { 
			duc.CloseKeep <- struct{}{}
			duc.CloseSend <- struct{}{}
			listen = false
		}
	} // fmt.Printf("\n(*DeviceUserClient) ListenForMessages() -> %s : %d -> DONE.\n", duc.MQTTClientID, start)
	duc.Close <- struct{}{}
}

/* GO ROUTINE: SEND WSMessage PERIODICALLY TO PREVENT WS DISCONNECT */
func (duc *DeviceUserClient) RunKeepAlive(start int64) {
	msg := fmt.Sprintf("%s : %d", duc.MQTTClientID, start)
	count := 0
	live := true
	for live {
		select {

		case <-duc.CloseKeep:
			live = false

		default:
			if count == DEVICE_USER_CLIENT_WS_KEEP_ALIVE_SEC {
				js, err := json.Marshal(&pkg.WSMessage{Type: "live", Data: msg})
				if err != nil {
					pkg.LogErr(err)
				}
				duc.DataOut <- string(js)
				count = 0
			}
			time.Sleep(time.Second * 1)
			count++ 
		}
	} // fmt.Printf("\n(*DeviceUserClient) RunKeepAlive() -> %s : %d -> DONE.\n", duc.MQTTClientID, start)
}

/* GO ROUTINE: SEND MESSAGES TO CONNECTED DEVICE USER */
func (duc *DeviceUserClient) SendMessages(ws *websocket.Conn, start int64) {
	send := true
	for send {
		select {

		case <-duc.CloseSend:
			send = false

		case data := <-duc.DataOut:
			if err := ws.WriteJSON(data); err != nil { 
				if !strings.Contains(err.Error(), "close sent") {
					pkg.LogErr(err) 
				}
			}
		}
	} // fmt.Printf("\n(*DeviceUserClient) SendMessages() -> %s : %d -> DONE.\n", duc.MQTTClientID, start)
}

/* UPDATE USER WITH DEVICE & DES CONNECTION STATUS AS OD WS CONNECT */
func (duc *DeviceUserClient) GetPingsOnConnect() {
	/* WHEN CALLED FROM DeviceUserClient_Connect,
	WE WANT TO ENSURE duc.SendMessages HAS BEEN STARTED*/
	time.Sleep(time.Second * 2)

	/* GET LAST DES CLIENT DEVICE PING FROM MAP */
	des_ping := DESDeviceClientPingsMapRead(duc.DESDevSerial)
	// pkg.Json("(*DeviceUserClient) GetPingsOnConnect(...) -> des_ping :", des_ping)

	/* CHECK PING TIME */
	if des_ping.OK && des_ping.Time+DES_PING_LIMIT < time.Now().UTC().UnixMilli() {
		des_ping.OK = false
		DESDeviceClientPingsMapWrite(duc.DESDevSerial, des_ping)
	}

	/* CREATE WSMessage */
	des_ping_js, err := json.Marshal(&pkg.WSMessage{Type: "des_ping", Data: des_ping})
	if err != nil {
		pkg.LogErr(err)
	} // pkg.Json("(*DeviceUserClient) GetPingsOnConnect(...) -> des_ping_js :", des_ping_js)

	/* SEND WSMessage AS JSON STRING */
	duc.DataOut <- string(des_ping_js)

	/* GET LAST DEVICE PING FROM MAP */
	device_ping := DevicePingsMapRead(duc.DESDevSerial)
	// pkg.Json("(*DeviceUserClient) GetPingsOnConnect(...) -> device_ping :", device_ping)

	/* CHECK PING TIME */
	if device_ping.OK && device_ping.Time+DEVICE_PING_LIMIT < time.Now().UTC().UnixMilli() {
		device_ping.OK = false
		DevicePingsMapWrite(duc.DESDevSerial, device_ping)
	}

	/* CREATE WSMessage */
	device_ping_js, err := json.Marshal(&pkg.WSMessage{Type: "ping", Data: device_ping})
	if err != nil {
		pkg.LogErr(err)
	} // pkg.Json("(*DeviceUserClient) GetPingsOnConnect(...) -> device_ping_js :", device_ping_js)

	/* SEND WSMessage AS JSON STRING */
	duc.DataOut <- string(device_ping_js)
}



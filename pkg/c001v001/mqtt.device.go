package c001v001

import (
	"encoding/json"
	"fmt"
	"time"

	phao "github.com/eclipse/paho.mqtt.golang"

	"github.com/leehayford/des/pkg"
)

/*
	 MQTT DEVICE CLIENT
		PUBLISHES ALL COMMANDS TO A SINGLE DEVICE
		SUBSCRIBES TO ALL SIGNALS FOR A SINGLE DEVICE
	  - WRITES MESSAGES TO THE CMD ARCHIVE AND ACTIVE JOB DATABASES
*/
func (device *Device) MQTTDeviceClient_Connect() (err error) {

	/* TODO: REPLACE WITH <Device.Class>-<Device.Version> SPECIFIC CREDENTIALS */
	class_version_user := pkg.MQTT_USER
	class_version_pw := pkg.MQTT_PW

	/* CREATE MQTT CLIENT ID; 23 CHAR MAXIMUM */
	device.MQTTUser = class_version_user
	device.MQTTPW = class_version_pw
	device.MQTTClientID = fmt.Sprintf(
		"%s-%s-%s-DES",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)

	/* CONNECT TO THE BROKER WITH 'CleanSession = false'
	AUTOMATICALLY RE-SUBSCRIBE ON RECONNECT AFTER */
	if err = device.DESMQTTClient.DESMQTTClient_Connect(false, true); err != nil {
		return err
	}

	/* SUBSCRIBE TO ALL MQTTSubscriptions */
	device.MQTTSubscription_DeviceClient_SIGStartJob().Sub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGEndJob().Sub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGDevicePing().Sub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGAdmin().Sub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGState().Sub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGHeader().Sub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGConfig().Sub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGEvent().Sub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGSample().Sub(device.DESMQTTClient)
	// device.MQTTSubscription_DeviceClient_SIGDiagSample() //.Sub(device.DESMQTTClient)

	return err
}
func (device *Device) MQTTDeviceClient_Disconnect() (err error) {

	/* UNSUBSCRIBE FROM ALL MQTTSubscriptions */
	device.MQTTSubscription_DeviceClient_SIGStartJob().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGEndJob().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGDevicePing().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGAdmin().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGState().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGHeader().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGConfig().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGEvent().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGSample().UnSub(device.DESMQTTClient)
	// device.MQTTSubscription_DeviceClient_SIGDiagSample() //.UnSub(device.DESMQTTClient)

	/* DISCONNECT THE DESMQTTCLient */
	if err = device.DESMQTTClient_Disconnect(); err != nil {
		pkg.LogErr(err)
	}

	fmt.Printf("\n(device) MQTTDeviceClient_Dicconnect( ) -> %s -> disconnected.\n", device.ClientID)
	return
}

/* SUBSCRIPTIONS ****************************************************************************************/

/* SUBSCRIPTION -> START JOB  -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGStartJob() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGStartJob(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			start := StartJob{}
			if err := json.Unmarshal(msg.Payload(), &start); err != nil {
				pkg.LogErr(err)
			}

			go device.StartJobX(start)

		},
	}
}

/* SUBSCRIPTION -> START JOB  -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGEndJob() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGEndJob(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			sta := State{}
			if err := json.Unmarshal(msg.Payload(), &sta); err != nil {
				pkg.LogErr(err)
			}

			go device.EndJobX(sta)

		},
	}
}

/* SUBSCRIPTION -> PING  -> UPON RECEIPT, ALERT USER CLIENTS, UPDATE DevicePingsMap */
func (device *Device) MQTTSubscription_DeviceClient_SIGDevicePing() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGDevicePing(),
		Handler: func(c phao.Client, msg phao.Message) {

			device.DESMQTTClient.WG.Add(1)

			// /* TODO : PARSE THE PING MESSAGE */
			// if err := json.Unmarshal(msg.Payload(), &ping); err != nil {
			// 	pkg.LogErr(err)
			// }

			ping := pkg.Ping{
				Time: time.Now().UTC().UnixMilli(),
				OK:   true,
			}

			/* TODO : CHECK LATENCEY BETWEEN DEVICE PING TIME AND SERVER TIME
			- IGNORE THE RECEIVED DEVICE TIME FOR NOW,
			- WE DON'T REALLY CARE FOR KEEP-ALIVE PURPOSES
			*/

			/* CALL IN GO ROUTINE  *** DES TOPIC *** - ALERT USER CLIENTS */
			go device.MQTTPublication_DeviceClient_DESDevicePing(ping)

			/* UPDATE THE DevicesPingMap - DO NOT CALL IN GOROUTINE */
			device.UpdateDevicePing(ping)

			device.DESMQTTClient.WG.Done()
		},
	}
}

/* SUBSCRIPTION -> ADMIN  -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGAdmin() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGAdmin(),
		Handler: func(c phao.Client, msg phao.Message) {

			device.DESMQTTClient.WG.Add(1)

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			adm := Admin{}
			if err := json.Unmarshal(msg.Payload(), &adm); err != nil {
				pkg.LogErr(err)
			}

			/* CALL DB WRITE IN GOROUTINE */
			go WriteADM(adm, &device.CmdDBC)

			/* DECIDE WHAT TO DO BASED ON LAST STATE */
			if device.STA.StaLogging > OP_CODE_JOB_START_REQ {

				/* CALL DB WRITE IN GOROUTINE */
				go WriteADM(adm, &device.JobDBC)
			}

			device.ADM = adm

			/* UPDATE THE DevicesMap - DO NOT CALL IN GOROUTINE  */
			device.UpdateMappedADM()
			device.DESMQTTClient.WG.Done()

		},
	}
}

/* SUBSCRIPTION -> STATE  -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGState() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGState(),
		Handler: func(c phao.Client, msg phao.Message) {

			device.DESMQTTClient.WG.Add(1)

			/* PARSE / STORE THE STATE IN CMDARCHIVE */
			sta := State{}
			if err := json.Unmarshal(msg.Payload(), &sta); err != nil {
				pkg.LogErr(err)
			}

			/* CALL DB WRITE IN GOROUTINE */
			go WriteSTA(sta, &device.CmdDBC)

			if device.STA.StaLogging > OP_CODE_JOB_START_REQ {

				/* STORE THE STATE IN THE ACTIVE JOB;  CALL DB WRITE IN GOROUTINE */
				go WriteSTA(sta, &device.JobDBC)
			}

			device.STA = sta

			/* UPDATE THE DevicesMap - DO NOT CALL IN GOROUTINE  */
			device.UpdateMappedSTA()
			device.DESMQTTClient.WG.Done()
		},
	}
}

/* SUBSCRIPTION -> HEADER -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGHeader() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGHeader(),
		Handler: func(c phao.Client, msg phao.Message) {

			device.DESMQTTClient.WG.Add(1)

			/* PARSE / STORE THE HEADER IN CMDARCHIVE */
			hdr := Header{}
			if err := json.Unmarshal(msg.Payload(), &hdr); err != nil {
				pkg.LogErr(err)
			}

			/* CALL DB WRITE IN GOROUTINE */
			go WriteHDR(hdr, &device.CmdDBC)

			/* DECIDE WHAT TO DO BASED ON LAST STATE */
			if device.STA.StaLogging > OP_CODE_JOB_START_REQ {

				/* CALL DB WRITE IN GOROUTINE */
				go WriteHDR(hdr, &device.JobDBC)

				/* UPDATE THE JOB SEARCH TEXT */
				d := device
				d.HDR = hdr
				go d.Update_DESJobSearch(d.DESRegistration)
			}

			device.HDR = hdr

			/* UPDATE THE DevicesMap - DO NOT CALL IN GOROUTINE  */
			device.UpdateMappedHDR()
			device.DESMQTTClient.WG.Done()

		},
	}
}

/* SUBSCRIPTION -> CONFIG -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGConfig() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGConfig(),
		Handler: func(c phao.Client, msg phao.Message) {

			device.DESMQTTClient.WG.Add(1)

			/* PARSE / STORE THE CONFIG IN CMDARCHIVE */
			cfg := Config{}
			if err := json.Unmarshal(msg.Payload(), &cfg); err != nil {
				pkg.LogErr(err)
			}

			/* CALL DB WRITE IN GOROUTINE */
			go WriteCFG(cfg, &device.CmdDBC)

			/* DECIDE WHAT TO DO BASED ON LAST STATE */
			if device.STA.StaLogging > OP_CODE_JOB_START_REQ {

				/* CALL DB WRITE IN GOROUTINE */
				go WriteCFG(cfg, &device.JobDBC)
			}

			device.CFG = cfg

			/* UPDATE THE DevicesMap - DO NOT CALL IN GOROUTINE  */
			device.UpdateMappedCFG()
			device.DESMQTTClient.WG.Done()

		},
	}
}

/* SUBSCRIPTION -> EVENT -> UPON RECEIPT, WRITE TO JOB DATABASE  */
func (device *Device) MQTTSubscription_DeviceClient_SIGEvent() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGEvent(),
		Handler: func(c phao.Client, msg phao.Message) {

			device.DESMQTTClient.WG.Add(1)

			/* PARSE / STORE THE EVENT IN CMDARCHIVE */
			evt := Event{}

			if err := json.Unmarshal(msg.Payload(), &evt); err != nil {
				pkg.LogErr(err)
			}

			/* CALL DB WRITE IN GOROUTINE */
			go WriteEVT(evt, &device.CmdDBC)

			/* DECIDE WHAT TO DO BASED ON LAST STATE */
			if device.STA.StaLogging > OP_CODE_JOB_START_REQ {

				/* STORE THE EVENT IN THE ACTIVE JOB; CALL DB WRITE IN GOROUTINE */
				go WriteEVT(evt, &device.JobDBC)
			}

			device.EVT = evt

			/* UPDATE THE DevicesMap - DO NOT CALL IN GOROUTINE  */
			device.UpdateMappedEVT()
			device.DESMQTTClient.WG.Done()

		},
	}
}

/* SUBSCRIPTION -> SAMPLE -> UPON RECEIPT, WRITE TO JOB DATABASE  */
func (device *Device) MQTTSubscription_DeviceClient_SIGSample() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGSample(),
		Handler: func(c phao.Client, msg phao.Message) {

			device.DESMQTTClient.WG.Add(1)

			/* DECODE THE PAYLOAD INTO AN MQTT_Sample */
			mqtts := MQTT_Sample{}
			if err := json.Unmarshal(msg.Payload(), &mqtts); err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceClient_SIGSample(...) ->  mqtts :", mqtts)

			/* CREATE Sample STRUCT INTO WHICH WE'LL DECODE THE MQTT_Sample  */
			smp := &Sample{SmpJobName: mqtts.DesJobName}

			/* TODO: CHECK SAMPLE JOB NAME & MAKE DATABASE IF IT DOES NOT EXIST
			DEVICE HAS STARTED A JOB WITHOUT THE DES KNOWING ABOUT IT:
			- CALL START JOB
			- REQUEST LAST: ADM, STA, HDR, CFG, EVT
			*/

			/* DECODE BASE64URL STRING ( DATA ) */
			if err := smp.DecodeMQTTSample(mqtts.Data); err != nil {
				pkg.LogErr(err)
			}

			/* DECIDE WHAT TO DO BASED ON LAST STATE */
			if device.STA.StaLogging > OP_CODE_JOB_START_REQ {

				/* WRITE TO JOB DATABASE  */
				go WriteSMP(*smp, &device.JobDBC)

			} else {

				/* WRITE TO JOB CMDARCHIVE */
				go WriteSMP(*smp, &device.CmdDBC)
			}

			device.SMP = *smp

			/* UPDATE THE DevicesMap - DO NOT CALL IN GOROUTINE  */
			device.UpdateMappedSMP()

			device.DESMQTTClient.WG.Done()

		},
	}
}

/* SUBSCRIPTION -> DIAG SAMPLE -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGDiagSample() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGDiagSample(),
		Handler: func(c phao.Client, msg phao.Message) {

			device.DESMQTTClient.WG.Add(1)

			fmt.Println("(device *Device) MQTTSubscription_DeviceClient_SIGDiagSample(...) DOES NOT EXIST... DUMMY...")

			device.DESMQTTClient.WG.Done()
		},
	}
}

/* DES PUBLICATIONS **************************************************************************************/

/*
	DES PUBLICATION -> DEVICE CLIENT CONNECTED

SENT BY THE DES TO USER CLIENTS (WS) TO SIGNAL THIS DEVICE CLIENT'S BROKER CONNECTION STATUS
*/
func (device *Device) MQTTPublication_DeviceClient_DESDeviceClientPing(ping pkg.Ping) {
	des := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_DESDeviceClientPing(),
		Message:  pkg.ModelToJSONString(ping),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	} // pkg.Json("(dev *Device) MQTTPublication_DeviceClient_DESDeviceClientPing(): -> ping", ping)

	des.Pub(device.DESMQTTClient)
}

/*
	DES PUBLICATION -> DEVICE CONNECTED

SENT BY THE DES TO USER CLIENTS (WS) TO SIGNAL THE DEVICE'S BROKER CONNECTION STATUS
*/
func (device *Device) MQTTPublication_DeviceClient_DESDevicePing(ping pkg.Ping) {
	des := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_DESDevicePing(),
		Message:  pkg.ModelToJSONString(ping),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	} // pkg.Json("(dev *Device) MQTTPublication_DeviceClient_DESDevicePing(): -> ping", ping)

	des.Pub(device.DESMQTTClient)
}

/* CMD PUBLICATIONS **************************************************************************************/

/* PUBLICATION -> START JOB */
func (device *Device) MQTTPublication_DeviceClient_CMDStartJob() {

	start := StartJob{
		ADM: device.ADM,
		STA: device.STA,
		HDR: device.HDR,
		CFG: device.CFG,
		EVT: device.EVT,
	}

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDStartJob(),
		Message:  pkg.ModelToJSONString(start),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	} // pkg.Json("(dev *Device) MQTTPublication_DeviceClient_CMDAdmin(): -> cmd", cmd)

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> END JOB */
func (device *Device) MQTTPublication_DeviceClient_CMDEndJob(evt Event) {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDEndJob(),
		Message:  pkg.ModelToJSONString(evt),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	} // pkg.Json("(dev *Device) MQTTPublication_DeviceClient_CMDEndJob(): -> cmd", cmd)

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> REPORT */
func (device *Device) MQTTPublication_DeviceClient_CMDReport() {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDReport(),
		Message:  "eeeyaaaah...",
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	} // pkg.Json("(dev *Device) MQTTPublication_DeviceClient_CMDReport(): -> cmd", cmd)

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> ADMINISTRATION */
func (device *Device) MQTTPublication_DeviceClient_CMDAdmin(adm Admin) {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDAdmin(),
		Message:  pkg.ModelToJSONString(adm),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	} // pkg.Json("(dev *Device) MQTTPublication_DeviceClient_CMDAdmin(): -> cmd", cmd)

	cmd.Pub(device.DESMQTTClient)
}
/* PUBLICATION -> STATE */
func (device *Device) MQTTPublication_DeviceClient_CMDState(sta State) {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDState(),
		Message:  pkg.ModelToJSONString(sta),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	} // pkg.Json("(dev *Device) MQTTPublication_DeviceClient_CMDState(): -> sta", sta)

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> HEADER */
func (device *Device) MQTTPublication_DeviceClient_CMDHeader(hdr Header) {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDHeader(),
		Message:  pkg.ModelToJSONString(hdr),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> CONFIGURATION */
func (device *Device) MQTTPublication_DeviceClient_CMDConfig(cfg Config) {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDConfig(),
		Message:  pkg.ModelToJSONString(cfg),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> EVENT */
func (device *Device) MQTTPublication_DeviceClient_CMDEvent(evt Event) {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDEvent(),
		Message:  pkg.ModelToJSONString(evt),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> MESSAGE LIMIT TEST ***TODO: REMOVE AFTER DEVELOPMENT*** */
func (device *Device) MQTTPublication_DeviceClient_CMDMsgLimit(msg MsgLimit) {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDMsgLimit(),
		Message:  pkg.ModelToJSONString(msg),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	cmd.Pub(device.DESMQTTClient)
}

/* MQTT TOPICS ************************************************************************
THESE ARE USED BY ALL TYPES OF CLIENTS: Device, User, Demo */

func (device *Device) MQTTTopic_DeviceRoot() (root string) {
	return fmt.Sprintf("%s/%s/%s", device.DESDevClass, device.DESDevVersion, device.DESDevSerial)
}

func (device *Device) MQTTTopic_SIGRoot() (root string) {
	return fmt.Sprintf("%s/sig", device.MQTTTopic_DeviceRoot())
}

func (device *Device) MQTTTopic_CMDRoot() (root string) {
	return fmt.Sprintf("%s/cmd", device.MQTTTopic_DeviceRoot())
}

func (device *Device) MQTTTopic_DESRoot() (root string) {
	return fmt.Sprintf("%s/des", device.MQTTTopic_DeviceRoot())
}

/* MQTT TOPICS - SIGNAL */
func (device *Device) MQTTTopic_SIGStartJob() (topic string) {
	return fmt.Sprintf("%s/start", device.MQTTTopic_SIGRoot())
}
func (device *Device) MQTTTopic_SIGEndJob() (topic string) {
	return fmt.Sprintf("%s/end", device.MQTTTopic_SIGRoot())
}
func (device *Device) MQTTTopic_SIGDevicePing() (topic string) {
	return fmt.Sprintf("%s/ping", device.MQTTTopic_SIGRoot())
}
func (device *Device) MQTTTopic_SIGAdmin() (topic string) {
	return fmt.Sprintf("%s/admin", device.MQTTTopic_SIGRoot())
}
func (device *Device) MQTTTopic_SIGState() (topic string) {
	return fmt.Sprintf("%s/state", device.MQTTTopic_SIGRoot())
}
func (device *Device) MQTTTopic_SIGHeader() (topic string) {
	return fmt.Sprintf("%s/header", device.MQTTTopic_SIGRoot())
}
func (device *Device) MQTTTopic_SIGConfig() (topic string) {
	return fmt.Sprintf("%s/config", device.MQTTTopic_SIGRoot())
}
func (device *Device) MQTTTopic_SIGEvent() (topic string) {
	return fmt.Sprintf("%s/event", device.MQTTTopic_SIGRoot())
}
func (device *Device) MQTTTopic_SIGSample() (topic string) {
	return fmt.Sprintf("%s/sample", device.MQTTTopic_SIGRoot())
}
func (device *Device) MQTTTopic_SIGDiagSample() (topic string) {
	return fmt.Sprintf("%s/diag_sample", device.MQTTTopic_SIGRoot())
}
func (device *Device) MQTTTopic_SIGMsgLimit() (topc string) {
	/*** TODO: REMOVE AFTER DEVELOPMENT ***/
	return fmt.Sprintf("%s/msg_limit", device.MQTTTopic_SIGRoot())
}

/* MQTT TOPICS - COMMAND */
func (device *Device) MQTTTopic_CMDStartJob() (topic string) {
	return fmt.Sprintf("%s/start", device.MQTTTopic_CMDRoot())
}
func (device *Device) MQTTTopic_CMDEndJob() (topic string) {
	return fmt.Sprintf("%s/end", device.MQTTTopic_CMDRoot())
}
func (device *Device) MQTTTopic_CMDReport() (topic string) {
	return fmt.Sprintf("%s/report", device.MQTTTopic_CMDRoot())
}
func (device *Device) MQTTTopic_CMDAdmin() (topic string) {
	return fmt.Sprintf("%s/admin", device.MQTTTopic_CMDRoot())
}
func (device *Device) MQTTTopic_CMDState() (topic string) {
	return fmt.Sprintf("%s/state", device.MQTTTopic_CMDRoot())
}
func (device *Device) MQTTTopic_CMDHeader() (topic string) {
	return fmt.Sprintf("%s/header", device.MQTTTopic_CMDRoot())
}
func (device *Device) MQTTTopic_CMDConfig() (topic string) {
	return fmt.Sprintf("%s/config", device.MQTTTopic_CMDRoot())
}
func (device *Device) MQTTTopic_CMDEvent() (topic string) {
	return fmt.Sprintf("%s/event", device.MQTTTopic_CMDRoot())
}
func (device *Device) MQTTTopic_CMDSample() (topic string) {
	return fmt.Sprintf("%s/sample", device.MQTTTopic_CMDRoot())
}
func (device *Device) MQTTTopic_CMDDiagSample() (topic string) {
	return fmt.Sprintf("%s/diag_sample", device.MQTTTopic_CMDRoot())
}
func (device *Device) MQTTTopic_CMDMsgLimit() (topc string) {
	/*** TODO: REMOVE AFTER DEVELOPMENT ***/
	return fmt.Sprintf("%s/msg_limit", device.MQTTTopic_CMDRoot())
}

/* MQTT TOPICS - DES MESSAGE */
func (device *Device) MQTTTopic_DESDeviceClientPing() (topic string) {
	return fmt.Sprintf("%s/des_ping", device.MQTTTopic_DESRoot())
}

func (device *Device) MQTTTopic_DESDevicePing() (topic string) {
	return fmt.Sprintf("%s/ping", device.MQTTTopic_DESRoot())
}

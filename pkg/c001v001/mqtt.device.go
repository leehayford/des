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

	if device.DESMQTTClient.Client != nil {
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
	}
	/* DISCONNECT THE DESMQTTCLient */
	device.DESMQTTClient_Disconnect()

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
			/* VALIDATE */
			if err := start.SIGValidate(device); err != nil { 
				go pkg.LogDESError(device.DESDevSerial, err.Error(), start)
			} else {
				if err := device.StartJob(start); err != nil {
					go pkg.LogDESError(device.DESDevSerial, err.Error(), start)
				}
			} 
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

			/* VALIDATE */
			if err := sta.SIGValidate(device); err != nil { 
				go pkg.LogDESError(device.DESDevSerial, err.Error(), sta)
			} else { 
				device.EndJob(sta)
			}

		},
	}
}

/* SUBSCRIPTION -> PING  -> UPON RECEIPT, ALERT USER CLIENTS, UPDATE DevicePingsMap */
func (device *Device) MQTTSubscription_DeviceClient_SIGDevicePing() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGDevicePing(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* TODO : PARSE THE PING MESSAGE
			TODO : CHECK LATENCEY BETWEEN DEVICE PING TIME AND SERVER TIME
			- IGNORE THE RECEIVED DEVICE TIME FOR NOW,
			- WE DON'T REALLY CARE FOR KEEP-ALIVE PURPOSES

			// if err := json.Unmarshal(msg.Payload(), &ping); err != nil {
			// 	pkg.LogErr(err)
			// }
			*/
			ping := pkg.Ping{
				Time: time.Now().UTC().UnixMilli(),
				OK:   true,
			}

			/* UPDATE THE DevicesPingMap - DO NOT CALL IN GOROUTINE */
			device.UpdateDevicePing(ping)
		},
	}
}

/* SUBSCRIPTION -> ADMIN  -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGAdmin() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGAdmin(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE ADMIN IN CMDARCHIVE */
			adm := Admin{}
			if err := json.Unmarshal(msg.Payload(), &adm); err != nil {
				pkg.LogErr(err)
			}

			/* VALIDATE */
			if err := adm.SIGValidate(device); err != nil { 
				go pkg.LogDESError(device.DESDevSerial, err.Error(), adm)
			} else {
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
			}
		},
	}
}

/* SUBSCRIPTION -> STATE  -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGState() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGState(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE STATE IN CMDARCHIVE */
			sta := State{}
			if err := json.Unmarshal(msg.Payload(), &sta); err != nil {
				pkg.LogErr(err)
			}

			/* VALIDATE */
			if err := sta.SIGValidate(device); err != nil { 
				go pkg.LogDESError(device.DESDevSerial, err.Error(), sta)
			} else {
				/* CALL DB WRITE IN GOROUTINE */
				go WriteSTA(sta, &device.CmdDBC)

				if device.STA.StaLogging > OP_CODE_JOB_START_REQ {

					/* STORE THE STATE IN THE ACTIVE JOB;  CALL DB WRITE IN GOROUTINE */
					go WriteSTA(sta, &device.JobDBC)
				}

				device.STA = sta

				/* UPDATE THE DevicesMap - DO NOT CALL IN GOROUTINE  */
				device.UpdateMappedSTA()
			}
		},
	}
}

/* SUBSCRIPTION -> HEADER -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGHeader() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGHeader(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE HEADER IN CMDARCHIVE */
			hdr := Header{}
			if err := json.Unmarshal(msg.Payload(), &hdr); err != nil {
				pkg.LogErr(err)
			}

			/* VALIDATE */
			if err := hdr.SIGValidate(device); err != nil { 
				go pkg.LogDESError(device.DESDevSerial, err.Error(), hdr)
			} else {
				/* CALL DB WRITE IN GOROUTINE */
				go WriteHDR(hdr, &device.CmdDBC)

				/* DECIDE WHAT TO DO BASED ON LAST STATE */
				if device.STA.StaLogging > OP_CODE_JOB_START_REQ {

					/* CALL DB WRITE IN GOROUTINE */
					go WriteHDR(hdr, &device.JobDBC)

					/* UPDATE THE JOB SEARCH TEXT */
					d := device
					d.HDR = hdr
					d.Update_DESJobSearch(d.DESRegistration)
				}

				device.HDR = hdr

				/* UPDATE THE DevicesMap - DO NOT CALL IN GOROUTINE  */
				device.UpdateMappedHDR()
			}
		},
	}
}

/* SUBSCRIPTION -> CONFIG -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGConfig() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGConfig(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE CONFIG IN CMDARCHIVE */
			cfg := Config{}
			if err := json.Unmarshal(msg.Payload(), &cfg); err != nil {
				pkg.LogErr(err)
			}

			/* VALIDATE */
			if err := cfg.SIGValidate(device); err != nil { 
				go pkg.LogDESError(device.DESDevSerial, err.Error(), cfg)
			} else {
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
			}
		},
	}
}

/* SUBSCRIPTION -> EVENT -> UPON RECEIPT, WRITE TO JOB DATABASE  */
func (device *Device) MQTTSubscription_DeviceClient_SIGEvent() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGEvent(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE EVENT IN CMDARCHIVE */
			evt := Event{}

			if err := json.Unmarshal(msg.Payload(), &evt); err != nil {
				pkg.LogErr(err)
			}

			/* VALIDATE */
			if err := evt.SIGValidate(device); err != nil { 
				go pkg.LogDESError(device.DESDevSerial, err.Error(), evt)
			} else {
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
			}
		},
	}
}

/* SUBSCRIPTION -> SAMPLE -> UPON RECEIPT, WRITE TO JOB DATABASE  */
func (device *Device) MQTTSubscription_DeviceClient_SIGSample() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGSample(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* DECODE THE PAYLOAD INTO AN MQTT_Sample */
			mqtts := MQTT_Sample{}
			if err := json.Unmarshal(msg.Payload(), &mqtts); err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceClient_SIGSample(...) ->  mqtts :", mqtts)

			device.HandleMQTTSample(mqtts)
		},
	}
}

/* SUBSCRIPTION -> DIAG SAMPLE -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGDiagSample() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGDiagSample(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* TODO: VALIDATE */
			valid := true
			if valid { 
				fmt.Println("(device *Device) MQTTSubscription_DeviceClient_SIGDiagSample(...) DOES NOT EXIST... DUMMY...")
			}
		},
	}
}


/* DES PUBLICATIONS **************************************************************************************/

/*
	DES PUBLICATION -> DEVICE CLIENT CONNECTED

SENT BY THE DES TO USER CLIENTS (WS) TO SIGNAL THIS DEVICE CLIENT'S BROKER CONNECTION STATUS
*/
func (device *Device) MQTTPublication_DeviceClient_DESDeviceClientPing(ping pkg.Ping) {
	
	json, err := pkg.ModelToJSONString(ping)
	if err != nil {
		pkg.LogErr(err)
	}

	des := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_DESDeviceClientPing(),
		Message:  json,
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

	json, err := pkg.ModelToJSONString(ping)
	if err != nil {
		pkg.LogErr(err)
	}

	des := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_DESDevicePing(),
		Message:  json,
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

	json, err := pkg.ModelToJSONString(start)
	if err != nil {
		pkg.LogErr(err)
	}

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDStartJob(),
		Message:  json,
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	} // pkg.Json("(dev *Device) MQTTPublication_DeviceClient_CMDAdmin(): -> cmd", cmd)

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> END JOB */
func (device *Device) MQTTPublication_DeviceClient_CMDEndJob(evt Event) {

	json, err := pkg.ModelToJSONString(evt)
	if err != nil {
		pkg.LogErr(err)
	}

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDEndJob(),
		Message:  json,
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
		Message:  "eeeyaaaah...", // I'm gonna need those TPS reports...
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	} // pkg.Json("(dev *Device) MQTTPublication_DeviceClient_CMDReport(): -> cmd", cmd)

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> ADMINISTRATION */
func (device *Device) MQTTPublication_DeviceClient_CMDAdmin(adm Admin) {

	json, err := pkg.ModelToJSONString(adm)
	if err != nil {
		pkg.LogErr(err)
	}

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDAdmin(),
		Message:  json,
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	} // pkg.Json("(dev *Device) MQTTPublication_DeviceClient_CMDAdmin(): -> cmd", cmd)

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> STATE */
func (device *Device) MQTTPublication_DeviceClient_CMDState(sta State) {

	json, err := pkg.ModelToJSONString(sta)
	if err != nil {
		pkg.LogErr(err)
	}

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDState(),
		Message:  json,
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	} // pkg.Json("(dev *Device) MQTTPublication_DeviceClient_CMDState(): -> sta", sta)

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> HEADER */
func (device *Device) MQTTPublication_DeviceClient_CMDHeader(hdr Header) {

	json, err := pkg.ModelToJSONString(hdr)
	if err != nil {
		pkg.LogErr(err)
	}

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDHeader(),
		Message:  json,
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> CONFIGURATION */
func (device *Device) MQTTPublication_DeviceClient_CMDConfig(cfg Config) {

	json, err := pkg.ModelToJSONString(cfg)
	if err != nil {
		pkg.LogErr(err)
	}

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDConfig(),
		Message:  json,
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> EVENT */
func (device *Device) MQTTPublication_DeviceClient_CMDEvent(evt Event) {

	json, err := pkg.ModelToJSONString(evt)
	if err != nil {
		pkg.LogErr(err)
	}

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDEvent(),
		Message:  json,
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> MESSAGE LIMIT TEST ***TODO: REMOVE AFTER DEVELOPMENT*** */
func (device *Device) MQTTPublication_DeviceClient_CMDMsgLimit(msg MsgLimit) {

	json, err := pkg.ModelToJSONString(msg)
	if err != nil {
		pkg.LogErr(err)
	}

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDMsgLimit(),
		Message:  json,
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> MESSAGE LIMIT TEST ***TODO: REMOVE AFTER DEVELOPMENT*** */
func (device *Device) MQTTPublication_DeviceClient_CMDTestOLS() {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDTestOLS(),
		Message:  "eeeyaaaah...",
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	cmd.Pub(device.DESMQTTClient)
}

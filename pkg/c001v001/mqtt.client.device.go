package c001v001

import (
	"fmt"

	phao "github.com/eclipse/paho.mqtt.golang"

	"github.com/leehayford/des/pkg"
)

/*
	MQTT DEVICE CLIENT

PUBLISHES ALL COMMANDS TO A SINGLE DEVICE
SUBSCRIBES TO ALL SIGNALS FOR A SINGLE DEVICE
  - WRITES MESSAGES TO THE JOB DATABASE
*/
func (device *Device) MQTTDeviceClient_Connect() (err error) {

	/* TODO: replace with <device.Class>-<device.Version> specific credentials */
	class_version_user := pkg.MQTT_USER
	class_version_pw := pkg.MQTT_PW

	device.MQTTUser = class_version_user
	device.MQTTPW = class_version_pw
	device.MQTTClientID = fmt.Sprintf(
		"%s-%s-%s-DES",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)

	if err = device.DESMQTTClient.DESMQTTClient_Connect(); err != nil {
		return err
	}
	// pkg.Json(`(device *Device) RegisterMQTTDESDeviceClient(...) -> device.DESMQTTClient.RegisterDESMQTTClient()
	// device.DESMQTTClient.ClientOptions.ClientID:`,
	// device.DESMQTTClient.ClientOptions.ClientID)

	device.MQTTSubscription_DeviceClient_SIGAdmin().Sub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGHeader().Sub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGConfig().Sub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGEvent().Sub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGSample().Sub(device.DESMQTTClient)
	// device.MQTTSubscription_DeviceClient_SIGDiagSample() //.Sub(device.DESMQTTClient)

	// pkg.MQTTDevClients[device.DESDevSerial] = device.DESMQTTClient
	// deviceClient := pkg.MQTTDevClients[device.DESDevSerial]
	// fmt.Printf("\n%s client ID: %s\n", device.DESDevSerial, deviceClient.MQTTClientID)

	Devices[device.DESDevSerial] = *device
	d := Devices[device.DESDevSerial]
	fmt.Printf("\n(device) MQTTDeviceClient_Connect( ) -> ClientID: %s\n", d.ClientID)
	return err
}
func (device *Device) MQTTDeviceClient_Dicconnect() {

	/* UNSUBSCRIBE FROM ALL MQTTSubscriptions */
	device.MQTTSubscription_DeviceClient_SIGAdmin().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGHeader().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGConfig().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGEvent().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGSample().UnSub(device.DESMQTTClient)
	// device.MQTTSubscription_DeviceClient_SIGDiagSample() //.UnSub(device.DESMQTTClient)

	/* DISCONNECT THE DESMQTTCLient */
	device.DESMQTTClient_Disconnect()

	// delete(pkg.MQTTDevClients, device.DESDevSerial)

	fmt.Printf("\n(device) MQTTDeviceClient_Dicconnect( ): Complete -> ClientID: %s\n", device.ClientID)
}

/* CREATE A DEVICE CLIENT FOR EACH REGISTERED DEVICE */
func MQTTDeviceClient_CreateAndConnectAll() (err error) {

	drs, err := GetDeviceList()
	if err != nil {
		return pkg.TraceErr(err)
	} // pkg.Json("GetDeviceList():", drs)

	for _, dr := range drs {
		device := Device{
			DESRegistration: dr,
			Job:             Job{DESRegistration: dr},
			DESMQTTClient:   pkg.DESMQTTClient{},
		}
		if err = device.MQTTDeviceClient_Connect(); err != nil {
			return pkg.TraceErr(err)
		}
	}

	return err
}

/*
SUBSCRIPTIONS
*/
/* SUBSCRIPTION -> ADMINISTRATION  -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGAdmin() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGAdmin(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE ADMIN IN ZERO JOB */
			zero := device.ZeroJob()
			device.ADM.AdmID = 0
			zero.WriteMQTT(msg.Payload(), &device.ADM)

			/* DECIDE WHAT TO DO BASED ON LAST EVENT */
			if device.EVT.EvtCode > 1 {
				device.ADM.AdmID = 0
				device.Job.WriteMQTT(msg.Payload(), &device.ADM)
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

			/* PARSE / STORE THE HEADER IN ZERO JOB */
			zero := device.ZeroJob()
			device.HDR.HdrID = 0
			zero.WriteMQTT(msg.Payload(), &device.HDR)
			
			/* DECIDE WHAT TO DO BASED ON LAST EVENT */
			if device.EVT.EvtCode > 1 {
				device.HDR.HdrID = 0
				device.Job.WriteMQTT(msg.Payload(), &device.HDR)
			}
		},
	}
}

/* SUBSCRIPTION -> CONFIGURATION -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGConfig() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGConfig(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE CONFIG IN ZERO JOB */
			zero := device.ZeroJob()
			device.CFG.CfgID = 0
			zero.WriteMQTT(msg.Payload(), &device.CFG)

			/* DECIDE WHAT TO DO BASED ON LAST EVENT */
			if device.EVT.EvtCode > 1 {
				device.CFG.CfgID = 0
				device.Job.WriteMQTT(msg.Payload(), &device.CFG)
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

			/* PARSE / STORE THE EVENT IN ZERO JOB */
			zero := device.ZeroJob()
			device.EVT.EvtID = 0
			zero.WriteMQTT(msg.Payload(), &device.EVT)
			pkg.Json("MQTTSubscription_DeviceClient_SIGEvent(...) -> evice.EVT :", device.EVT)

			/* DECIDE WHAT TO DO BASED ON LAST EVENT */
			switch device.EVT.EvtCode {

			case 1: // End Job
				device.EndJob()

			case 2: // Start Job
				device.StartJob()

			case 10: // Mode Vent
			case 11: // Mode Build
			case 12: // Mode Hi Flow
			case 13: // Mode Lo Flow
			default:
				// WRITE TO CURRETN JOB
				device.Job.WriteMQTT(msg.Payload(), &device.EVT)

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
			device.Job.WriteMQTTSample(msg.Payload(), &device.SMP)
		},
	}
}

/* SUBSCRIPTION -> DIAG SAMPLE -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGDiagSample() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGDiagSample(),
		Handler: func(c phao.Client, msg phao.Message) {
			fmt.Println("(device *Device) MQTTSubscription_DeviceClient_SIGDiagSample(...) DOES NOT EXIST... DUMMY...")
		},
	}
}

/*
PUBLICATIONS
*/
/* PUBLICATION -> ADMINISTRATION */
func (device *Device) MQTTPublication_DeviceClient_CMDAdmin(adm Admin) {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDAdmin(),
		Message:  pkg.MakeMQTTMessage(adm.FilterAdmRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	} // pkg.Json("(dev *Device) MQTTPublication_DeviceClient_CMDAdmin(): -> cmd", cmd)
	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> HEADER */
func (device *Device) MQTTPublication_DeviceClient_CMDHeader(hdr Header) {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDHeader(),
		Message:  pkg.MakeMQTTMessage(hdr.FilterHdrRecord()),
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
		Message:  pkg.MakeMQTTMessage(cfg.FilterCfgRecord()),
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
		Message:  pkg.MakeMQTTMessage(evt.FilterEvtRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}
	cmd.Pub(device.DESMQTTClient)
}

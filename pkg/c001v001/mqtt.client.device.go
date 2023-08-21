
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
		"DESDevice-%s-%s-%s",
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

	device.MQTTSubscription_DeviceClient_SIGDiagSample() //.Sub(device.DESMQTTClient)

	return err
}
func (device *Device) MQTTDeviceClient_Dicconnect() {

	/* UNSUBSCRIBE FROM ALL MQTTSubscriptions */
	device.MQTTSubscription_DeviceClient_SIGAdmin().UnSub(device.DESMQTTClient)

	device.MQTTSubscription_DeviceClient_SIGHeader().UnSub(device.DESMQTTClient)

	device.MQTTSubscription_DeviceClient_SIGConfig().UnSub(device.DESMQTTClient)

	device.MQTTSubscription_DeviceClient_SIGEvent().UnSub(device.DESMQTTClient)

	device.MQTTSubscription_DeviceClient_SIGSample().UnSub(device.DESMQTTClient)

	device.MQTTSubscription_DeviceClient_SIGDiagSample() //.UnSub(device.DESMQTTClient)

	/* DISCONNECT THE DESMQTTCLient */
	device.DESMQTTClient_Disconnect()

	fmt.Printf(" (device *Device) MQTTDeviceClient_Dicconnect( ... ): Complete; OKCancel.\n")
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
			device.Job.WriteMQTT(msg.Payload(), Admin{})
		},
	}
}

/* SUBSCRIPTION -> HEADER -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGHeader() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGHeader(),
		Handler: func(c phao.Client, msg phao.Message) {
			device.Job.WriteMQTT(msg.Payload(), Header{})
		},
	}
}


/* SUBSCRIPTION -> CONFIGURATION -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGConfig() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGConfig(),
		Handler: func(c phao.Client, msg phao.Message) {
			device.Job.WriteMQTT(msg.Payload(), Config{})
		},
	}
}

/* SUBSCRIPTION -> EVENT -> UPON RECEIPT, WRITE TO JOB DATABASE  */
func (device *Device) MQTTSubscription_DeviceClient_SIGEvent() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGEvent(),
		Handler: func(c phao.Client, msg phao.Message) {
			device.Job.WriteMQTT(msg.Payload(), Event{})
		},
	}
}

/* SUBSCRIPTION -> SAMPLE -> UPON RECEIPT, WRITE TO JOB DATABASE  */
func (device *Device) MQTTSubscription_DeviceClient_SIGSample() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGSample(),
		Handler: func(c phao.Client, msg phao.Message) {
			device.Job.WriteMQTTSample(msg.Payload())
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
func (device *Device) MQTTPublication_DeviceClient_CMDAdmin(adm Admin) bool {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDAdmin(),
		Message:  pkg.MakeMQTTMessage(adm.FilterAdmRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	return cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> HEADER */
func (device *Device) MQTTPublication_DeviceClient_CMDHeader(hdr Header) bool {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDHeader(),
		Message:  pkg.MakeMQTTMessage(hdr.FilterHdrRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	return cmd.Pub(device.DESMQTTClient)
}


/* PUBLICATION -> CONFIGURATION */
func (device *Device) MQTTPublication_DeviceClient_CMDConfig(cfg Config) bool {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDConfig(),
		Message:  pkg.MakeMQTTMessage(cfg.FilterCfgRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	return cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> EVENT */
func (device *Device) MQTTPublication_DeviceClient_CMDEvent(evt Event) bool {

	cmd := pkg.MQTTPublication{
		Topic:    device.MQTTTopic_CMDEvent(),
		Message:  pkg.MakeMQTTMessage(evt.FilterEvtRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	return cmd.Pub(device.DESMQTTClient)
}

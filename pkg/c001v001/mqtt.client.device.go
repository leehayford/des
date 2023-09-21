package c001v001

import (
	"encoding/json"
	"fmt"

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

	if err = device.DESMQTTClient.DESMQTTClient_Connect(false); err != nil {
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

	return err
}
func (device *Device) MQTTDeviceClient_Disconnect() (err error) {

	/* UNSUBSCRIBE FROM ALL MQTTSubscriptions */
	device.MQTTSubscription_DeviceClient_SIGAdmin().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGHeader().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGConfig().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGEvent().UnSub(device.DESMQTTClient)
	device.MQTTSubscription_DeviceClient_SIGSample().UnSub(device.DESMQTTClient)
	// device.MQTTSubscription_DeviceClient_SIGDiagSample() //.UnSub(device.DESMQTTClient)

	/* DISCONNECT THE DESMQTTCLient */
	err = device.DESMQTTClient_Disconnect()

	fmt.Printf("\n(device) MQTTDeviceClient_Dicconnect( ): Complete -> ClientID: %s\n", device.ClientID)

	return err
}

/* SUBSCRIPTIONS ****************************************************************************************/

/* SUBSCRIPTION -> ADMINISTRATION  -> UPON RECEIPT, WRITE TO JOB DATABASE */
func (device *Device) MQTTSubscription_DeviceClient_SIGAdmin() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: device.MQTTTopic_SIGAdmin(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE / STORE THE ADMIN IN ZERO JOB */
			if err := json.Unmarshal(msg.Payload(), &device.ADM); err != nil {
				pkg.TraceErr(err)
			}
			go device.CmdDBC.Write(device.ADM)
			// device.ZeroDBC.WriteMQTT(msg.Payload(), &device.ADM)

			/* DECIDE WHAT TO DO BASED ON LAST EVENT */
			if device.EVT.EvtCode > STATUS_JOB_START_REQ {
				go device.JobDBC.Write(device.ADM)
				// device.JobDBC.WriteMQTT(msg.Payload(), &device.ADM)
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
			if err := json.Unmarshal(msg.Payload(), &device.HDR); err != nil {
				pkg.TraceErr(err)
			}
			go device.CmdDBC.Write(device.HDR)
			// device.ZeroDBC.WriteMQTT(msg.Payload(), &device.HDR)

			/* DECIDE WHAT TO DO BASED ON LAST EVENT */
			if device.EVT.EvtCode > STATUS_JOB_START_REQ {
				go device.JobDBC.Write(device.HDR)
				// device.JobDBC.WriteMQTT(msg.Payload(), &device.HDR)
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
			if err := json.Unmarshal(msg.Payload(), &device.CFG); err != nil {
				pkg.TraceErr(err)
			}
			go device.CmdDBC.Write(device.CFG)
			// device.ZeroDBC.WriteMQTT(msg.Payload(), &device.CFG)

			/* DECIDE WHAT TO DO BASED ON LAST EVENT */
			if device.EVT.EvtCode > STATUS_JOB_START_REQ {
				/* PARSE / STORE THE EVENT IN THE ACTIVE JOB */
				go device.JobDBC.Write(device.CFG)
				// device.JobDBC.WriteMQTT(msg.Payload(), &device.CFG)
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

			/* CAPTURE THE ORIGINAL DEVICE STATE EVENT CODE */
			// state := device.EVT.EvtCode

			evt := Event{}
			/* PARSE / STORE THE EVENT IN ZERO JOB */
			if err := json.Unmarshal(msg.Payload(), &evt); err != nil {
				pkg.TraceErr(err)
			}
			go device.CmdDBC.Write(evt)

			/* CHECK THE RECEIVED EVENT CODE */
			switch evt.EvtCode {

			// case 0:
			/* REGISTRATION EVENT: USED TO ASSIGN THIS DEVICE TO
			A DIFFERENT DATA EXCHANGE SERVER */

			case STATUS_JOB_ENDED:
				go device.EndJob(evt)

			case STATUS_JOB_STARTED:
				go device.StartJob(evt)

			default:

				/* CHECK THE ORIGINAL DEVICE STATE EVENT CODE
				TO SEE IF WE SHOULD WRITE TO THE ACTIVE JOB */
				if device.EVT.EvtCode > STATUS_JOB_START_REQ {
					/* STORE THE EVENT IN THE ACTIVE JOB */
					go device.JobDBC.Write(evt)
					// device.JobDBC.WriteMQTT(msg.Payload(), &device.EVT)
				}
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
			/* TODO: MOVE WRITE SAMPLE FUCTION TO Device */
			// go device.Job.WriteMQTTSample(msg.Payload(), &device.SMP)
			// fmt.Printf("\n(device *Device) MQTTSubscription_DeviceClient_SIGSample() -> device.JobDBC: \n%s \n", device.JobDBC.ConnStr)
			if device.EVT.EvtCode > STATUS_JOB_START_REQ {
				// Decode the payload into an MQTTSampleMessage
				mqtts := &MQTT_Sample{}
				if err := json.Unmarshal(msg.Payload(), &mqtts); err != nil {
					pkg.TraceErr(err)
				} // pkg.Json("DecodeMQTTSampleMessage(...) ->  msg :", msg)

				for _, b64 := range mqtts.Data {

					// Decode base64 string
					device.SMP.SmpJobName = mqtts.DesJobName
					if err := device.Job.DecodeMQTTSample(b64, &device.SMP); err != nil {
						pkg.TraceErr(err)
					}

					// Write the Sample to the job database
					go device.JobDBC.Write(device.SMP)

				}
				// device.UpdateMappedSMP()
			}
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

/* PUBLICATIONS ******************************************************************************************/

/* PUBLICATION -> ADMINISTRATION */
func (device *Device) MQTTPublication_DeviceClient_CMDAdmin(adm Admin) {

	cmd := pkg.MQTTPublication{
		Topic:   device.MQTTTopic_CMDAdmin(),
		Message: pkg.MakeMQTTMessage(adm),
		// Message:  pkg.MakeMQTTMessage(adm.FilterAdmRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	} // pkg.Json("(dev *Device) MQTTPublication_DeviceClient_CMDAdmin(): -> cmd", cmd)

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> HEADER */
func (device *Device) MQTTPublication_DeviceClient_CMDHeader(hdr Header) {

	cmd := pkg.MQTTPublication{
		Topic:   device.MQTTTopic_CMDHeader(),
		Message: pkg.MakeMQTTMessage(hdr),
		// Message:  pkg.MakeMQTTMessage(hdr.FilterHdrRecord()),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	cmd.Pub(device.DESMQTTClient)
}

/* PUBLICATION -> CONFIGURATION */
func (device *Device) MQTTPublication_DeviceClient_CMDConfig(cfg Config) {

	cmd := pkg.MQTTPublication{
		Topic:   device.MQTTTopic_CMDConfig(),
		Message: pkg.MakeMQTTMessage(cfg),
		// Message:  pkg.MakeMQTTMessage(cfg.FilterCfgRecord()),
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
		Message:  pkg.MakeMQTTMessage(evt),
		Retained: false,
		WaitMS:   0,
		Qos:      0,
	}

	cmd.Pub(device.DESMQTTClient)
}

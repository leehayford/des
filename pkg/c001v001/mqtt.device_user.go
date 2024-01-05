package c001v001

import (
	"encoding/json"
	// "fmt"

	phao "github.com/eclipse/paho.mqtt.golang"

	"github.com/leehayford/des/pkg"
)

/*
	 MQTT DEVICE USER CLIENT
		SUBSCRIBES TO ALL SIGNALS FOR A SINGLE DEVICE
		PUBLICATIONS: NONE
*/
func (duc *DeviceUserClient) MQTTDeviceUserClient_Connect() (err error) {

	/* TODO: replace with user specific credentials */
	user := pkg.MQTT_USER
	pw := pkg.MQTT_PW

	duc.MQTTUser = user
	duc.MQTTPW = pw

	/* DEVICE USER CLIENTS ***DO NOT*** AUTOMATICALLY RESUBSCRIBE */
	if err = duc.DESMQTTClient.DESMQTTClient_Connect(true, false); err != nil {
		return err
	}
	// pkg.Json(`(*DeviceUserClient) MQTTDeviceUserClient_Connect(...) -> duc.DESMQTTClient.DESMQTTClient_Connect()
	// duc.DESMQTTClient.ClientOptions.ClientID:`,
	// duc.DESMQTTClient.ClientOptions.ClientID)

	duc.MQTTSubscription_DeviceUserClient_StartJob(duc.MQTTTopic_SIGStartJob()).Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_StartJob(duc.MQTTTopic_CMDStartJob()).Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGEndJob().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_CMDEndJob().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_DESDeviceClientPing().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_DESDevicePing().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGAdmin().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGState().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGHeader().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGConfig().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGEvent().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGSample().Sub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGDiagSample().Sub(duc.DESMQTTClient)

	/* MESSAGE LIMIT TEST ***TODO: REMOVE AFTER DEVELOPMENT*** */
	duc.MQTTSubscription_DeviceUserClient_SIGMsgLimit().Sub(duc.DESMQTTClient)

	// fmt.Printf("\n(*DeviceUserClient) MQTTDeviceUserClient_Connect( ): %s\n", duc.MQTTClientID)
	return err
}
func (duc *DeviceUserClient) MQTTDeviceUserClient_Disconnect() {

	/* UNSUBSCRIBE FROM ALL MQTTSubscriptions */
	duc.MQTTSubscription_DeviceUserClient_StartJob(duc.MQTTTopic_SIGStartJob()).UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_StartJob(duc.MQTTTopic_CMDStartJob()).UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGEndJob().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_CMDEndJob().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_DESDeviceClientPing().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_DESDevicePing().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGAdmin().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGState().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGHeader().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGConfig().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGEvent().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGSample().UnSub(duc.DESMQTTClient)
	duc.MQTTSubscription_DeviceUserClient_SIGDiagSample().UnSub(duc.DESMQTTClient)

	/* MESSAGE LIMIT TEST ***TODO: REMOVE AFTER DEVELOPMENT*** */
	duc.MQTTSubscription_DeviceUserClient_SIGMsgLimit().UnSub(duc.DESMQTTClient)

	/* DISCONNECT THE DESMQTTCLient */
	duc.DESMQTTClient_Disconnect()

	// fmt.Printf("\n(*DeviceUserClient) MQTTDeviceUserClient_Disconnect( ): %s\n", duc.MQTTClientID)
}

/* SUBSCRIPTIONS ****************************************************************************************/

/* SUBSCRIPTIONS -> START JOB -> SIGNAL FROM DEVICE & CMD FROM OTHER USERS */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_StartJob(topic string) pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: topic,
		Handler: func(c phao.Client, msg phao.Message) {

			/* DECODE MESSAGE PAYLOAD TO Admin STRUCT */
			start := StartJob{}
			if err := json.Unmarshal(msg.Payload(), &start); err != nil {
				pkg.LogErr(err)
			}

			/* CREATE JSON WSMessage STRUCT */
			js, err := json.Marshal(&pkg.WSMessage{Type: "start", Data: start})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGStartJob(...) -> start :", start)

			/* SEND WSMessage AS JSON STRING */
			duc.WriteDataOut(string(js))

		},
	}
}
/* SUBSCRIPTIONS -> START END -> SIGNAL FROM DEVICE */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGEndJob() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGEndJob(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* DECODE MESSAGE PAYLOAD TO Admin STRUCT */
			sta := State{}
			if err := json.Unmarshal(msg.Payload(), &sta); err != nil {
				pkg.LogErr(err)
			}

			/* CREATE JSON WSMessage STRUCT */
			js, err := json.Marshal(&pkg.WSMessage{Type: "end_sig", Data: sta})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGEndJob(...) -> sta :", sta)

			/* SEND WSMessage AS JSON STRING */
			duc.WriteDataOut(string(js))

		},
	}
}
/* SUBSCRIPTIONS -> START END -> CMD FROM OTHER USERS */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_CMDEndJob() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_CMDEndJob(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* DECODE MESSAGE PAYLOAD TO Admin STRUCT */
			evt := Event{}
			if err := json.Unmarshal(msg.Payload(), &evt); err != nil {
				pkg.LogErr(err)
			}

			/* CREATE JSON WSMessage STRUCT */
			js, err := json.Marshal(&pkg.WSMessage{Type: "end_cmd", Data: evt})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_CMDEndJob(...) -> evt :", evt)

			/* SEND WSMessage AS JSON STRING */
			duc.WriteDataOut(string(js))

		},
	}
}


/* SUBSCRIPTIONS -> DES DEVICE PING  */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_DESDeviceClientPing( /* TODO: PASS IN USER ROLE */ ) pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_DESDeviceClientPing(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* DECODE MESSAGE PAYLOAD TO Ping STRUCT */
			ping := pkg.Ping{}
			if err := json.Unmarshal(msg.Payload(), &ping); err != nil {
				pkg.LogErr(err)
			}

			/* CREATE JSON WSMessage STRUCT */
			js, err := json.Marshal(&pkg.WSMessage{Type: "des_ping", Data: ping})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_DESDeviceClientPing(...) -> ping :", ping)

			/* SEND WSMessage AS JSON STRING */
			duc.WriteDataOut(string(js))

		},
	}
}

/* SUBSCRIPTIONS -> DES DEVICE PING  */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_DESDevicePing( /* TODO: PASS IN USER ROLE */ ) pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_DESDevicePing(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* DECODE MESSAGE PAYLOAD TO Ping STRUCT */
			ping := pkg.Ping{}
			if err := json.Unmarshal(msg.Payload(), &ping); err != nil {
				pkg.LogErr(err)
			}

			/* CREATE JSON WSMessage STRUCT */
			js, err := json.Marshal(&pkg.WSMessage{Type: "ping", Data: ping})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_DESDevicePing(...) -> ping :", js)

			/* SEND WSMessage AS JSON STRING */
			duc.WriteDataOut(string(js))

		},
	}
}

/* SUBSCRIPTIONS -> ADMIN  */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGAdmin( /* TODO: PASS IN USER ROLE */ ) pkg.MQTTSubscription {
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
			js, err := json.Marshal(&pkg.WSMessage{Type: "admin", Data: adm})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGAdmin(...) -> adm :", adm)

			/* SEND WSMessage AS JSON STRING */
			duc.WriteDataOut(string(js))

		},
	}
}

/* SUBSCRIPTIONS -> STATE  */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGState( /* TODO: PASS IN USER ROLE */ ) pkg.MQTTSubscription {
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
			js, err := json.Marshal(&pkg.WSMessage{Type: "state", Data: sta})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGState(...) -> sta :", sta)

			/* SEND WSMessage AS JSON STRING */
			duc.WriteDataOut(string(js))

		},
	}
}

/* SUBSCRIPTION -> HEADER   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGHeader( /* TODO: PASS IN USER ROLE */ ) pkg.MQTTSubscription {
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
			js, err := json.Marshal(&pkg.WSMessage{Type: "header", Data: hdr})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGHeader(...) -> hdr :", hdr)

			/* SEND WSMessage AS JSON STRING */
			duc.WriteDataOut(string(js))

		},
	}
}

/* SUBSCRIPTION -> CONFIG */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGConfig( /* TODO: PASS IN USER ROLE */ ) pkg.MQTTSubscription {
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
			js, err := json.Marshal(&pkg.WSMessage{Type: "config", Data: cfg})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGConfig(...) -> cfg :", cfg)

			/* SEND WSMessage AS JSON STRING */
			duc.WriteDataOut(string(js))

		},
	}
}

/* SUBSCRIPTION -> EVENT */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGEvent( /* TODO: PASS IN USER ROLE */ ) pkg.MQTTSubscription {
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
			js, err := json.Marshal(&pkg.WSMessage{Type: "event", Data: evt})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DeviceUserClient_SIGEvent(...) -> evt :", evt)

			/* SEND WSMessage AS JSON STRING */
			duc.WriteDataOut(string(js))

		},
	}
}

/* SUBSCRIPTION -> SAMPLE   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGSample( /* TODO: PASS IN USER ROLE */ ) pkg.MQTTSubscription {
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
			js, err := json.Marshal(&pkg.WSMessage{Type: "sample", Data: smp})
			if err != nil {
				pkg.LogErr(err)
			} else {
				// pkg.Json("MQTTSubscription_DeviceUserClient_SIGSample:", js)
				/* SEND WSMessage AS JSON STRING */
				duc.WriteDataOut(string(js))
			}

		},
	}
}

/* SUBSCRIPTION -> DIAG SAMPLE   */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGDiagSample( /* TODO: PASS IN USER ROLE */ ) pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGDiagSample(),
		Handler: func(c phao.Client, msg phao.Message) {
			/* WRANGLE WS DATA */
			/* SEND WS DATA */
			// duc.WriteDataOut(string(js))
		},
	}
}

/* SUBSCRIPTIONS -> MESSAGE LIMIT TEST ***TODO: REMOVE AFTER DEVELOPMENT*** */
func (duc *DeviceUserClient) MQTTSubscription_DeviceUserClient_SIGMsgLimit() pkg.MQTTSubscription {
	return pkg.MQTTSubscription{

		Qos:   0,
		Topic: duc.MQTTTopic_SIGMsgLimit(),
		Handler: func(c phao.Client, msg phao.Message) {

			/* PARSE MsgLimit IN CMDARCHIVE */
			kafka := MsgLimit{}
			if err := json.Unmarshal(msg.Payload(), &kafka); err != nil {
				pkg.LogErr(err)
			}

			/* CREATE JSON WSMessage STRUCT */
			js, err := json.Marshal(&pkg.WSMessage{Type: "msg_limit", Data: kafka})
			if err != nil {
				pkg.LogErr(err)
			} // pkg.Json("MQTTSubscription_DemoDeviceClient_SIGMsgLimit(...) -> kafka :", kafka)

			/* SEND WSMessage AS JSON STRING */
			duc.WriteDataOut(string(js))
		},
	}
}

/* PUBLICATIONS ******************************************************************************************/
/* NONE; WE DON'T DO THAT;
ALL COMMANDS SENT TO THE DEVICE GO THROUGH HTTP HANDLERS
*/

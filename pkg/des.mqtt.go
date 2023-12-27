/* Data Exchange Server (DES) is a component of the Datacan Data2Desk (D2D) Platform.
License:

	[PROPER LEGALESE HERE...]

	INTERIM LICENSE DESCRIPTION:
	In spirit, this license:
	1. Allows <Third Party> to use, modify, and / or distributre this software in perpetuity so long as <Third Party> understands:
		a. The software is porvided as is without guarantee of additional support from DataCan in any form.
		b. The software is porvided as is without guarantee of exclusivity.

	2. Prohibits <Third Party> from taking any action which might interfere with DataCan's right to use, modify and / or distributre this software in perpetuity.
*/

package pkg

import (
	"encoding/json"
	"fmt"
	"time"

	phao "github.com/eclipse/paho.mqtt.golang"
)

type DESMQTTClient struct {
	MQTTUser     string
	MQTTPW       string
	MQTTClientID string
	phao.ClientOptions
	phao.Client
	Subs []MQTTSubscription
}

func (desm *DESMQTTClient) DESMQTTClient_Connect(falseToResub, autoReconn bool) (err error) {

	/* CREATE MQTT CLEITN OPTIONS */
	desm.ClientOptions = *phao.NewClientOptions()
	desm.AddBroker(MQTT_BROKER)
	desm.SetUsername(desm.MQTTUser)
	desm.SetPassword(desm.MQTTPW)
	desm.SetClientID(desm.MQTTClientID)
	desm.SetPingTimeout(time.Second * 20) // Must be 1.5 x greater than Keep-Alive
	desm.SetKeepAlive(time.Second * 10)
	desm.SetAutoReconnect(autoReconn)
	desm.SetCleanSession(falseToResub) // FALSE to ensure subscriptions are active on reconnect
	desm.SetMaxReconnectInterval(time.Second * 10)
	desm.OnConnect = func(c phao.Client) {
		// fmt.Printf("\n(desm *DESMQTTClient) DESMQTTClient_Connect( ): %s -> connected...\n", desm.MQTTClientID)
	}
	desm.OnConnectionLost = func(c phao.Client, err error) {
		if ( err.Error() != "EOF" ) {
			fmt.Printf(
				"\n(desm *DESMQTTClient) DESMQTTClient_Connect( ): %s -> connection lost...\n%s\n", 
				desm.MQTTClientID,
				err.Error(),
			)
		}
	}
	desm.DefaultPublishHandler = func(c phao.Client, msg phao.Message) {
		// fmt.Printf(
		// 	"\n(desm *DESMQTTClient) DESMQTTClient_Connect( ): %s\nDefault Handler:\nTopic: %s:\n\n",
		// 	desm.MQTTClientID,
		// 	msg.Topic(),
		// )
	} // fmt.Printf("\n(desm *DESMQTTClient)RegisterDESMQTTClient( ... ) -> desm.ClientID: %s\n", desm.ClientID)

	/*Cerate MQTT Client*/
	c := phao.NewClient(&desm.ClientOptions)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("\n(desm *DESMQTTClient) DESMQTTClient_Connect( ): %s -> FAILED!\n%s\n", desm.MQTTClientID, token.Error())
		return token.Error()
	}

	desm.Client = c

	return err
}
/* TODO: FIND OUT WHY THIS NEVER RETURNS... */
func (desm *DESMQTTClient) DESMQTTClient_Disconnect() {
	// desm.WG.Wait()
	desm.Client.Disconnect(10) // Wait 10 milliseconds
}

/* ALL MQTT SUBSCRIPTIONS ON THE DES ARE MANAGED USING THIS STRUCTURE */
type MQTTSubscription struct {
	Topic   string
	Qos     byte
	Handler phao.MessageHandler
}

func (sub MQTTSubscription) Sub(client DESMQTTClient) {
	token := client.Subscribe(sub.Topic, sub.Qos, sub.Handler)
	// token.WaitTimeout(time.Millisecond * 100)
	token.Wait() // fmt.Printf("\nSubscribed: %s to:\t%s\n\n", client.MQTTClientID, sub.Topic)
}
func (sub MQTTSubscription) UnSub(client DESMQTTClient) {
	token := client.Unsubscribe(sub.Topic)
	// token.WaitTimeout(time.Millisecond * 100)
	token.Wait() // fmt.Printf("\nUnsubscribed: %s from:\t%s\n", client.MQTTClientID, sub.Topic)
}

/* ALL MQTT PUBLICATIONS ON THE DES ARE MANAGED USING THIS STRUCTURE */
type MQTTPublication struct {
	Topic    string
	Qos      byte
	Retained bool
	Message  string
	WaitMS   int64
}

func (pub MQTTPublication) Pub(client DESMQTTClient) {

	// pkg.Json("DEMO_PublishSIG_MQTTSample(...) ->  des.MQTTPublication -> Pub(client phao.Client):", client)
	if client.Client == nil {
		fmt.Printf("\n (pub MQTTPublication) Pub( NO CLIENT )")
	} else {
		if token := client.Publish(
			pub.Topic,
			pub.Qos,
			pub.Retained,
			pub.Message,
		); token.Wait() && token.Error() != nil {
			LogErr(token.Error())
		}
	}
}

func ModelToJSONB(mod interface{}) (jsonb []byte) {

	jsonb, err := json.Marshal(mod)
	if err != nil {
		LogErr(err)
	}
	// fmt.Printf("\n%s\n", string(jsonb))
	return
}

func ModelToJSONString(mod interface{}) (msg string) {

	js, err := json.Marshal(mod)
	if err != nil {
		LogErr(err)
	}
	// fmt.Printf("\n%s\n", string(js))
	return string(js)
}

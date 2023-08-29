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
}

type MQTTClientsMap map[string]DESMQTTClient
var MQTTDevClients  = make(MQTTClientsMap)
var MQTTUserClients  = make(MQTTClientsMap)
var MQTTDemoClients  = make(MQTTClientsMap)

func (desm *DESMQTTClient) DESMQTTClient_Connect() (err error) {

	/*Cerate MQTT Client Options*/
	desm.ClientOptions = *phao.NewClientOptions()
	desm.AddBroker(MQTT_BROKER)
	desm.SetUsername(desm.MQTTUser)
	desm.SetPassword(desm.MQTTPW)
	desm.SetClientID(desm.MQTTClientID)
	desm.SetKeepAlive(time.Second * 60)
	desm.SetAutoReconnect(true)
	desm.SetMaxReconnectInterval(time.Second * 60)
	desm.OnConnect = func(c phao.Client) {
		fmt.Printf(
			"\nDESMQTTClient: %s connected...\n", desm.MQTTClientID,
		)
	}
	desm.OnConnectionLost = func(c phao.Client, err error) {
		fmt.Printf(
			"\nDESMQTTClient: %s connection lost...\n%s\n", desm.MQTTClientID,
			err.Error(),
		)
	}
	desm.DefaultPublishHandler = func(c phao.Client, msg phao.Message) {
		fmt.Printf(
			"\nDESMQTTClient: %s\nDefault Handler:\nTopic: %s:\nMessage:\n%s\n\n",
			desm.MQTTClientID,
			msg.Topic(),
			msg.Payload(),
		)
	} // fmt.Printf("\n(desm *DESMQTTClient)RegisterDESMQTTClient( ... ) -> desm.ClientID: %s\n", desm.ClientID)

	/*Cerate MQTT Client*/
	c := phao.NewClient(&desm.ClientOptions)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("\nDESMQTTClient: %s connection failed...\n%s\n", desm.MQTTClientID, token.Error())
		return token.Error()
	}

	desm.Client = c

	return err
}
func (desm *DESMQTTClient) DESMQTTClient_Disconnect() (err error) {
	desm.Client.Disconnect(0) // Wait 10 milliseconds
	return err
}

type MQTTSubscription struct {
	Topic   string
	Qos     byte
	Handler phao.MessageHandler
}

func (sub MQTTSubscription) Sub(client DESMQTTClient) {

	token := client.Subscribe(sub.Topic, sub.Qos, sub.Handler)
	// token.WaitTimeout(time.Millisecond * 100)
	token.Wait()

	fmt.Printf("\nSubscribed: %s to:\t%s\n\n", client.MQTTClientID, sub.Topic)
}
func (sub MQTTSubscription) UnSub(client DESMQTTClient) {

	token := client.Unsubscribe(sub.Topic)

	// token.WaitTimeout(time.Millisecond * 100)
	token.Wait()
	fmt.Printf("\nUnsubscribed: %s from:\t%s\n", client.MQTTClientID, sub.Topic)
}

type MQTTPublication struct {
	Topic    string
	Qos      byte
	Retained bool
	Message  string
	WaitMS   int64
}

func (pub MQTTPublication) Pub(client DESMQTTClient) bool {

	// pkg.Json("DEMO_PublishSIG_MQTTSample(...) ->  des.MQTTPublication -> Pub(client phao.Client):", client)
	token := client.Publish(
		pub.Topic,
		pub.Qos,
		pub.Retained,
		pub.Message,
	)

	if pub.WaitMS == 0 {
		// return token.Wait()
		x := token.Wait()
		// pkg.Json("DEMO_PublishSIG_MQTTSample(...) ->  des.MQTTPublication -> token.Wait():", x)
		return x
	} else {
		return token.WaitTimeout(time.Millisecond * 100)
	}
}

func MakeMQTTMessage(mqtt interface{}) (msg string) {

	js, err := json.Marshal(mqtt)
	if err != nil {
		Trace(err)
	}
	return string(js)
}



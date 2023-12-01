package c001v001

import "fmt"

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

/* DEVELOPMENT TOPIC ***TODO: REMOVE AFTER DEVELOPMENT*** */
func (device *Device) MQTTTopic_SIGMsgLimit() (topc string) {
	/*** TODO: REMOVE AFTER DEVELOPMENT ***/
	return fmt.Sprintf("%s/msg_limit", device.MQTTTopic_SIGRoot())
}
/* DEVELOPMENT TOPIC ***TODO: REMOVE AFTER DEVELOPMENT*** */
func (device *Device) MQTTTopic_SIGTestOLS() (topc string) {
	/*** TODO: REMOVE AFTER DEVELOPMENT ***/
	return fmt.Sprintf("%s/test_ols", device.MQTTTopic_SIGRoot())
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

/* DEVELOPMENT TOPIC ***TODO: REMOVE AFTER DEVELOPMENT*** */
func (device *Device) MQTTTopic_CMDMsgLimit() (topc string) {
	/*** TODO: REMOVE AFTER DEVELOPMENT ***/
	return fmt.Sprintf("%s/msg_limit", device.MQTTTopic_CMDRoot())
}
/* DEVELOPMENT TOPIC ***TODO: REMOVE AFTER DEVELOPMENT*** */
func (device *Device) MQTTTopic_CMDTestOLS() (topc string) {
	/*** TODO: REMOVE AFTER DEVELOPMENT ***/
	return fmt.Sprintf("%s/test_ols", device.MQTTTopic_CMDRoot())
}

/* MQTT TOPICS - DES MESSAGE */
func (device *Device) MQTTTopic_DESDeviceClientPing() (topic string) {
	return fmt.Sprintf("%s/des_ping", device.MQTTTopic_DESRoot())
}

func (device *Device) MQTTTopic_DESDevicePing() (topic string) {
	return fmt.Sprintf("%s/ping", device.MQTTTopic_DESRoot())
}

package c001v001

import (
	"fmt"

	"github.com/leehayford/des/pkg"
)

const DEVICE_CLASS = "001"
const DEVICE_VERSION = "001"

type Device struct {
	Job `json:"job"` // The active job for this device ( last job if it has ended )
	pkg.DESRegistration `json:"reg"`
	pkg.DESMQTTClient `json:"-"`
}


func GetDeviceList() (devices []pkg.DESRegistration, err error) {

	/*
		WHERE A DEVICE HAS MORE THAN ONE REGISTRATION RECORD
		WE WANT THE LATEST
	*/
	devSubQry := pkg.DES.DB.
		Table("des_devs").
		Where("des_dev_class = '001' AND des_dev_version = '001' ").
		Select("des_dev_serial, MAX(des_dev_reg_time) AS max_time").
		Group("des_dev_serial")
	
	/*
		WHERE A MORE THAN ONE JOB IS ACTIVE ( des_job_end = 0 )
		WE WANT THE LATEST
	*/
	jobSubQry := pkg.DES.DB.
		Table("des_jobs").
		Where("des_jobs.des_job_end = 0").
		Select("des_job_id, MAX(des_job_start) AS max_start").
		Group("des_job_id")

	jobQry := pkg.DES.DB.
		Table("des_jobs").
		Select("*").
		Joins(`JOIN ( ? ) jobs
			ON des_jobs.des_job_id = jobs.des_job_id
			AND des_jobs.des_job_start = jobs.max_start`,
			jobSubQry)
		
	qry := pkg.DES.DB.
		Table("des_devs").
		Select(" des_devs.*, j.* ").
		Joins(`JOIN ( ? ) d 
			ON des_devs.des_dev_serial = d.des_dev_serial 
			AND des_devs.des_dev_reg_time = d.max_time`,
			devSubQry).
		Joins(`JOIN ( ? ) j
			ON des_devs.des_dev_id = j.des_job_dev_id`,
			jobQry).
		Order("des_devs.des_dev_serial DESC")

		res := qry.Scan(&devices)
		err = res.Error
		return 
}


/* Create a device client for all registered devices */
func MQTTDeviceClient_CreateAndConnectAll() (err error) {

	drs, err := GetDeviceList() 
	if err != nil {
		return pkg.Trace(err)
	}
	pkg.Json("GetDeviceList():", drs)

	for _, dr := range drs {
		device := Device{
			DESRegistration:     dr,
			Job:           Job{DESRegistration: dr},
			DESMQTTClient: pkg.DESMQTTClient{},
		}
		if err = device.MQTTDeviceClient_Connect(); err != nil {
			return pkg.Trace(err)
		}
	}

	return err
}

/* MQTT TOPICS - SIGNAL */
func (device *Device) MQTTTopic_SIGAdmin() (topic string) {
	return fmt.Sprintf("%s/%s/%s/sig/admin",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_SIGConfig() (topic string) {
	return fmt.Sprintf("%s/%s/%s/sig/config",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_SIGEvent() (topic string) {
	return fmt.Sprintf("%s/%s/%s/sig/event",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_SIGSample() (topic string) {
	return fmt.Sprintf("%s/%s/%s/sig/sample",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_SIGDiagSample() (topic string) {
	return fmt.Sprintf("%s/%s/%s/sig/diag_sample",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}

/* MQTT TOPICS - COMMAND */
func (device *Device) MQTTTopic_CMDAdmin() (topic string) {
	return fmt.Sprintf("%s/%s/%s/cmd/admin",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_CMDConfig() (topic string) {
	return fmt.Sprintf("%s/%s/%s/cmd/config",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_CMDEvent() (topic string) {
	return fmt.Sprintf("%s/%s/%s/cmd/event",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_CMDSample() (topic string) {
	return fmt.Sprintf("%s/%s/%s/cmd/sample",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
func (device *Device) MQTTTopic_CMDDiagSample() (topic string) {
	return fmt.Sprintf("%s/%s/%s/cmd/diag_sample",
		device.DESDevClass,
		device.DESDevVersion,
		device.DESDevSerial,
	)
}
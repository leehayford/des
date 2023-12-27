
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
	"time"
)

type DESDev struct {
	DESDevID int64 `gorm:"unique; primaryKey" json:"des_dev_id"`	
	
	DESDevRegTime   int64  `gorm:"not null" json:"des_dev_reg_time"`
	DESDevRegAddr   string `json:"des_dev_reg_addr"`
	DESDevRegUserID string `gorm:"not null; varchar(36)" json:"des_dev_reg_user_id"`
	DESDevRegApp    string `gorm:"not null; varchar(36)" json:"des_dev_reg_app"`

	DESDevSerial  string   `gorm:"not null; varchar(10)" json:"des_dev_serial"`
	DESDevVersion string   `gorm:"not null; varchar(3)" json:"des_dev_version"`
	DESDevClass   string   `gorm:"not null; varchar(3)" json:"des_dev_class"`
	DESJobs []DESJob `gorm:"foreignKey:DESJobDevID" json:"-"`
	User User `gorm:"foreignKey:DESDevRegUserID" json:"-"`
}
func WriteDESDevice(device DESDev) (err error) {
	device.DESDevID = 0
	res := DES.DB.Create(&device)
	return res.Error
}

/* NOT IN USE 20231227 */
func GetDESDevList(devices *[]DESDev) (err error) {

	/*
		WHERE A DEVICE HAS MORE THAN ONE REGISTRATION RECORD
		WE WANT THE LATEST
	*/
	subQry := DES.DB.
		Table("des_devs").
		Select(`des_dev_serial, MAX(des_dev_reg_time) AS max_time`).
		Group("des_dev_serial")

	qry := DES.DB.
		Select(" * ").
		Joins(`JOIN ( ? ) x 
		ON des_devs.des_dev_serial = x.des_dev_serial 
		AND des_devs.des_dev_reg_time = x.max_time`,
			subQry).
		Order("des_devs.des_dev_serial DESC")

		res := qry.Find(&devices)
		return res.Error
}


type DESDevError struct {
	DESDevErrID int64 `gorm:"unique; primaryKey" json:"des_dev_err_id"`	
	DESDevErrTime   int64  `gorm:"not null" json:"des_dev_err_time"`
	DESDevErrMsg	string `gorm:"not null" json:"des_dev_err_msg"`
	DESDevErrJson	string `json:"des_dev_err_json"` // If there is an object associated with the error, this is the JSON string thereof.
	DESErrDevID int64   `json:"des_err_dev_id"`
	DESDev DESDev `gorm:"foreignKey:DESErrDevID" json:"-"`
}
func WriteDESDevError(dev_err DESDevError) (err error) {
	dev_err.DESDevErrID = 0
	res := DES.DB.Create(&dev_err)
	return res.Error
}
func GetDESDevErrorList(devices *[]DESDev) (err error) {

	/*
		WHERE A DEVICE HAS MORE THAN ONE REGISTRATION RECORD
		WE WANT THE LATEST
	*/
	subQry := DES.DB.
		Table("des_devs").
		Select(`des_dev_serial, MAX(des_dev_reg_time) AS max_time`).
		Group("des_dev_serial")

	qry := DES.DB.
		Select(" * ").
		Joins(`JOIN ( ? ) x 
		ON des_devs.des_dev_serial = x.des_dev_serial 
		AND des_devs.des_dev_reg_time = x.max_time`,
			subQry).
		Order("des_devs.des_dev_serial DESC")

		res := qry.Find(&devices)
		return res.Error
}

type ObjError struct {
	Msg string      `json:"msg"`
}

func (des_dev DESDev)MakeDESDevError(msg string, obj interface{}) (dev_err DESDevError, err error) {

	t := time.Now().UTC().UnixMilli()

	js, err := ModelToJSONString(obj)
	if err != nil {
		LogErr(err)
		b, _ := json.Marshal(&ObjError{Msg: "Model could not be converted to json string."})
		js = string(b)
	}

	dev_err = DESDevError{
		DESDevErrTime: t,
		DESDevErrMsg: msg,
		DESDevErrJson: js,
		DESErrDevID: des_dev.DESDevID,
	}

	err = WriteDESDevError(dev_err)

	return
}



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
	"runtime"
	"time"
)

const ERR_DB_EXISTS string = "Database already exists"

const ERR_FILE_NAME_EMPTY string = "File name is empty"

const ERR_AUTH string = ""
const ERR_AUTH_INVALID_SESSION string = "Invalid user session ID"

const ERR_AUTH_SUPER string = "You must be super to perform this action"
const ERR_AUTH_ADMIN string = "You must be an administrator to perform this action"
const ERR_AUTH_OPERATOR string = "You must be an operator to perform this action"
const ERR_AUTH_VIEWER string = "You must be a viewer to perform this action"
const ERR_AUTH_USER_NOT_FOUND string = "User not found"

const ERR_SRC_TIME_PAST string = "Invalid message source; time has too long since passed"
const ERR_SRC_TIME_FUTURE string = "Invalid message source; time has not yet come to pass"
const ERR_INVALID_SRC_SIG string = "Invalid device message source data"
const ERR_INVALID_SRC_CMD string = "Invalid user message source data"
const ERR_INVALID_SRC_OP_CODE_CMD string = "Invalid user message op code"

const ERR_MQTT_DEVICE_CONN string = "Device not connected to broker"

type DESError struct {
	DESErrID   int64  `gorm:"unique; primaryKey" json:"des_dev_err_id"`
	DESErrTime int64  `gorm:"not null" json:"des_err_time"`
	DESErrMsg  string `gorm:"not null" json:"des_err_msg"`
	DESErrJson string `json:"des_err_json"`
	DESErrRef  string `json:"des_err_ref"`
}
type DESErrObj struct {
	Msg string `json:"msg"`
}

func WriteDESError(des_err DESError) (err error) {
	des_err.DESErrID = 0
	res := DES.DB.Create(&des_err)
	return res.Error
}
func GetDESErrorList(ref string) (errs []DESError, err error) {

	qry := DES.DB.
		Table("des_errors").
		Select("*").
		Where("des_errors.des_err_ref = ?", ref).
		Order(".des_err_time DESC")

	res := qry.Find(&errs)
	err = res.Error
	return
}
func LogDESError(ref, msg string, obj interface{}) (des_err DESError, err error) {

	t := time.Now().UTC().UnixMilli()

	js, err := ModelToJSONString(obj)
	if err != nil {
		LogErr(err)
		b, _ := json.Marshal(&DESErrObj{Msg: "Model could not be converted to json string."})
		js = string(b)
	}

	des_err = DESError{
		DESErrTime: t,
		DESErrMsg:  msg,
		DESErrJson: js,
		DESErrRef:  ref,
	}

	err = WriteDESError(des_err)

	return
}

func LogErr(err error) error {
	pc, file, line, _ := runtime.Caller(1)
	name := runtime.FuncForPC(pc).Name()

	fmt.Printf("***ERROR***\n\tFile :\t%s\n\tFunc :\t%s\n\tLine :\t%d\n\n\t%s\n\n***********\n\n", file, name, line, err.Error())
	/* TODO: LOG err TO FILE */

	return err
}

func LogChk(msg string) {
	pc, file, line, _ := runtime.Caller(1)
	name := runtime.FuncForPC(pc).Name()

	fmt.Printf("***OK***\n\tFile :\t%s\n\tFunc :\t%s\n\tLine :\t%d\n\n\t%s\n\n***********\n\n", file, name, line, msg)
	/* TODO: LOG msg TO FILE */
}

func Json(name string, v any) {
	js, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		LogErr(err)
	}
	fmt.Printf("\nJSON: %s:\n%s\n", name, string(js))
}

type DESMessageSource struct {
	Time   int64  `json:"time"`
	Addr   string `json:"addr"`
	UserID string `json:"user_id"`
	App    string `json:"app"`
}

func (src *DESMessageSource) ValidateSRC_CMD(dev_src DESMessageSource, uid string, mod interface{}) (err error) {

	usrc, err := GetUserReferenceSRC(uid)
	if err != nil {
		return
	}

	if err = ValidateUnixMilli(src.Time); err != nil {
		_, err = LogDESError(usrc.UserID, err.Error(), mod)
		return
	}

	/* COMMANDS CAN NOT ORIGINATE FROM DEVICES */
	if src.Addr == dev_src.Addr {
		LogDESError(dev_src.UserID, ERR_INVALID_SRC_SIG, mod)
		src.Addr = usrc.Addr
	}

	/* COMMANDS CAN NOT ORIGINATE FROM DEVICES */
	if src.App == dev_src.App {
		LogDESError(dev_src.UserID, ERR_INVALID_SRC_SIG, mod)
		src.App = usrc.App
	}

	/* COMMANDS CAN NOT ORIGINATE FROM DEVICES */
	
	fmt.Printf("(*DESMessageSource) ValidateSRC_CMD(): -> src.%s == dev_src.%s\n", src.UserID, dev_src.UserID)
	if src.UserID == dev_src.UserID {
		LogDESError(dev_src.UserID, ERR_INVALID_SRC_SIG, mod)
		src.UserID = usrc.UserID
	}

	return
}
func (src *DESMessageSource) ValidateSRC_SIG(dev_src DESMessageSource, mod interface{}) (err error) {

	if err = ValidateUnixMilli(src.Time); err != nil {
		_, err = LogDESError(dev_src.UserID, err.Error(), mod)
		return
	}
	if src.Addr != dev_src.Addr {
		LogDESError(dev_src.UserID, ERR_INVALID_SRC_SIG, mod)
		src.Addr = dev_src.Addr
	}

	return
}

/* VALIDATE UNIX MILLI */
const MIN_TIME = 946710000

func ValidateUnixMilli(t int64) (err error) {
	now := time.Now().UTC()
	y := now.Year()
	m := now.Month()
	// d := now.Day( )

	test := time.UnixMilli(t)
	ty := test.Year()
	tm := test.Month()
	// td := test.Day()

	if t < MIN_TIME {
		return fmt.Errorf(ERR_SRC_TIME_PAST)
	} else if (ty - y) > 1 {
		return fmt.Errorf(ERR_SRC_TIME_FUTURE)
	} else if (ty-y) > 0 && (tm-m+12) > 0 {
		return fmt.Errorf(ERR_SRC_TIME_FUTURE)
	}
	return
}

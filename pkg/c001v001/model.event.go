package c001v001

import (
	"errors"

	"github.com/leehayford/des/pkg"
)

/*
EVENT - AS WRITTEN TO JOB DATABASE
*/
type Event struct {
	EvtID int64 `gorm:"unique; primaryKey" json:"-"`

	EvtTime   int64  `gorm:"not null" json:"evt_time"`
	EvtAddr   string `json:"evt_addr"`
	EvtUserID string `gorm:"not null; varchar(36)" json:"evt_user_id"`
	EvtApp    string `gorm:"not null; varchar(36)" json:"evt_app"`

	EvtCode  int32    `json:"evt_code"`
	EvtTitle string   `gorm:"varchar(36)" json:"evt_title"`
	EvtMsg   string   `gorm:"varchar(512)" json:"evt_msg"`
	EvtType  EventTyp `gorm:"foreignKey:EvtCode; references:EvtTypCode" json:"-"`
}

func WriteEVT(evt Event, dbc *pkg.DBClient) (err error) {

	/* WHEN Write IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING
	WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	dbc.WG.Add(1)
	evt.EvtID = 0
	res := dbc.Create(&evt)
	evt.EvtID = 0
	dbc.WG.Done()

	return res.Error
}

/*
EVENT - AS STORED IN DEVICE FLASH
*/
func (evt Event) EventToBytes() (out []byte) {

	out = append(out, pkg.Int64ToBytes(evt.EvtTime)...)
	out = append(out, pkg.StringToNBytes(evt.EvtAddr, 36)...)
	out = append(out, pkg.StringToNBytes(evt.EvtUserID, 36)...)
	out = append(out, pkg.StringToNBytes(evt.EvtApp, 36)...)

	out = append(out, pkg.Int32ToBytes(evt.EvtCode)...)
	out = append(out, pkg.StringToNBytes(evt.EvtTitle, 36)...)
	out = append(out, pkg.StringToNBytes(evt.EvtMsg, len(evt.EvtMsg))...)

	return
}
func (evt *Event) EventFromBytes(b []byte) {

	evt = &Event{

		EvtTime:   pkg.BytesToInt64_L(b[0:8]),
		EvtAddr:   pkg.StrBytesToString(b[8:44]),
		EvtUserID: pkg.StrBytesToString(b[44:80]),
		EvtApp:    pkg.StrBytesToString(b[80:116]),

		EvtCode:  pkg.BytesToInt32_L(b[116:120]),
		EvtTitle: pkg.StrBytesToString(b[120:156]),
		EvtMsg:   pkg.StrBytesToString(b[156:]),
	}
	//  pkg.Json("(demo *DemoDeviceClient)MakeEvtFromBytes() -> evt", evt)
	return
}

/*
EVENT - DEFAULT VALUES
*/
func (evt *Event) DefaultSettings_Event(job pkg.DESRegistration) {
	evt.EvtTime = job.DESJobRegTime
	evt.EvtAddr = job.DESJobRegAddr
	evt.EvtUserID = job.DESJobRegUserID
	evt.EvtApp = job.DESJobRegApp
	evt.EvtCode = OP_CODE_DES_REG_REQ
	evt.EvtTitle = "DEVICE REGISTRATION REQUESTED"
	evt.EvtMsg = ""
}

/*
EVENT - VALIDATE FIELDS
  - INVALID FIELDS ARE MADE VALID
*/
func (evt *Event) Validate() {
	/* TODO: SET ACCEPTABLE LIMITS FOR THE REST OF THE CONFIG SETTINGS */

	evt.EvtAddr = pkg.ValidateStringLength(evt.EvtAddr, 36)
	evt.EvtUserID = pkg.ValidateStringLength(evt.EvtUserID, 36)
	evt.EvtApp = pkg.ValidateStringLength(evt.EvtApp, 36)

	evt.EvtTitle = pkg.ValidateStringLength(evt.EvtTitle, 36)
	evt.EvtMsg = pkg.ValidateStringLength(evt.EvtMsg, 512)
}

/*
EVENT - VALIDATE MQTT SIG FROM DEVICE
*/
func (evt *Event) SIGValidate(device *Device) (err error) {

	if err = pkg.ValidateUnixMilli(evt.EvtTime); err != nil {
		return pkg.LogErr(err)
	}
	if evt.EvtAddr != device.DESDevSerial { 
		pkg.LogErr(errors.New("\nInvalid device.EVT.EvtAddr."))
		evt.EvtAddr = device.DESDevSerial 
	}
	if evt.EvtCode > MAX_OP_CODE && 
		evt.EvtCode <= MAX_STATUS_CODE && 
		evt.EvtUserID != device.DESU.ID.String() {
		pkg.LogErr(errors.New("\nInvalid device.DESU: wrong user ID."))
		evt.EvtUserID = device.DESU.ID.String()
	}
	evt.Validate()

	return
}

type EventTyp struct {
	EvtTypID   int64  `gorm:"unique; primaryKey" json:"-"`
	EvtTypCode int32  `gorm:"unique" json:"evt_typ_code"`
	EvtTypName string `json:"evt_typ_name"`
	EvtTypDesc string `json:"evt_typ_desc"`
}

func WriteETYP(etyp EventTyp, dbc *pkg.DBClient) (err error) {

	/* WHEN Write IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING
	WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	dbc.WG.Add(1)
	etyp.EvtTypID = 0
	res := dbc.Create(&etyp)
	dbc.WG.Done()

	return res.Error
}

/* TODO: TEST MAP IMPLEMENTATION */
// type EventTypesMap map[string]EventTyp
// var EventTypes = make(EventTypesMap)
// func LoadEventTypes() {
// 	EventTypes["req_reg"] = EventTyp{EvtTypCode: STATUS_DES_REG_REQ, EvtTypName: "DEVICE REGISTRATION REQUESTED"}
// 	EventTypes["reg"] = EventTyp{EvtTypCode: STATUS_DES_REGISTERED, EvtTypName: "DEVICE REGISTERED"}
// }

var EVENT_TYPES = []EventTyp{

	/* DEVICE OPERATION EVENT TYPES: 0 - 999 */
	{EvtTypCode: OP_CODE_DES_REG_REQ, EvtTypName: "DEVICE REGISTRATION REQUESTED"},
	{EvtTypCode: OP_CODE_DES_REGISTERED, EvtTypName: "DEVICE REGISTERED"},
	{EvtTypCode: OP_CODE_JOB_ENDED, EvtTypName: "JOB ENDED"},
	{EvtTypCode: OP_CODE_JOB_START_REQ, EvtTypName: "START JOB REQUESTED"},
	{EvtTypCode: OP_CODE_JOB_STARTED, EvtTypName: "JOB STARTED"},
	{EvtTypCode: OP_CODE_JOB_END_REQ, EvtTypName: "END JOB REQUESTED"},
	{EvtTypCode: OP_CODE_JOB_OFFLINE_START, EvtTypName: "JOB STARTED OFFLINE"},
	{EvtTypCode: OP_CODE_JOB_OFFLINE_END, EvtTypName: "JOB ENDED OFFLINE"},
	{EvtTypCode: OP_CODE_GPS_ACQ, EvtTypName: "DEVICE ACQUIRING GPS"},

	/* ALARM EVENT TYPES 1000 -1999 */
	{EvtTypCode: STATUS_BAT_HIGH_AMP, EvtTypName: "ALARM HIGH BATTERY CURRENT"},
	{EvtTypCode: STATUS_BAT_LOW_VOLT, EvtTypName: "ALARM LOW BATTERY VOLTAGE"},
	{EvtTypCode: STATUS_MOT_HIGH_AMP, EvtTypName: "ALARM HIGH MOTOR CURRENT"},
	{EvtTypCode: STATUS_MAX_PRESSURE, EvtTypName: "ALARM MAX PRESSURE"},
	{EvtTypCode: STATUS_HFS_MAX_FLOW, EvtTypName: "ALARM HFS MAX FLOW"},
	{EvtTypCode: STATUS_HFS_MAX_PRESS, EvtTypName: "ALARM HFS MAX PRESSURE"},
	{EvtTypCode: STATUS_HFS_MAX_DIFF, EvtTypName: "ALARM HFS MAX DIFF-PRESSURE"},
	{EvtTypCode: STATUS_LFS_MAX_FLOW, EvtTypName: "ALARM LFS MAX FLOW"},
	{EvtTypCode: STATUS_LFS_MAX_PRESS, EvtTypName: "ALARM LFS MAX PRESSURE"},
	{EvtTypCode: STATUS_LFS_MAX_DIFF, EvtTypName: "ALARM LFS MAX DIFF-PRESSURE"},

	// 	/* ANNOTATION EVENT TYPES 2000 - 65535 */
	{EvtTypCode: 2000, EvtTypName: "OPERATOR COMMENT"},
	{EvtTypCode: 2001, EvtTypName: "REPORT COMMENT"},
}

func GetEventTypeByCode(code int32) (name string) {
	for i := range EVENT_TYPES {
		if EVENT_TYPES[i].EvtTypCode == code {
			name = EVENT_TYPES[i].EvtTypName
			break
		}
	}
	return
}

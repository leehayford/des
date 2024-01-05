package c001v001

import (
	"sync"
	"github.com/leehayford/des/pkg"
)

/*
EVENT - AS WRITTEN TO JOB DATABASE
*/
type Event struct {
	// EvtID   int64  `gorm:"unique; primaryKey" json:"-"` // POSTGRES
	EvtID int64 `gorm:"autoIncrement" json:"-"` // SQLITE

	EvtTime   int64  `gorm:"not null" json:"evt_time"`
	EvtAddr   string `gorm:"varchar(36)" json:"evt_addr"`
	EvtUserID string `gorm:"not null; varchar(36)" json:"evt_user_id"`
	EvtApp    string `gorm:"varchar(36)" json:"evt_app"`

	EvtCode  int32    `json:"evt_code"`
	EvtTitle string   `gorm:"varchar(36)" json:"evt_title"`
	EvtMsg   string   `gorm:"varchar(512)" json:"evt_msg"`
	EvtType  EventTyp `gorm:"foreignKey:EvtCode; references:EvtTypCode" json:"-"`
}

func WriteEVT(evt Event, jdbc *pkg.JobDBClient) (err error) {

	/* WHEN Write IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING
	WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	if jdbc.RWM == nil {
		jdbc.RWM = &sync.RWMutex{}
	}
	jdbc.RWM.Lock()
	res := jdbc.Create(&evt)
	jdbc.RWM.Unlock()

	return res.Error
}
func ReadLastEVT(evt *Event, jdbc *pkg.JobDBClient) (err error) {
	
	/* WHEN Read IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING
	WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	if jdbc.RWM == nil {
		jdbc.RWM = &sync.RWMutex{}
	}
	jdbc.RWM.Lock()
	res := jdbc.Last(&evt)
	jdbc.RWM.Unlock()

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

	if evt.EvtTitle == "" {
		evt.EvtTitle = GetEventTypeByCode(evt.EvtCode)
	}

	evt.EvtTitle = pkg.ValidateStringLength(evt.EvtTitle, 36)
	evt.EvtMsg = pkg.ValidateStringLength(evt.EvtMsg, 512)
}

func (evt *Event) GetMessageSource() (src pkg.DESMessageSource) {
	src.Time = evt.EvtTime
	src.Addr = evt.EvtAddr
	src.UserID = evt.EvtUserID
	src.App = evt.EvtApp
	return
}

/* EVENT - VALIDATE CMD REQUEST FROM USER */
func (evt *Event) CMDValidate(device *Device, uid string) (err error) {

	src := evt.GetMessageSource()
	dev_src := device.ReferenceSRC()
	if err = src.ValidateSRC_CMD(dev_src, uid, evt); err != nil {
		return
	}

	/* USERS CAN NOT SEND RESPONSE CODES TO THE DEVICE */
	switch evt.EvtCode {
	case OP_CODE_DES_REGISTERED:
	case OP_CODE_JOB_ENDED:
	case OP_CODE_JOB_STARTED:
	case OP_CODE_GPS_ACQ:
		_, err = pkg.LogDESError(uid, pkg.ERR_INVALID_SRC_OP_CODE_CMD, evt)
		return
	}

	return
}

/*
EVENT - VALIDATE MQTT SIG FROM DEVICE
*/
func (evt *Event) SIGValidate(device *Device) (err error) {

	src := evt.GetMessageSource()
	dev_src := device.ReferenceSRC()
	if err = src.ValidateSRC_SIG(dev_src, evt); err != nil {
		return
	}

	if evt.EvtCode > MAX_OP_CODE {
		if (evt.EvtCode <= MAX_STATUS_CODE) && evt.EvtUserID != dev_src.UserID {
			pkg.LogDESError(dev_src.UserID, pkg.ERR_INVALID_SRC_SIG, evt)
			evt.EvtUserID = dev_src.UserID

		} else if evt.EvtCode > MAX_STATUS_CODE && evt.EvtUserID == dev_src.UserID {
			pkg.LogDESError(dev_src.UserID, pkg.ERR_INVALID_SRC_SIG, evt)
		}
	}

	evt.Validate()

	return
}

type EventTyp struct {
	// EvtTypID   int64  `gorm:"unique; primaryKey" json:"-"` // POSTGRESS
	EvtTypID   int64  `gorm:"autoIncrement" json:"-"` // SQLITE
	EvtTypCode int32  `gorm:"unique" json:"evt_typ_code"`
	EvtTypName string `json:"evt_typ_name"`
	EvtTypDesc string `json:"evt_typ_desc"`
}

func WriteETYP(etyp EventTyp, jdbc *pkg.JobDBClient) (err error) {

	/* WHEN Write IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING
	WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	if jdbc.RWM == nil {
		jdbc.RWM = &sync.RWMutex{}
	}
	jdbc.RWM.Lock()
	res := jdbc.Create(&etyp)
	jdbc.RWM.Unlock()

	return res.Error
}

/* TODO: TEST MAP IMPLEMENTATION */
// type EventTypesMap map[int32]EventTyp
// var EVENT_TYPE_MAP = make(EventTypesMap)
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
	{EvtTypCode: NOTE_OPERATOR_COMMENT, EvtTypName: "OPERATOR COMMENT"},
	{EvtTypCode: NOTE_REPORT_COMMENT, EvtTypName: "REPORT COMMENT"},
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

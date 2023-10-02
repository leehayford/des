package c001v001

import (
	"github.com/leehayford/des/pkg"
)

/*
EVENT - AS WRITTEN TO JOB DATABASE
*/
type Event struct {
	
	EvtTime   int64  `gorm:"not null" json:"evt_time"`
	EvtAddr   string `json:"evt_addr"`
	EvtUserID string `gorm:"not null; varchar(36)" json:"evt_user_id"`
	EvtApp    string `gorm:"not null; varchar(36)" json:"evt_app"`

	EvtCode  int32    `json:"evt_code"`
	EvtTitle string   `gorm:"varchar(36)" json:"evt_title"`
	EvtMsg   string   `json:"evt_msg"`
	EvtType  EventTyp `gorm:"foreignKey:EvtCode; references:evt_typ_code" json:"-"`
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
	evt.EvtTime =  job.DESJobRegTime
	evt.EvtAddr = job.DESJobRegAddr
	evt.EvtUserID = job.DESJobRegUserID
	evt.EvtApp = job.DESJobRegApp
	evt.EvtCode = STATUS_DES_REG_REQ
	evt.EvtTitle = "A Device is Born"
	evt.EvtMsg = `Congratulations, it's a class 001, version 001 device! This text is here to take up space. Normal people would use the function that shits out latin but I don't; partly because I don't remember what it is and partly because I don't feel like looking it up.`
}

type EventTyp struct {
	// EvtTypID   int64  `gorm:"unique; primaryKey" json:"evt_typ_id"`
	EvtTypCode int32  `gorm:"unique" json:"evt_typ_code"`
	EvtTypName string `json:"evt_typ_name"`
	EvtTypDesc string `json:"evt_typ_desc"`
}

var EVENT_TYPES = []EventTyp{

	/* DEVICE CONTROL EVENT TYPES: 0 - 999 */
	{EvtTypCode: STATUS_DES_REG_REQ, EvtTypName: "DEVICE REGISTRATION REQUESTED"},
	{EvtTypCode: STATUS_DES_REGISTERED, EvtTypName: "DEVICE REGISTERED"},
	{EvtTypCode: STATUS_JOB_ENDED, EvtTypName: "JOB ENDED"},
	{EvtTypCode: STATUS_JOB_START_REQ, EvtTypName: "START JOB REQUESTED"},
	{EvtTypCode: STATUS_JOB_STARTED, EvtTypName: "JOB STARTED"},
	{EvtTypCode: STATUS_JOB_END_REQ, EvtTypName: "END JOB REQUESTED"},

// 	/*OPERATIONAL ALARM EVENT TYPES 1000 -1999 */
// 	{EvtTypCode: 1000, EvtTypName: "ALARM HIGH BATTERY CURRENT"},
// 	{EvtTypCode: 1001, EvtTypName: "ALARM LOW BATTERY VOLTAGE"},
// 	{EvtTypCode: 1002, EvtTypName: "ALARM HIGH MOTOR CURRENT"},
// 	{EvtTypCode: 1003, EvtTypName: "ALARM HIGH PRESSURE"},
// 	{EvtTypCode: 1004, EvtTypName: "ALARM HIGH FLOW"},

// 	/* OPERATIONAL STATUS EVENT TYPES 2000 - 2999 */
// 	{EvtTypCode: 2000, EvtTypName: "CONFIGURATION CHANGED"},
// 	{EvtTypCode: 2001, EvtTypName: "SHUT-IN PRESSURE STABILIZED"},
// 	{EvtTypCode: 2002, EvtTypName: "MODE VENT"},
// 	{EvtTypCode: 2003, EvtTypName: "MODE BUILD"},
// 	{EvtTypCode: 2004, EvtTypName: "MODE HIGH FLOW"},
// 	{EvtTypCode: 2005, EvtTypName: "MODE LOW FLOW"},

}



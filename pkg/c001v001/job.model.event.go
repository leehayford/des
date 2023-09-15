package c001v001

import (
	"github.com/leehayford/des/pkg"
)

/*
EVENT - AS WRITTEN TO JOB DATABASE
*/
type Event struct {
	// EvtID int64 `gorm:"unique; primaryKey" json:"evt_id"`

	EvtTime   int64  `gorm:"not null" json:"evt_time"`
	EvtAddr   string `json:"evt_addr"`
	EvtUserID string `gorm:"not null; varchar(36)" json:"evt_user_id"`
	EvtApp    string `gorm:"not null; varchar(36)" json:"evt_app"`

	EvtCode  int32    `json:"evt_code"`
	EvtTitle string   `gorm:"varchar(36)" json:"evt_title"`
	EvtMsg   string   `json:"evt_msg"`
	EvtType  EventTyp `gorm:"foreignKey:EvtCode; references:evt_typ_code" json:"-"`
}

// /*
// EVENT - MQTT MESSAGE STRUCTURE
// */
// type MQTT_Event struct {
// 	EvtTime   int64  `json:"evt_time"`
// 	EvtAddr   string `json:"evt_addr"`
// 	EvtUserID string `json:"evt_user_id"`
// 	EvtApp    string `json:"evt_app"`

// 	EvtCode  int32  `json:"evt_code"`
// 	EvtTitle string `json:"evt_title"`
// 	EvtMsg   string `json:"evt_msg"`
// }

// func (evt *Event) FilterEvtRecord() MQTT_Event {
// 	return MQTT_Event{
// 		EvtTime:   evt.EvtTime,
// 		EvtAddr:   evt.EvtAddr,
// 		EvtUserID: evt.EvtUserID,
// 		EvtApp:    evt.EvtApp,

// 		EvtCode:  evt.EvtCode,
// 		EvtTitle: evt.EvtTitle,
// 		EvtMsg:   evt.EvtMsg,
// 	}
// }

/*
EVENT - AS STORED IN DEVICE FLASH
*/
func (evt *Event) EventToBytes() (out []byte) {

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

type EventTyp struct {
	EvtTypID   int64  `gorm:"unique; primaryKey" json:"evt_typ_id"`
	EvtTypCode int32  `gorm:"unique" json:"evt_typ_code"`
	EvtTypName string `json:"evt_typ_name"`
	EvtTypDesc string `json:"evt_typ_desc"`
}

var EVENT_TYPES = []EventTyp{

	/*ADMIN EVENT TYPES*/
	{EvtTypCode: 0, EvtTypName: "DEVICE REGISTRATION"},

	/*OPERATIONAL EVENT TYPES*/
	{EvtTypCode: 1, EvtTypName: "JOB END"},

	{EvtTypCode: 2, EvtTypName: "JOB START"},

	{EvtTypCode: 3, EvtTypName: "CONFIGURATION CHANGED"},

	{EvtTypCode: 4, EvtTypName: "SHUT-IN PRESSURE STABILIZED"},

	/*OPERATION ALARM EVENT TYPES*/
	{EvtTypCode: 5, EvtTypName: "ALARM HIGH BATTERY CURRENT"},

	{EvtTypCode: 6, EvtTypName: "ALARM LOW BATTERY VOLTAGE"},

	{EvtTypCode: 7, EvtTypName: "ALARM HIGH MOTOR CURRENT"},

	{EvtTypCode: 8, EvtTypName: "ALARM HIGH PRESSURE"},

	{EvtTypCode: 9, EvtTypName: "ALARM HIGH FLOW"},

	/*OPERATION MODE EVENT TYPES*/
	{EvtTypCode: 10, EvtTypName: "MODE VENT"},

	{EvtTypCode: 11, EvtTypName: "MODE BUILD"},

	{EvtTypCode: 12, EvtTypName: "MODE HIGH FLOW"},

	{EvtTypCode: 13, EvtTypName: "MODE LOW FLOW"},
}



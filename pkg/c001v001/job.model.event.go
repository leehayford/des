package c001v001

/*
EVENT - AS WRITTEN TO JOB DATABASE
*/
type Event struct {
	EvtID int64 `gorm:"unique; primaryKey" json:"evt_id"`

	EvtTime   int64  `gorm:"not null" json:"evt_time"`
	EvtAddr   string `json:"evt_addr"`
	EvtUserID string `gorm:"not null; varchar(36)" json:"evt_user_id"`
	EvtApp    string `gorm:"not null; varchar(36)" json:"evt_app"`

	EvtTitle string   `json:"evt_title"`
	EvtMsg   string   `json:"evt_msg"`
	EvtCode  int64    `json:"evt_code"`
	EvtType  EventTyp `gorm:"foreignKey:EvtCode; references:evt_typ_code" json:"-"`
}

/*
EVENT - MQTT MESSAGE STRUCTURE
*/
type MQTT_Event struct {
	EvtTime   int64  `json:"evt_time"`
	EvtAddr   string `json:"evt_addr"`
	EvtUserID string `json:"evt_user_id"`
	EvtApp    string `json:"evt_app"`

	EvtTitle string `json:"evt_title"`
	EvtMsg   string `json:"evt_msg"`
	EvtCode  int64  `json:"evt_code"`
}

func (evt *Event) FilterEvtRecord() MQTT_Event {
	return MQTT_Event{
		EvtTime:   evt.EvtTime,
		EvtAddr:   evt.EvtAddr,
		EvtUserID: evt.EvtUserID,
		EvtApp:    evt.EvtApp,

		EvtTitle: evt.EvtTitle,
		EvtMsg:   evt.EvtMsg,
		EvtCode:  evt.EvtCode,
	}
}

type EventTyp struct {
	EvtTypID   int64  `gorm:"unique; primaryKey" json:"evt_typ_id"`
	EvtTypCode int64  `gorm:"unique" json:"evt_typ_code"`
	EvtTypName string `json:"evt_typ_name"`
	EvtTypDesc string `json:"evt_typ_desc"`
}

var EVENT_TYPES = []EventTyp{

	/*ADMIN EVENT TYPES*/
	{EvtTypCode: 0, EvtTypName: "DEVICE REGISTRATION"},

	/*OPERATIONAL EVENT TYPES*/
	{EvtTypCode: 1, EvtTypName: "JOB STARTED"},

	{EvtTypCode: 2, EvtTypName: "JOB ENDED"},

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

// /*ADMIN EVENT TYPES*/
// var EVT_TYP_REGISTER_DEVICE EventTyp = EventTyp{EvtTypCode: 0, EvtTypName: "DEVICE REGISTRATION"}

// /*OPERATIONAL EVENT TYPES*/
// var EVT_TYP_JOB_START EventTyp = EventTyp{EvtTypCode: 1, EvtTypName: "JOB STARTED"}

// var EVT_TYP_JOB_END EventTyp = EventTyp{EvtTypCode: 2, EvtTypName: "JOB ENDED"}

// var EVT_TYP_JOB_CONFIG EventTyp = EventTyp{EvtTypCode: 3, EvtTypName: "CONFIGURATION CHANGED"}

// var EVT_TYP_JOB_SSP EventTyp = EventTyp{EvtTypCode: 4, EvtTypName: "SHUT-IN PRESSURE STABILIZED"}

// /*OPERATION ALARM EVENT TYPES*/
// var EVT_TYP_ALARM_HI_BAT_AMP EventTyp = EventTyp{EvtTypCode: 5, EvtTypName: "ALARM HIGH BATTERY CURRENT"}

// var EVT_TYP_ALARM_LO_BAT_VOLT EventTyp = EventTyp{EvtTypCode: 6, EvtTypName: "ALARM LOW BATTERY VOLTAGE"}

// var EVT_TYP_ALARM_HI_MOT_AMP EventTyp = EventTyp{EvtTypCode: 7, EvtTypName: "ALARM HIGH MOTOR CURRENT"}

// var EVT_TYP_ALARM_HI_PRESS EventTyp = EventTyp{EvtTypCode: 8, EvtTypName: "ALARM HIGH PRESSURE"}

// var EVT_TYP_ALARM_HI_FLOW EventTyp = EventTyp{EvtTypCode: 9, EvtTypName: "ALARM HIGH FLOW"}

// /*OPERATION MODE EVENT TYPES*/
// var EVT_TYP_MODE_VENT EventTyp = EventTyp{EvtTypCode: 10, EvtTypName: "MODE VENT"}

// var EVT_TYP_MODE_BUILD EventTyp = EventTyp{EvtTypCode: 11, EvtTypName: "MODE BUILD"}

// var EVT_TYP_MODE_HI_FLOW EventTyp = EventTyp{EvtTypCode: 12, EvtTypName: "MODE HIGH FLOW"}

// var EVT_TYP_MODE_LO_FLOW EventTyp = EventTyp{EvtTypCode: 13, EvtTypName: "MODE LOW FLOW"}

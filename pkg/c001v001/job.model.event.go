package c001v001

/*
EVENT - AS WRITTEN TO JOB DATABASE
*/
type Evt struct {
	EvtID int64 `gorm:"unique; primaryKey" json:"evt_id"`

	EvtTime   int64  `gorm:"not null" json:"evt_time"`
	EvtAddr   string `json:"evt_addr"`
	EvtUserID string `gorm:"not null; varchar(36)" json:"evt_user_id"`
	EvtApp    string `gorm:"not null; varchar(36)" json:"evt_app"`

	EvtTitle  string `json:"evt_title"`
	EvtMsg    string `json:"evt_msg"`
	EvtCode   int64  `json:"evt_code"`
	EvtType EvtTyp `gorm:"foreignKey:EvtCode; references:evt_typ_code" json:"-"`
}

/*
EVENT - MQTT MESSAGE STRUCTURE
*/
type MQTT_Evt struct {
	EvtTime   int64  `json:"evt_time"`
	EvtAddr   string `json:"evt_addr"`
	EvtUserID string `json:"evt_user_id"`
	EvtApp    string `json:"evt_app"`

	EvtTitle string `json:"evt_title"`
	EvtMsg   string `json:"evt_msg"`
	EvtCode  int64  `json:"evt_code"`
}

func (evt *Evt) FilterEvtRecord() MQTT_Evt {
	return MQTT_Evt{
		EvtTime:   evt.EvtTime,
		EvtAddr:   evt.EvtAddr,
		EvtUserID: evt.EvtUserID,
		EvtApp:    evt.EvtApp,

		EvtTitle: evt.EvtTitle,
		EvtMsg:   evt.EvtMsg,
		EvtCode:  evt.EvtCode,
	}
}

type EvtTyp struct {
	EvtTypID int64  `gorm:"unique; primaryKey" json:"evt_typ_id"`
	EvtTypCode   int64  `gorm:"unique" json:"evt_typ_code"`
	EvtTypName   string `json:"evt_typ_name"`
	EvtTypDesc   string `json:"evt_typ_desc"`
}

/*ADMIN EVENT TYPES*/
var EVT_TYP_REGISTER_DEVICE EvtTyp = EvtTyp{EvtTypCode: 0, 
	EvtTypName: "DEVICE REGISTRATION"}


/*OPERATIONAL EVENT TYPES*/
var EVT_TYP_JOB_START EvtTyp = EvtTyp{EvtTypCode: 1, 
	EvtTypName: "JOB STARTED"}

var EVT_TYP_JOB_END EvtTyp = EvtTyp{EvtTypCode: 2, 
	EvtTypName: "JOB ENDED"}

var EVT_TYP_JOB_CONFIG EvtTyp = EvtTyp{EvtTypCode: 3, 
	EvtTypName: "CONFIGURATION CHANGED"}

var EVT_TYP_JOB_SSP EvtTyp = EvtTyp{EvtTypCode: 4, 
	EvtTypName: "SHUT-IN PRESSURE STABILIZED"}


/*OPERATION ALARM EVENT TYPES*/
var EVT_TYP_ALARM_HI_BAT_AMP EvtTyp = EvtTyp{EvtTypCode: 5, 
	EvtTypName: "ALARM HIGH BATTERY CURRENT"}

var EVT_TYP_ALARM_LO_BAT_VOLT EvtTyp = EvtTyp{EvtTypCode: 6, 
	EvtTypName: "ALARM LOW BATTERY VOLTAGE"}

var EVT_TYP_ALARM_HI_MOT_AMP EvtTyp = EvtTyp{EvtTypCode: 7, 
	EvtTypName: "ALARM HIGH MOTOR CURRENT"}

var EVT_TYP_ALARM_HI_PRESS EvtTyp = EvtTyp{EvtTypCode: 8, 
	EvtTypName: "ALARM HIGH PRESSURE"}

var EVT_TYP_ALARM_HI_FLOW EvtTyp = EvtTyp{EvtTypCode: 9, 
	EvtTypName: "ALARM HIGH FLOW"}
	

/*OPERATION MODE EVENT TYPES*/
var EVT_TYP_MODE_VENT EvtTyp = EvtTyp{EvtTypCode: 10, 
	EvtTypName: "MODE VENT"}

var EVT_TYP_MODE_BUILD EvtTyp = EvtTyp{EvtTypCode: 11, 
	EvtTypName: "MODE BUILD"}

var EVT_TYP_MODE_HI_FLOW EvtTyp = EvtTyp{EvtTypCode: 12, 
	EvtTypName: "MODE HIGH FLOW"}

var EVT_TYP_MODE_LO_FLOW EvtTyp = EvtTyp{EvtTypCode: 13, 
	EvtTypName: "MODE LOW FLOW"}



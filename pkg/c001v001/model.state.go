package c001v001

import (
	"fmt"

	"github.com/leehayford/des/pkg"
)

/*
HARDWARE IDs AS WRITTEN TO JOB DATABASE
*/
type State struct {
	StaID int64 `gorm:"unique; primaryKey" json:"-"`	

	StaTime   int64  `gorm:"not null" json:"sta_time"`
	StaAddr   string `json:"sta_addr"`
	StaUserID string `gorm:"not null; varchar(36)" json:"sta_user_id"`
	StaApp    string `gorm:"not null; varchar(36)" json:"sta_app"`

	/*DEVICE*/
	StaSerial  string `gorm:"not null; varchar(10)" json:"sta_serial"`
	StaVersion string `gorm:"not null; varchar(3)" json:"sta_version"`
	StaClass   string `gorm:"not null; varchar(3)" json:"sta_class"`

	/* FW VERSIONS */
	StaLogFw  string `gorm:"not null; varchar(10)" json:"sta_log_fw"`
	StaModFw  string `gorm:"not null; varchar(10)" json:"sta_mod_fw"`
	
	/* LOGGING STATE */
	StaLogging int32 `json:"sta_logging"`
	StaJobName string `gorm:"not null; varchar(24)" json:"sta_job_name"`

	/* CHIP UID (STMicro) */
	StaStmUID1 int32 `json:"sta_stm_uid1"`
	StaStmUID2 int32 `json:"sta_stm_uid2"`
	StaStmUID3 int32 `json:"sta_stm_uid3"`
}
func WriteSTA(sta State, dbc *pkg.DBClient) (err error) {

	/* WHEN Write IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING 
		WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	dbc.WG.Add(1)
	sta.StaID = 0
	res := dbc.Create(&sta) 
	dbc.WG.Done()

	return res.Error
}



/*
STATE AS STORED IN DEVICE FLASH ( HEX )
*/
func (sta State) StateToBytes() (out []byte) {

	out = append(out, pkg.Int64ToBytes(sta.StaTime)...)
	out = append(out, pkg.StringToNBytes(sta.StaAddr, 36)...)
	out = append(out, pkg.StringToNBytes(sta.StaUserID, 36)...)
	out = append(out, pkg.StringToNBytes(sta.StaApp, 36)...)

	out = append(out, pkg.StringToNBytes(sta.StaSerial, 10)...)
	out = append(out, pkg.StringToNBytes(sta.StaVersion, 3)...)
	out = append(out, pkg.StringToNBytes(sta.StaClass, 3)...)

	out = append(out, pkg.StringToNBytes(sta.StaLogFw, 10)...)
	out = append(out, pkg.StringToNBytes(sta.StaModFw, 10)...)
	
	out = append(out, pkg.Int32ToBytes(sta.StaLogging)...)
	out = append(out, pkg.StringToNBytes(sta.StaJobName, 24)...)
	
	out = append(out, pkg.Int32ToBytes(sta.StaStmUID1)...)
	out = append(out, pkg.Int32ToBytes(sta.StaStmUID2)...)
	out = append(out, pkg.Int32ToBytes(sta.StaStmUID3)...)

	return
}
func (sta State) StateFromBytes(b []byte) {

	sta = State {

		StaTime:   pkg.BytesToInt64_L(b[0:8]),
		StaAddr:   pkg.StrBytesToString(b[8:44]),
		StaUserID: pkg.StrBytesToString(b[44:80]),
		StaApp:    pkg.StrBytesToString(b[80:116]),

		StaSerial: pkg.StrBytesToString(b[116:126]),
		StaVersion: pkg.StrBytesToString(b[126:129]),
		StaClass: pkg.StrBytesToString(b[129:132]),

		StaLogFw: pkg.StrBytesToString(b[132:142]),
		StaModFw: pkg.StrBytesToString(b[142:152]),
		
		StaLogging: pkg.BytesToInt32_L(b[152:156]),
		StaJobName:  pkg.StrBytesToString(b[156:180]),
		
		StaStmUID1: pkg.BytesToInt32_L(b[180:184]),
		StaStmUID2: pkg.BytesToInt32_L(b[184:188]),
		StaStmUID3: pkg.BytesToInt32_L(b[188:192]),
	}
	//  pkg.Json("(demo *DemoDeviceClient)StateFromBytes() -> sta", sta)
}

/*
STATE - DEFAULT VALUES
*/
func (sta *State) DefaultSettings_State(reg pkg.DESRegistration) {
	sta.StaTime = reg.DESJobRegTime
	sta.StaAddr =  reg.DESJobRegAddr
	sta.StaUserID = reg.DESJobRegUserID
	sta.StaApp = reg.DESJobRegApp

	sta.StaSerial =  reg.DESDevSerial
	sta.StaVersion = DEVICE_VERSION
	sta.StaClass =  DEVICE_CLASS

	sta.StaLogFw = "00.000.000"
	sta.StaModFw = "00.000.000"

	sta.StaLogging = 0
	sta.StaJobName = fmt.Sprintf("%s_CMDARCHIVE", sta.StaSerial)

	sta.StaStmUID1 = 0
	sta.StaStmUID2 = 0
	sta.StaStmUID3 = 0
	
}

/*
STATE - VALIDATE FIELDS
*/
func (sta *State) Validate() {
	/* TODO: SET ACCEPTABLE LIMITS FOR THE REST OF THE CONFIG SETTINGS */

	sta.StaAddr = pkg.ValidateStringLength(sta.StaAddr, 36)
	sta.StaUserID = pkg.ValidateStringLength(sta.StaUserID, 36)
	sta.StaApp = pkg.ValidateStringLength(sta.StaApp, 36)
	
	sta.StaSerial = pkg.ValidateStringLength(sta.StaSerial, 10)
	sta.StaVersion = pkg.ValidateStringLength(sta.StaVersion, 3)
	sta.StaClass = pkg.ValidateStringLength(sta.StaClass, 3)
	
	sta.StaJobName = pkg.ValidateStringLength(sta.StaJobName, 24)

}
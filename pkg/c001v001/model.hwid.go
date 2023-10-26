package c001v001

import (
	"github.com/leehayford/des/pkg"
)

/*
HARDWARE IDs AS WRITTEN TO JOB DATABASE
*/
type HwID struct {
	HwID int64 `gorm:"unique; primaryKey" json:"-"`	

	HwTime   int64  `gorm:"not null" json:"hw_time"`
	HwAddr   string `json:"hw_addr"`
	HwUserID string `gorm:"not null; varchar(36)" json:"hw_user_id"`
	HwApp    string `gorm:"not null; varchar(36)" json:"hw_app"`

	/*DEVICE*/
	HwSerial  string `gorm:"not null; varchar(10)" json:"adm_serial"`
	HwVersion string `gorm:"not null; varchar(3)" json:"adm_version"`
	HwClass   string `gorm:"not null; varchar(3)" json:"adm_class"`

	HwLogFw  string `gorm:"not null; varchar(10)" json:"hw_log_fw"`
	HwModFw  string `gorm:"not null; varchar(10)" json:"hw_mod_fw"`
}
func WriteHW(hw HwID, dbc *pkg.DBClient) (err error) {

	/* WHEN Write IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING 
		WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	dbc.WG.Add(1)
	hw.HwID = 0
	res := dbc.Create(&hw) 
	dbc.WG.Done()

	return res.Error
}



/*
HARDWARE IDs AS STORED IN DEVICE FLASH ( HEX )
*/
func (hw HwID) HwIDToBytes() (out []byte) {

	out = append(out, pkg.Int64ToBytes(hw.HwTime)...)
	out = append(out, pkg.StringToNBytes(hw.HwAddr, 36)...)
	out = append(out, pkg.StringToNBytes(hw.HwUserID, 36)...)
	out = append(out, pkg.StringToNBytes(hw.HwApp, 36)...)

	out = append(out, pkg.StringToNBytes(hw.HwSerial, 10)...)
	out = append(out, pkg.StringToNBytes(hw.HwVersion, 3)...)
	out = append(out, pkg.StringToNBytes(hw.HwClass, 3)...)

	out = append(out, pkg.StringToNBytes(hw.HwLogFw, 10)...)
	out = append(out, pkg.StringToNBytes(hw.HwModFw, 10)...)

	return
}
func (hw HwID) HwIDFromBytes(b []byte) {

	hw = HwID {

		HwTime:   pkg.BytesToInt64_L(b[0:8]),
		HwAddr:   pkg.StrBytesToString(b[8:44]),
		HwUserID: pkg.StrBytesToString(b[44:80]),
		HwApp:    pkg.StrBytesToString(b[80:116]),

		HwSerial: pkg.StrBytesToString(b[116:126]),
		HwVersion: pkg.StrBytesToString(b[126:129]),
		HwClass: pkg.StrBytesToString(b[129:132]),

		HwLogFw: pkg.StrBytesToString(b[132:142]),
		HwModFw: pkg.StrBytesToString(b[142:152]),
	}
	//  pkg.Json("(demo *DemoDeviceClient)HwIDFromBytes() -> hw", hw)
}

/*
HARDWARE IDs - DEFAULT VALUES
*/
func (hw *HwID) DefaultSettings_HwID(reg pkg.DESRegistration) {
	hw.HwTime = reg.DESJobRegTime
	hw.HwAddr =  reg.DESJobRegAddr
	hw.HwUserID = reg.DESJobRegUserID
	hw.HwApp = reg.DESJobRegApp

	hw.HwSerial =  reg.DESDevSerial
	hw.HwVersion = DEVICE_VERSION
	hw.HwClass =  DEVICE_CLASS

	hw.HwLogFw = "00.000.000"
	hw.HwModFw = "00.000.000"
}

/*
HARDWARE IDs - VALIDATE FIELDS
*/
func (hw *HwID) Validate() {
	/* TODO: SET ACCEPTABLE LIMITS FOR THE REST OF THE CONFIG SETTINGS */

	hw.HwAddr = pkg.ValidateStringLength(hw.HwAddr, 36)
	hw.HwUserID = pkg.ValidateStringLength(hw.HwUserID, 36)
	hw.HwApp = pkg.ValidateStringLength(hw.HwApp, 36)
	
	hw.HwSerial = pkg.ValidateStringLength(hw.HwSerial, 10)
	hw.HwVersion = pkg.ValidateStringLength(hw.HwVersion, 3)
	hw.HwClass = pkg.ValidateStringLength(hw.HwClass, 3)

}
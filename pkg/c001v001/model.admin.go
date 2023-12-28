package c001v001

import (
	"errors"
	"fmt"

	"github.com/leehayford/des/pkg"
)

/*
ADMIN - AS WRITTEN TO JOB DATABASE
*/
type Admin struct {
	AdmID int64 `gorm:"unique; primaryKey" json:"-"`	

	AdmTime   int64  `gorm:"not null" json:"adm_time"`
	AdmAddr   string `gorm:"varchar(36)" json:"adm_addr"`
	AdmUserID string `gorm:"not null; varchar(36)" json:"adm_user_id"`
	AdmApp    string `gorm:"varchar(36)" json:"adm_app"`

	/*BROKER*/
	AdmDefHost string `gorm:"varchar(32)" json:"adm_def_host"`
	AdmDefPort int32  `json:"adm_def_port"`
	AdmOpHost  string `gorm:"varchar(32)" json:"adm_op_host"`
	AdmOpPort  int32  `json:"adm_op_port"`

	/*BATTERY ALARMS*/
	AdmBatHiAmp  float32 `json:"adm_bat_hi_amp"`
	AdmBatLoVolt float32 `json:"adm_bat_lo_volt"`

	/*MOTOR ALARMS*/
	AdmMotHiAmp float32 `json:"adm_mot_hi_amp"`

	AdmPress float32 `json:"adm_press"` // 6991.3 kPa (1014 psia)
	AdmPressMin float32 `json:"adm_press_min"` // 689.5 kPa (100 psia)
	AdmPressMax float32 `json:"adm_press_max"` // 6991.3 kPa (1014 psia)

	// /* POSTURE - NOT IMPLEMENTED */
	// AdmTiltTgt float32 `json:"adm_tilt_tgt"` // 90.0 °
	// AdmTiltMgn float32 `json:"adm_tilt_mgn"` // 3.0 °
	// AdmAzimTgt float32 `json:"adm_azim_tgt"` // 180.0 °
	// AdmAzimMgn float32 `json:"adm_azim_mgn"` // 3.0 °

	/* HIGH FLOW SENSOR ( HFS )*/
	AdmHFSFlow     float32 `json:"adm_hfs_flow"`      // 200.0 L/min
	AdmHFSFlowMin  float32 `json:"adm_hfs_flow_min"`  // 150.0 L/min
	AdmHFSFlowMax  float32 `json:"adm_hfs_flow_max"`  //  250.0 L/min
	AdmHFSPress    float32 `json:"adm_hfs_press"`     // 1103.1 kPa (160 psia)
	AdmHFSPressMin float32 `json:"adm_hfs_press_min"` // 158.6 kPa (23 psia)
	AdmHFSPressMax float32 `json:"adm_hfs_press_max"` // 1378.9 kPa (200 psia)
	AdmHFSDiff     float32 `json:"adm_hfs_diff"`      // 448.2 kPa (65 psia)
	AdmHFSDiffMin  float32 `json:"adm_hfs_diff_min"`  // 68.9 kPa (10 psia)
	AdmHFSDiffMax  float32 `json:"adm_hfs_diff_max"`  // 517.1 kPa (75 psia)

	/* LOW FLOW SENSOR ( LFS )*/
	AdmLFSFlow     float32 `json:"adm_lfs_flow"`      // 1.85 L/min
	AdmLFSFlowMin  float32 `json:"adm_lfs_flow_min"`  // 0.5 L/min
	AdmLFSFlowMax  float32 `json:"adm_lfs_flow_max"`  // 2.0 L/min
	AdmLFSPress    float32 `json:"adm_lfs_press"`     // 413.7 kPa (60 psia)
	AdmLFSPressMin float32 `json:"adm_lfs_press_min"` // 137.9 kPa (20 psia)
	AdmLFSPressMax float32 `json:"adm_lfs_press_max"` // 551.5 kPa (80 psia)
	AdmLFSDiff     float32 `json:"adm_lfs_diff"`      // 62.0 kPa (9 psia)
	AdmLFSDiffMin  float32 `json:"adm_lfs_diff_min"`  // 13.8 kPa (2 psia)
	AdmLFSDiffMax  float32 `json:"adm_lfs_diff_max"`  // 68.9 kPa (10 psia)
}
func WriteADM(adm Admin, dbc *pkg.DBClient) (err error) {

	/* WHEN Write IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING 
		WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	dbc.WG.Add(1)
	adm.AdmID = 0
	res := dbc.Create(&adm) 
	dbc.WG.Done()

	return res.Error
}


/*
ADMIN - AS STORED IN DEVICE FLASH
*/
func (adm Admin) AdminToBytes() (out []byte) {

	out = append(out, pkg.Int64ToBytes(adm.AdmTime)...)
	out = append(out, pkg.StringToNBytes(adm.AdmAddr, 36)...)
	out = append(out, pkg.StringToNBytes(adm.AdmUserID, 36)...)
	out = append(out, pkg.StringToNBytes(adm.AdmApp, 36)...)

	out = append(out, pkg.StringToNBytes(adm.AdmDefHost, 32)...)
	out = append(out, pkg.Int32ToBytes(adm.AdmDefPort)...)
	out = append(out, pkg.StringToNBytes(adm.AdmOpHost, 32)...)
	out = append(out, pkg.Int32ToBytes(adm.AdmOpPort)...)

	out = append(out, pkg.Float32ToBytes(adm.AdmBatHiAmp)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmBatLoVolt)...)

	out = append(out, pkg.Float32ToBytes(adm.AdmMotHiAmp)...)
	
	out = append(out, pkg.Float32ToBytes(adm.AdmPress)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmPressMin)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmPressMax)...)

	out = append(out, pkg.Float32ToBytes(adm.AdmHFSFlow)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmHFSFlowMin)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmHFSFlowMax)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmHFSPress)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmHFSPressMin)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmHFSPressMax)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmHFSDiff)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmHFSDiffMin)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmHFSDiffMax)...)

	out = append(out, pkg.Float32ToBytes(adm.AdmLFSFlow)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmLFSFlowMin)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmLFSFlowMax)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmLFSPress)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmLFSPressMin)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmLFSPressMax)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmLFSDiff)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmLFSDiffMin)...)
	out = append(out, pkg.Float32ToBytes(adm.AdmLFSDiffMax)...)

	return
}
func (adm *Admin) AdminFromBytes(b []byte) {

	adm = &Admin{

		AdmTime:   pkg.BytesToInt64_L(b[0:8]),
		AdmAddr:   pkg.StrBytesToString(b[8:44]),
		AdmUserID: pkg.StrBytesToString(b[44:80]),
		AdmApp:    pkg.StrBytesToString(b[80:116]),

		AdmDefHost: pkg.StrBytesToString(b[116:148]),
		AdmDefPort: pkg.BytesToInt32_L(b[148:152]),
		AdmOpHost:  pkg.StrBytesToString(b[152:184]),
		AdmOpPort:  pkg.BytesToInt32_L(b[184:188]),

		AdmBatHiAmp:  pkg.BytesToFloat32_L(b[188:192]),
		AdmBatLoVolt: pkg.BytesToFloat32_L(b[192:196]),

		AdmMotHiAmp: pkg.BytesToFloat32_L(b[196:200]),

		AdmPress: pkg.BytesToFloat32_L(b[196:200]),
		AdmPressMin: pkg.BytesToFloat32_L(b[200:204]),
		AdmPressMax: pkg.BytesToFloat32_L(b[204:208]),

		AdmHFSFlow:     pkg.BytesToFloat32_L(b[208:212]),
		AdmHFSFlowMin:  pkg.BytesToFloat32_L(b[212:216]),
		AdmHFSFlowMax:  pkg.BytesToFloat32_L(b[216:220]),
		AdmHFSPress:    pkg.BytesToFloat32_L(b[220:224]),
		AdmHFSPressMin: pkg.BytesToFloat32_L(b[224:228]),
		AdmHFSPressMax: pkg.BytesToFloat32_L(b[228:232]),
		AdmHFSDiff:     pkg.BytesToFloat32_L(b[232:236]),
		AdmHFSDiffMin:  pkg.BytesToFloat32_L(b[236:240]),
		AdmHFSDiffMax:  pkg.BytesToFloat32_L(b[244:248]),

		AdmLFSFlow:     pkg.BytesToFloat32_L(b[248:252]),
		AdmLFSFlowMin:  pkg.BytesToFloat32_L(b[252:256]),
		AdmLFSFlowMax:  pkg.BytesToFloat32_L(b[256:260]),
		AdmLFSPress:    pkg.BytesToFloat32_L(b[260:264]),
		AdmLFSPressMin: pkg.BytesToFloat32_L(b[264:268]),
		AdmLFSPressMax: pkg.BytesToFloat32_L(b[268:272]),
		AdmLFSDiff:     pkg.BytesToFloat32_L(b[272:276]),
		AdmLFSDiffMin:  pkg.BytesToFloat32_L(b[276:280]),
		AdmLFSDiffMax:  pkg.BytesToFloat32_L(b[280:284]),
	}
	//  pkg.Json("(demo *DemoDeviceClient)MakeAdmFromBytes() -> adm", adm)
	return
}

/*
ADMIN - DEFAULT VALUES
*/
func (adm *Admin) DefaultSettings_Admin(reg pkg.DESRegistration) {

		adm.AdmTime = reg.DESJobRegTime
		adm.AdmAddr = reg.DESJobRegAddr
		adm.AdmUserID = reg.DESJobRegUserID
		adm.AdmApp = reg.DESJobRegApp

		/* BROKER */
		adm.AdmDefHost = pkg.MQTT_HOST
		adm.AdmDefPort = pkg.MQTT_PORT
		adm.AdmOpHost = pkg.MQTT_HOST
		adm.AdmOpPort = pkg.MQTT_PORT

		/* BATTERY */
		adm.AdmBatHiAmp = 2.5  // Amps
		adm.AdmBatLoVolt = 10.5 // Volts

		/* MOTOR */
		adm.AdmMotHiAmp =1.9 // Volts

        adm.AdmPress = 6894.8 // kPa (1014 psia)
        adm.AdmPressMin = 689.5 // kPa (100 psia)
        adm.AdmPressMax = 6894.8 // kPa (1014 psia)

		// /* POSTURE - NOT IMPLEMENTED */
		// TiltTarget float32 `json:"tilt_target"` // 90.0 °
		// TiltMargin float32 `json:"tilt_margin"` // 3.0 °
		// AzimTarget float32 `json:"azim_target"` // 180.0 °
		// AzimMargin float32 `json:"azim_margin"` // 3.0 °

		/* HIGH FLOW SENSOR ( HFS )*/
		adm.AdmHFSFlow = 200.0 // 200.0 L/min
		adm.AdmHFSFlowMin =  150.0 // 150.0 L/min
		adm.AdmHFSFlowMax =  250.0 //  250.0 L/min
		adm.AdmHFSPress =    1103.1 // kPa (160 psia)
		adm.AdmHFSPressMin = 158.6 // kPa (23 psia)
		adm.AdmHFSPressMax = 1378.9 // kPa (200 psia)
		adm.AdmHFSDiff =    448.2 // kPa (65 psia)
		adm.AdmHFSDiffMin =  68.9 // kPa (10 psia)
		adm.AdmHFSDiffMax =  517.1 // kPa (75 psia)

		/* LOW FLOW SENSOR ( LFS )*/
		adm.AdmLFSFlow =    1.85 // 1.85 L/min
		adm.AdmLFSFlowMin =  0.5  // 0.5 L/min
		adm.AdmLFSFlowMax =  2.0  // 2.0 L/min
		adm.AdmLFSPress =    413.7 // kPa (60 psia)
		adm.AdmLFSPressMin = 137.9 // kPa (20 psia)
		adm.AdmLFSPressMax = 551.5 // kPa (80 psia)
		adm.AdmLFSDiff =     62.0 // kPa (9 psia)
		adm.AdmLFSDiffMin =  13.8 // kPa (2 psia)
		adm.AdmLFSDiffMax =  68.9 // kPa (10 psia)
}

/*
ADMIN - VALIDATE FIELDS
*/
func (adm *Admin) Validate() {
	/* TODO: SET ACCEPTABLE LIMITS FOR THE REST OF THE CONFIG SETTINGS */

	adm.AdmAddr = pkg.ValidateStringLength(adm.AdmAddr, 36)
	adm.AdmUserID = pkg.ValidateStringLength(adm.AdmUserID, 36)
	adm.AdmApp = pkg.ValidateStringLength(adm.AdmApp, 36)

	adm.AdmDefHost = pkg.ValidateStringLength(adm.AdmDefHost, 32)
	adm.AdmOpHost = pkg.ValidateStringLength(adm.AdmOpHost, 32)

}

/* 
ADMIN - VALIDATE MQTT SIG FROM DEVICE
*/
func (adm *Admin) SIGValidate(device *Device) (err error) {
	
	if err = pkg.ValidateUnixMilli(adm.AdmTime); err != nil {
		return fmt.Errorf("Invlid AdmTime: %s", err.Error())
	}
	if adm.AdmAddr != device.DESDevSerial { 
		pkg.LogErr(errors.New("\nInvalid device.ADM.AdmAddr."))
		adm.AdmAddr = device.DESDevSerial 
	}
	// if adm.AdmAddr == device.DESDevSerial && adm.AdmUserID != device.DESU.ID.String() { 
	// 	pkg.LogErr(errors.New("\nInvalid device.DESU: wrong user ID."))
	// 	adm.AdmUserID = device.DESU.ID.String() 
	// }
	adm.Validate()
	
	return
}
package c001v001

import (
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

	// /* POSTURE - NOT IMPLEMENTED */
	// AdmTiltTgt float32 `json:"adm_tilt_tgt"` // 90.0 °
	// AdmTiltMgn float32 `json:"adm_tilt_mgn"` // 3.0 °
	// AdmAzimTgt float32 `json:"adm_azim_tgt"` // 180.0 °
	// AdmAzimMgn float32 `json:"adm_azim_mgn"` // 3.0 °

	/* HIGH FLOW SENSOR ( HFS )*/
	AdmHFSFlow     float32 `json:"adm_hfs_flow"`      // 200.0 L/min
	AdmHFSFlowMin  float32 `json:"adm_hfs_flow_min"`  // 150.0 L/min
	AdmHFSFlowMax  float32 `json:"adm_hfs_flow_max"`  //  250.0 L/min
	AdmHFSPress    float32 `json:"adm_hfs_press"`     // 160.0 psia
	AdmHFSPressMin float32 `json:"adm_hfs_press_min"` //  23.0 psia
	AdmHFSPressMax float32 `json:"adm_hfs_press_max"` //  200.0 psia
	AdmHFSDiff     float32 `json:"adm_hfs_diff"`      //  65.0 psi
	AdmHFSDiffMin  float32 `json:"adm_hfs_diff_min"`  //  10.0 psi
	AdmHFSDiffMax  float32 `json:"adm_hfs_diff_max"`  //  75.0 psi

	/* LOW FLOW SENSOR ( LFS )*/
	AdmLFSFlow     float32 `json:"adm_lfs_flow"`      // 1.85 L/min
	AdmLFSFlowMin  float32 `json:"adm_lfs_flow_min"`  // 0.5 L/min
	AdmLFSFlowMax  float32 `json:"adm_lfs_flow_max"`  // 2.0 L/min
	AdmLFSPress    float32 `json:"adm_lfs_press"`     // 60.0 psia
	AdmLFSPressMin float32 `json:"adm_lfs_press_min"` // 20.0 psia
	AdmLFSPressMax float32 `json:"adm_lfs_press_max"` // 80.0 psia
	AdmLFSDiff     float32 `json:"adm_lfs_diff"`      // 9.0 psi
	AdmLFSDiffMin  float32 `json:"adm_lfs_diff_min"`  // 2.0 psi
	AdmLFSDiffMax  float32 `json:"adm_lfs_diff_max"`  // 10.0 psi
}
func  (adm *Admin) Write(dbc *pkg.DBClient) (err error) {

	/* WHEN Write IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING 
		WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	dbc.WG.Add(1)
	adm.AdmID = 0
	res := dbc.Create(adm) 
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

		AdmHFSFlow:     pkg.BytesToFloat32_L(b[200:204]),
		AdmHFSFlowMin:  pkg.BytesToFloat32_L(b[204:208]),
		AdmHFSFlowMax:  pkg.BytesToFloat32_L(b[208:212]),
		AdmHFSPress:    pkg.BytesToFloat32_L(b[212:216]),
		AdmHFSPressMin: pkg.BytesToFloat32_L(b[216:220]),
		AdmHFSPressMax: pkg.BytesToFloat32_L(b[220:224]),
		AdmHFSDiff:     pkg.BytesToFloat32_L(b[224:228]),
		AdmHFSDiffMin:  pkg.BytesToFloat32_L(b[228:232]),
		AdmHFSDiffMax:  pkg.BytesToFloat32_L(b[232:236]),

		AdmLFSFlow:     pkg.BytesToFloat32_L(b[236:240]),
		AdmLFSFlowMin:  pkg.BytesToFloat32_L(b[240:244]),
		AdmLFSFlowMax:  pkg.BytesToFloat32_L(b[244:248]),
		AdmLFSPress:    pkg.BytesToFloat32_L(b[248:252]),
		AdmLFSPressMin: pkg.BytesToFloat32_L(b[252:256]),
		AdmLFSPressMax: pkg.BytesToFloat32_L(b[256:260]),
		AdmLFSDiff:     pkg.BytesToFloat32_L(b[260:264]),
		AdmLFSDiffMin:  pkg.BytesToFloat32_L(b[264:268]),
		AdmLFSDiffMax:  pkg.BytesToFloat32_L(b[268:272]),
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

		// /* POSTURE - NOT IMPLEMENTED */
		// TiltTarget float32 `json:"tilt_target"` // 90.0 °
		// TiltMargin float32 `json:"tilt_margin"` // 3.0 °
		// AzimTarget float32 `json:"azim_target"` // 180.0 °
		// AzimMargin float32 `json:"azim_margin"` // 3.0 °

		/* HIGH FLOW SENSOR ( HFS )*/
		adm.AdmHFSFlow = 200.0 // 200.0 L/min
		adm.AdmHFSFlowMin =  150.0 // 150.0 L/min
		adm.AdmHFSFlowMax =  250.0 //  250.0 L/min
		adm.AdmHFSPress =    160.0 // 160.0 psia
		adm.AdmHFSPressMin = 23    //  23.0 psia
		adm.AdmHFSPressMax = 200.0 //  200.0 psia
		adm.AdmHFSDiff =    65.0  //  65.0 psi
		adm.AdmHFSDiffMin =  10.0  //  10.0 psi
		adm.AdmHFSDiffMax =  75.0  //  75.0 psi

		/* LOW FLOW SENSOR ( LFS )*/
		adm.AdmLFSFlow =    1.85 // 1.85 L/min
		adm.AdmLFSFlowMin =  0.5  // 0.5 L/min
		adm.AdmLFSFlowMax =  2.0  // 2.0 L/min
		adm.AdmLFSPress =    60.0 // 60.0 psia
		adm.AdmLFSPressMin = 20.0 // 20.0 psia
		adm.AdmLFSPressMax = 800  // 80.0 psia
		adm.AdmLFSDiff =     9.0  // 9.0 psi
		adm.AdmLFSDiffMin =  2.0  // 2.0 psi
		adm.AdmLFSDiffMax =  10.0 // 10.0 psi
}

/*
ADMIN - VALIDATE FIELDS
*/
func (adm *Admin) Validate() {
	/* TODO: SET ACCEPTABLE LIMITS FOR THE REST OF THE CONFIG SETTINGS */

	adm.AdmAddr = pkg.ValidateStringLength(adm.AdmAddr, 36)
	adm.AdmUserID = pkg.ValidateStringLength(adm.AdmUserID, 36)
	adm.AdmApp = pkg.ValidateStringLength(adm.AdmApp, 36)

	adm.AdmDefHost = pkg.ValidateStringLength(adm.AdmApp, 32)
	adm.AdmOpHost = pkg.ValidateStringLength(adm.AdmApp, 32)

}
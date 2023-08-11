package c001v001

/*
ADMIN - AS WRITTEN TO JOB DATABASE
*/
type Admin struct {
	AdmID int64 `gorm:"unique; primaryKey" json:"adm_id"`

	AdmTime   int64  `gorm:"not null" json:"adm_time"`
	AdmAddr   string `json:"adm_addr"`
	AdmUserID string `gorm:"not null; varchar(36)" json:"adm_user_id"`
	AdmApp    string `gorm:"not null; varchar(36)" json:"adm_app"`

	/*BROKER*/
	AdmDefHost string `json:"adm_def_host"`
	AdmDefPort int32  `json:"adm_def_port"`
	AdmOpHost  string `json:"adm_op_host"`
	AdmOpsPort int32  `json:"adm_op_port"`

	/*DEVICE*/
	AdmSerial  string `gorm:"not null; varchar(10)" json:"adm_serial"`
	AdmVersion string `gorm:"not null; varchar(3)" json:"adm_version"`
	AdmClass   string `gorm:"not null; varchar(3)" json:"adm_class"`

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

/*
ADMIN - MQTT MESSAGE STRUCTURE
*/
type MQTT_Admin struct {
	AdmTime   int64  `json:"adm_time"`
	AdmAddr   string `json:"adm_addr"`
	AdmUserID string `json:"adm_user_id"`
	AdmApp    string `json:"adm_app"`

	/*BROKER*/
	AdmDefHost string `json:"adm_def_host"`
	AdmDefPort int32  `json:"adm_def_port"`
	AdmOpHost  string `json:"adm_op_host"`
	AdmOpsPort int32  `json:"adm_op_port"`

	/*DEVICE*/
	AdmSerial  string `json:"adm_serial"`
	AdmVersion string `json:"adm_version"`
	AdmClass   string `json:"adm_class"`

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

func (adm *Admin) FilterAdmRecord() MQTT_Admin {
	return MQTT_Admin{
		AdmTime:   adm.AdmTime,
		AdmAddr:   adm.AdmAddr,
		AdmUserID: adm.AdmUserID,
		AdmApp:    adm.AdmApp,

		AdmDefHost: adm.AdmDefHost,
		AdmDefPort: adm.AdmDefPort,
		AdmOpHost:  adm.AdmOpHost,
		AdmOpsPort: adm.AdmOpsPort,

		AdmSerial:  adm.AdmSerial,
		AdmVersion: adm.AdmVersion,
		AdmClass:   adm.AdmClass,

		AdmBatHiAmp:  adm.AdmBatHiAmp,
		AdmBatLoVolt: adm.AdmBatLoVolt,

		AdmMotHiAmp: adm.AdmMotHiAmp,

		AdmHFSFlow:     adm.AdmHFSFlow,
		AdmHFSFlowMin:  adm.AdmHFSFlowMin,
		AdmHFSFlowMax:  adm.AdmHFSFlowMax,
		AdmHFSPress:    adm.AdmHFSPress,
		AdmHFSPressMin: adm.AdmHFSPressMin,
		AdmHFSPressMax: adm.AdmHFSPressMax,
		AdmHFSDiff:     adm.AdmHFSDiff,
		AdmHFSDiffMin:  adm.AdmHFSDiffMin,
		AdmHFSDiffMax:  adm.AdmHFSDiffMax,

		AdmLFSFlow:     adm.AdmLFSFlow,
		AdmLFSFlowMin:  adm.AdmLFSFlowMin,
		AdmLFSFlowMax:  adm.AdmLFSFlowMax,
		AdmLFSPress:    adm.AdmLFSPress,
		AdmLFSPressMin: adm.AdmLFSPressMin,
		AdmLFSPressMax: adm.AdmLFSPressMax,
		AdmLFSDiff:     adm.AdmLFSDiff,
		AdmLFSDiffMin:  adm.AdmLFSDiffMin,
		AdmLFSDiffMax:  adm.AdmLFSDiffMax,
	}
}

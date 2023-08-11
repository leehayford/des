package c001v001

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/leehayford/des/pkg"
)

func (job *Job) Write(model interface{}) (err error) {

	db := job.JDB()
	db.Connect()
	defer db.Close()
	if res := db.Create(model); res.Error != nil {
		return res.Error
	}
	return db.Close()
}

func (job *Job) WriteMQTT(msg []byte, model interface{}) (err error) {

	if err = json.Unmarshal(msg, model); err != nil {
		return pkg.Trace(err)
	}
	return job.Write(model)
}

/*
USED WHEN NEW C001V001 JOBS ARE GREATED
  - Create a job record in the DES database
  - Create a new Job database for the job data
  - Sets the the Device's active job
*/
func (job *Job) RegisterJob() (err error) {

	/* Create a job record in the DES database */
	if job_res := pkg.DES.DB.Create(&job.DESJob); job_res.Error != nil {
		return job_res.Error
	}

	/* ADMIN DB - CONNECT TO THE ADMIN DATABASE */
	adb := pkg.DBI{ConnStr: pkg.ADMIN_DB_CONNECTION_STRING}
	adb.Connect()
	defer adb.Close()
	existing := adb.CreateDatabase(strings.ToLower(job.DESJobName), false)
	adb.Close()

	if !existing {

		/* CREATE NEW DATABASE */
		db := job.JDB()
		db.Connect()
		defer db.Close()

		if err = db.Migrator().CreateTable(
			&Admin{},
			&Config{},
			&EventTyp{},
			&Event{},
			&JobSample{},
		); err != nil {
			return err
		}

		/* CREATE EVENT TYPES */
		db.Create(&EVT_TYP_REGISTER_DEVICE)

		db.Create(&EVT_TYP_JOB_START)
		db.Create(&EVT_TYP_JOB_END)
		db.Create(&EVT_TYP_JOB_CONFIG)
		db.Create(&EVT_TYP_JOB_SSP)

		db.Create(&EVT_TYP_ALARM_HI_BAT_AMP)
		db.Create(&EVT_TYP_ALARM_LO_BAT_VOLT)
		db.Create(&EVT_TYP_ALARM_HI_MOT_AMP)
		db.Create(&EVT_TYP_ALARM_HI_PRESS)
		db.Create(&EVT_TYP_ALARM_HI_FLOW)

		db.Create(&EVT_TYP_MODE_VENT)
		db.Create(&EVT_TYP_MODE_BUILD)
		db.Create(&EVT_TYP_MODE_HI_FLOW)
		db.Create(&EVT_TYP_MODE_LO_FLOW)

		job.Admins = []Admin{job.RegisterJob_Default_JobAdmin()}
		if adm_res := db.Create(&job.Admins[0]); adm_res.Error != nil {
			fmt.Printf("\n(job *Job) RegisterJob() -> db.Create(&jobAdmins[0]) -> Error:\n%s\n", adm_res.Error.Error())
			return adm_res.Error
		}

		job.Configs = []Config{job.RegisterJob_Default_JobConfig()}
		if cfg_res := db.Create(&job.Configs[0]); cfg_res.Error != nil {
			fmt.Printf("\n(job *Job) RegisterJob() -> db.Create(&job.Configs[0]) -> Error:\n%s\n", cfg_res.Error.Error())
			return cfg_res.Error
		}

		job.Events = []Event{job.RegisterJob_Default_JobEvent()}
		if evt_res := db.Create(&job.Events[0]); evt_res.Error != nil {
			fmt.Printf("\n(job *Job) RegisterJob() -> db.Create(&job.Events[0]) -> Error:\n%s\n", evt_res.Error.Error())
			return evt_res.Error
		}

		job.Samples = []JobSample{}

	}
	return
}
func (job *Job) RegisterJob_Default_JobAdmin() (adm Admin) {
	return Admin{
		AdmTime:   job.DESJobRegTime,
		AdmAddr:   job.DESJobRegAddr,
		AdmUserID: job.DESJobRegUserID,
		AdmApp:    job.DESJobRegApp,

		/* BROKER */
		AdmDefHost: pkg.MQTT_HOST,
		AdmDefPort: pkg.MQTT_PORT,
		AdmOpHost:  pkg.MQTT_HOST,
		AdmOpsPort: pkg.MQTT_PORT,

		/* DEVICE */
		AdmClass:   DEVICE_CLASS,
		AdmVersion: DEVICE_VERSION,
		AdmSerial:  job.DESDevSerial,

		/* BATTERY */
		AdmBatHiAmp:  2.5,  // Amps
		AdmBatLoVolt: 10.5, // Volts

		/* MOTOR */
		AdmMotHiAmp: 1.9, // Volts

		// /* POSTURE - NOT IMPLEMENTED */
		// TiltTarget float32 `json:"tilt_target"` // 90.0 °
		// TiltMargin float32 `json:"tilt_margin"` // 3.0 °
		// AzimTarget float32 `json:"azim_target"` // 180.0 °
		// AzimMargin float32 `json:"azim_margin"` // 3.0 °

		/* HIGH FLOW SENSOR ( HFS )*/
		AdmHFSFlow:     200.0, // 200.0 L/min
		AdmHFSFlowMin:  150.0, // 150.0 L/min
		AdmHFSFlowMax:  250.0, //  250.0 L/min
		AdmHFSPress:    160.0, // 160.0 psia
		AdmHFSPressMin: 23,    //  23.0 psia
		AdmHFSPressMax: 200.0, //  200.0 psia
		AdmHFSDiff:     65.0,  //  65.0 psi
		AdmHFSDiffMin:  10.0,  //  10.0 psi
		AdmHFSDiffMax:  75.0,  //  75.0 psi

		/* LOW FLOW SENSOR ( LFS )*/
		AdmLFSFlow:     1.85, // 1.85 L/min
		AdmLFSFlowMin:  0.5,  // 0.5 L/min
		AdmLFSFlowMax:  2.0,  // 2.0 L/min
		AdmLFSPress:    60.0, // 60.0 psia
		AdmLFSPressMin: 20.0, // 20.0 psia
		AdmLFSPressMax: 800,  // 80.0 psia
		AdmLFSDiff:     9.0,  // 9.0 psi
		AdmLFSDiffMin:  2.0,  // 2.0 psi
		AdmLFSDiffMax:  10.0, // 10.0 psi
	}
}
func (job *Job) RegisterJob_Default_JobConfig() (cfg Config) {
	return Config{
		CfgTime:   job.DESJobRegTime,
		CfgAddr:   job.DESJobRegAddr,
		CfgUserID: job.DESJobRegUserID,
		CfgApp:    job.DESJobRegApp,

		/* JOB */
		CfgJobName:  job.DESJobName,
		CfgJobStart: job.DESJobRegTime,
		CfgJobEnd:   0,
		CfgSCVD:     596.8, // m
		CfgSCVDMult: 10.5,  // kPa / m
		CfgSSPRate:  1.95,  // kPa / hour
		CfgSSPDur:   6.0,   // hour
		CfgHiSCVF:   201.4, //  L/min

		/* VALVE */
		CfgVlvTgt: 2, // vent
		CfgVlvPos: 2, // vent

		/* OP PERIODS*/
		CfgOpLog:    10000, // millisecond
		CfgOpTrans:  60000, // millisecond
		CfgOpSample: 1000,  // millisecond

		/* DIAG PERIODS */
		CfgDiagLog:    100000, // millisecond
		CfgDiagTrans:  600000, // millisecond
		CfgDiagSample: 10000,  // millisecond
	}
}
func (job *Job) RegisterJob_Default_JobEvent() (evt Event) {
	return Event{
		EvtTime:   job.DESJobRegTime,
		EvtAddr:   job.DESJobRegAddr,
		EvtUserID: job.DESJobRegUserID,
		EvtApp:    job.DESJobRegApp,
		EvtCode:   EVT_TYP_REGISTER_DEVICE.EvtTypCode,
		EvtTitle:  "A Device is Born",
		EvtMsg:    "Congratulations, it's a c001v001...",
	}
}

package c001v001

import (
	"encoding/json"
	"fmt"
	// "math/rand"
	"strings"

	"github.com/gofiber/fiber/v2"

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
			&Header{},
			&Config{},
			&EventTyp{},
			&Event{},
			&Sample{},
		); err != nil {
			return err
		}

		/* CREATE EVENT TYPES */
		for _, typ := range EVENT_TYPES {
			db.Create(&typ)
		}

		job.Admins = []Admin{job.RegisterJob_Default_JobAdmin()}
		if adm_res := db.Create(&job.Admins[0]); adm_res.Error != nil {
			fmt.Printf("\n(job *Job) RegisterJob() -> db.Create(&jobAdmins[0]) -> Error:\n%s\n", adm_res.Error.Error())
			return adm_res.Error
		}

		job.Headers = []Header{job.RegisterJob_Default_JobHeader()}
		if hdr_res := db.Create(&job.Headers[0]); hdr_res.Error != nil {
			fmt.Printf("\n(job *Job) RegisterJob() -> db.Create(&jobHeaderss[0]) -> Error:\n%s\n", hdr_res.Error.Error())
			return hdr_res.Error
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

		job.Samples = []Sample{}

		db.Close()
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
func (job *Job) RegisterJob_Default_JobHeader() (hdr Header) {
	return Header{
		HdrTime:   job.DESJobRegTime,
		HdrAddr:   job.DESJobRegAddr,
		HdrUserID: job.DESJobRegUserID,
		HdrApp:    job.DESJobRegApp,

		HdrWellCo: "UNKNOWN",
		HdrWellName: job.DESJobName,
		HdrWellSFLoc: "UNKNOWN",
		HdrWellBHLoc: "UNKNOWN",
		HdrWellLic: "UNKNOWN",

		HdrJobName:  job.DESJobName,
		HdrJobStart: job.DESJobStart,
		HdrJobEnd:   0,

		HdrGeoLng: job.DESJobLng,
		HdrGeoLat: job.DESJobLat,

		// HdrGeoLng: -114.75 + rand.Float32() * ( -110.15 - 114.75 ),
		// HdrGeoLat: 51.85 + rand.Float32() * ( 54.35 - 51.85 ),
	}
}
func (job *Job) RegisterJob_Default_JobConfig() (cfg Config) {
	return Config{
		CfgTime:   job.DESJobRegTime,
		CfgAddr:   job.DESJobRegAddr,
		CfgUserID: job.DESJobRegUserID,
		CfgApp:    job.DESJobRegApp,

		/* JOB */
		CfgSCVD:     596.8, // m
		CfgSCVDMult: 10.5,  // kPa / m
		CfgSSPRate:  1.95,  // kPa / hour
		CfgSSPDur:   6.0,   // hour
		CfgHiSCVF:   201.4, //  L/min

		/* VALVE */
		CfgVlvTgt: 2, // vent
		CfgVlvPos: 2, // vent

		/* OP PERIODS*/
		CfgOpSample: 1000,  // millisecond
		CfgOpLog:    1000, // millisecond
		CfgOpTrans:  1000, // millisecond

		/* DIAG PERIODS */
		CfgDiagSample: 10000,  // millisecond
		CfgDiagLog:    100000, // millisecond
		CfgDiagTrans:  600000, // millisecond
	}
}
func (job *Job) RegisterJob_Default_JobEvent() (evt Event) {
	return Event{
		EvtTime:   job.DESJobRegTime,
		EvtAddr:   job.DESJobRegAddr,
		EvtUserID: job.DESJobRegUserID,
		EvtApp:    job.DESJobRegApp,
		EvtCode:   EVENT_TYPES[0].EvtTypCode,
		EvtTitle:  "A Device is Born",
		EvtMsg:    `Congratulations, it's a class 001, version 001 device! This test is here to take up space; normal people would use the function that shits out latin but I don't. Partly because I don't remember what it is and the other reason is I don't feel like lookingit up.`,
	}
}

func (job *Job) GetJobData(limit int) (err error) {
	db := job.JDB()
	db.Connect()
	defer db.Close()
	db.Select("*").Table("admins").Limit(limit).Order("adm_time DESC").Scan(&job.Admins)
	db.Select("*").Table("headers").Limit(limit).Order("hdr_time DESC").Scan(&job.Headers)
	db.Select("*").Table("configs").Limit(limit).Order("cfg_time DESC").Scan(&job.Configs)
	db.Select("*").Table("events").Limit(limit).Order("evt_time DESC").Scan(&job.Events)
	db.Select("*").Table("samples").Limit(limit).Order("smp_time DESC").Scan(&job.Samples)
	for _, smp := range job.Samples {
		job.XYPoints.AppendXYSample(smp)
	}
	db.Close() 
	// pkg.Json("GetJobData(): job", job)
	return
}

func HandleGetEventTypeLists(c *fiber.Ctx) (err error) {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"event_types": EVENT_TYPES},
	})
}

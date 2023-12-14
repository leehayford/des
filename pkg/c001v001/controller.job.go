package c001v001

import (
	"fmt"
	"strings"
	"time"

	"github.com/leehayford/des/pkg"
	"gorm.io/gorm/clause"
)

type Job struct {
	Admins              []Admin  `json:"admins"`
	States              []State  `json:"states"`
	Headers             []Header `json:"headers"`
	Configs             []Config `json:"configs"`
	Events              []Event  `json:"events"`
	Samples             []Sample `json:"samples"`
	XYPoints            XYPoints `json:"xypoints"`
	Reports             []Report `json:"reports"`
	pkg.DESRegistration `json:"reg"`
	pkg.DBClient        `json:"-"`
}


/*************************************************************************************************************************/
/* AN OPEN Job.DBClient CONNECTION REQUIRED FOR ALL OTHER (job *Job)FUNCTIONS **************************/
func (job *Job) ConnectDBC() (err error) {
	job.DBClient = pkg.DBClient{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(job.DESJobName))}
	return job.DBClient.Connect()
}
/*************************************************************************************************************************/

/* RETURNS ALL DATA FOR THIS JOB */
func (job *Job) GetJobData() (err error) {
	// db := job.JDB()
	// if err = db.Connect(); err != nil {
	// 	return
	// }
	// defer db.Disconnect()

	job.DBClient.DB.Select("*").Table("admins").Order("adm_time ASC").Scan(&job.Admins)

	job.DBClient.DB.Select("*").Table("states").Order("sta_time ASC").Scan(&job.States)

	job.DBClient.DB.Select("*").Table("headers").Order("hdr_time ASC").Scan(&job.Headers)

	job.DBClient.DB.Select("*").Table("configs").Order("cfg_time ASC").Scan(&job.Configs)

	job.DBClient.DB.Select("*").Table("events").Order("evt_time ASC").Scan(&job.Events)

	job.DBClient.DB.Select("*").Table("samples").Order("smp_time ASC").Scan(&job.Samples)
	for _, smp := range job.Samples {
		job.XYPoints.AppendXYSample(smp)
	}

	x := job.DBClient.DB.Preload("RepSecs.SecDats").Preload("RepSecs.SecAnns.AnnEvt").Preload(clause.Associations).Find(&job.Reports)
	if x.Error != nil {
		err = x.Error
		return
	} // pkg.Json("GetJobData() -> Reports ", x.Error)

	// db.Disconnect()
	return
}

/* RETURNS ALL EVENTS FOR THIS JOB */
func (job *Job) GetJobEvents() (err error) {
	// db := job.JDB()
	// if err = db.Connect(); err != nil {
	// 	return
	// }
	// defer db.Disconnect()

	res := job.DBClient.DB.Select("*").Table("events").Order("evt_time ASC").Scan(&job.Events)
	err = res.Error

	// db.Disconnect()
	return
}

/* CREATES A RECORD IN THIS JOB'S REPORTS TABLE */
func (job *Job) CreateReport(rep *Report) {
	if rep.RepTitle == "" {
		rep.RepTitle = fmt.Sprintf("%s-%d", job.DESJobName, time.Now().UTC().UnixMilli())
	}

	/* WRITE TO JOB DB */
	job.DBClient.DB.Create(&rep)

	// db := job.JDB()
	// db.Connect()
	// defer db.Disconnect()
	// db.Create(&rep)
	// db.Disconnect()
	// pkg.Json("CreateReport( ): -> rep ", rep)
	return
}

/* CREATES A RECORD IN THIS JOB'S REPORT SECTIONS TABLE */
func (job *Job) CreateRepSection(rep *Report, start, end int64, name string) (sec *RepSection, err error) {

	sec = &RepSection{}
	sec.SecUserID = rep.RepUserID
	sec.SecRepID = rep.RepID
	sec.SecStart = start
	sec.SecEnd = end
	sec.SecName = name

	/* WRITE TO JOB DB */
	job.DBClient.DB.Create(&sec)

	// db := job.JDB()
	// db.Connect()
	// defer db.Disconnect()
	// db.Create(&sec)
	// db.Disconnect()
	// pkg.Json("CreateRepSection( ): -> sec ", sec)

	return
}

/* CREATES A RECORD IN THIS JOB'S REPORT SECTION DATASETS TABLE */
func (job *Job) CreateSecDataset(sec *RepSection, csv, plot bool, yaxis string, ymin, ymax float32) (dat *SecDataset, err error) {

	dat = &SecDataset{}
	dat.DatUserID = sec.SecUserID
	dat.DatSecID = sec.SecID
	dat.DatCSV = csv
	dat.DatPlot = plot
	dat.DatYAxis = yaxis
	dat.DatYMin = ymin
	dat.DatYMax = ymax

	/* WRITE TO JOB DB */
	job.DBClient.DB.Create(&dat)

	// db := job.JDB()
	// db.Connect()
	// defer db.Disconnect()
	// db.Create(&dat)
	// db.Disconnect()
	// pkg.Json("CreateSecDataset( ): -> dat ", dat)

	return
}

/* CREATES A RECORD IN THIS JOB'S REPORT SECTION ANNOTATIONS TABLE */
func (job *Job) AutoScaleSection(scls *SecScales, start, end int64) (err error) {

	/* GET MIN / MAX FOR EACH VALUE IN THE SECTION */
	// db := job.JDB()
	// db.Connect()
	// defer db.Disconnect()
	qry := job.DBClient.
		Table("samples").
		Select(`
			MIN(smp_ch4) min_ch4, 
			MAX(smp_ch4) max_ch4, 

			MIN(smp_hi_flow) min_hf, 
			MAX(smp_hi_flow) max_hf, 

			MIN(smp_lo_flow) min_lf, 
			MAX(smp_lo_flow) max_lf, 
 
			MIN(smp_press) min_press,
			MAX(smp_press) max_press
			`).
		Where(`smp_time >= ? AND smp_time <= ?`, start, end)

	res := qry.Scan(&scls)
	// db.Disconnect()
	// pkg.Json("AutoScaleSection(): SecScales", scls)
	err = res.Error
	if err != nil {
		return
	}

	/* ADD A MARGIN TO THE QUERIED RESULTS */
	m := float32(0.1)
	margin := (scls.MaxCH4 - scls.MinCH4) * m
	scls.MinCH4 -= margin
	scls.MaxCH4 += margin

	margin = (scls.MaxHF - scls.MinHF) * m
	scls.MinHF -= margin
	scls.MaxHF += margin

	margin = (scls.MaxLF - scls.MinLF) * m
	scls.MinLF -= margin
	scls.MaxLF += margin

	margin = (scls.MaxPress - scls.MinPress) * m
	scls.MinPress -= margin
	scls.MaxPress += margin

	/* FIXED SCALES ( FOR THIS PURPOSE ) */
	scls.MinBatAmp = 0
	scls.MaxBatAmp = 1.5

	scls.MinBatVolt = 0
	scls.MaxBatVolt = 15

	scls.MinMotVolt = 0
	scls.MaxMotVolt = 15

	// pkg.Json("AutoScaleSection(): SecScales -> with margin ", scls)

	return
}

/* CREATES A RECORD IN THIS JOB'S REPORT SECTION ANNOTATIONS TABLE */
func (job *Job) CreateSecAnnotation(sec *RepSection, csv, plot bool, evt Event) (ann *SecAnnotation, err error) {

	ann = &SecAnnotation{}
	ann.AnnUserID = sec.SecUserID
	ann.AnnSecID = sec.SecID
	ann.AnnCSV = csv
	ann.AnnPlot = plot
	ann.AnnEvtID = evt.EvtID

	/* WRITE TO JOB DB */
	job.DBClient.Create(&ann)

	// db := job.JDB()
	// db.Connect()
	// defer db.Disconnect()
	// db.Create(&ann)
	// db.Disconnect()
	// pkg.Json("CreateSecAnnotation( ): -> ann ", ann)

	return
}

/* CREATES A RECORD IN THIS JOB'S EVENTS TABLE */
func (job *Job) NewReportEvent(src string, evt *Event) (err error) {

	evt.EvtAddr = src
	evt.Validate()

	/* WRITE TO JOB DB */
	job.DBClient.Create(&evt)

	// db := job.JDB()
	// db.Connect()
	// defer db.Disconnect()
	// db.Create(&evt)
	// db.Disconnect()

	return
}

/* GENERATES A REPORT WITH SECTIONS BASED ON VALVE  TARGET; RUNS:
	- AUTOMATICALLY WHEN A JOB HAS ENDED 
	- WHEN A USER ADDS A NEW REPORT MANUALLY
*/
func (job *Job) GenerateReport(rep *Report) {

	/* GET START & END OF JOB */
	start := job.DESJobStart

	job.GetJobData()
	job.CreateReport(rep)
	// pkg.Json("CreateDefaultReport( ): -> rep ", rep)

	buildCount := 0
	ventCount := 0
	flowCount := 0

	secStart := start
	secEnd := start
	secName := "Job Start"
	curCFG := job.Configs[0]
	pkg.Json("CreateDefaultReport( ): -> curCFG ", curCFG)

	/* CREATE SECTIONS BY CFG */
	for _, cfg := range job.Configs {

		/*  CREATE NEW SECTIONS FOR EACH MODE CHANGE */
		if cfg.CfgVlvTgt != curCFG.CfgVlvTgt && cfg.CfgAddr == job.DESDevSerial {

			fmt.Printf("\ncfg.CfgVlvTgt: %d -> curCFG.CfgVlvTgt %d", cfg.CfgVlvTgt, curCFG.CfgVlvTgt)

			secEnd = cfg.CfgTime

			job.CreateSectionByConfig(rep, &secStart, &secEnd, &buildCount, &ventCount, &flowCount, secName, curCFG)

			/* UPDATE START TIME AND CURRENT MODE */
			secStart = cfg.CfgTime
		}

		if cfg.CfgAddr == job.DESDevSerial {
			curCFG = cfg
			pkg.Json("CreateDefaultReport( ): -> update curCFG ", curCFG)
		}
	}

	/* CREATE SECTION FOR SAMPLES COLLECTED AFTER FINAL MODE CHANGE */
	secEnd = job.DESJobEnd
	job.CreateSectionByConfig(rep, &secStart, &secEnd, &buildCount, &ventCount, &flowCount, secName, curCFG)
	return
}
func (job *Job) CreateSectionByConfig(rep *Report, secStart, secEnd *int64, buildCount, ventCount, flowCount *int, secName string, curCFG Config) {

	switch curCFG.CfgVlvTgt {

	case MODE_BUILD:
		*buildCount++
		secName = fmt.Sprintf("Pressure Build-Up %d", *buildCount)
		job.CreateBuildUpSection(rep, *secStart, *secEnd, secName, curCFG)

	case MODE_VENT:
		*ventCount++
		secName = fmt.Sprintf("Vent %d", *ventCount)
		job.CreateVentSection(rep, *secStart, *secEnd, secName, curCFG)

	case MODE_HI_FLOW:
		*flowCount++
		secName = fmt.Sprintf("Flow %d", *flowCount)
		job.CreateFlowSection(rep, *secStart, *secEnd, secName, curCFG)

	case MODE_LO_FLOW:
		*flowCount++
		secName = fmt.Sprintf("Flow %d", *flowCount)
		job.CreateFlowSection(rep, *secStart, *secEnd, secName, curCFG)
	}
	return
}
func (job *Job) CreateBuildUpSection(rep *Report, start, end int64, name string, cfg Config) {

	sec, err := job.CreateRepSection(rep, start, end, name)
	if err != nil {
		pkg.LogErr(err)
	}
	pkg.Json("CreateBuildUpSection( ): -> CreateRepSection( ) -> sec ", sec)

	/* ADD DATASETS */
	scls := &SecScales{}
	job.AutoScaleSection(scls, start, end)

	ch4, err := job.CreateSecDataset(sec, true, true, "y_ch4", scls.MinCH4, scls.MaxCH4)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateBuildUpSection( ): -> CreateSecDataset( ) -> ch4 ", ch4)
	sec.SecDats = append(sec.SecDats, *ch4)

	hf, err := job.CreateSecDataset(sec, true, false, "y_hi_flow", scls.MinHF, scls.MaxHF)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateBuildUpSection( ): -> CreateSecDataset( ) -> hf ", hf)
	sec.SecDats = append(sec.SecDats, *hf)

	lf, err := job.CreateSecDataset(sec, true, true, "y_lo_flow", scls.MinLF, scls.MaxLF)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateBuildUpSection( ): -> CreateSecDataset( ) -> lf ", lf)
	sec.SecDats = append(sec.SecDats, *lf)

	p, err := job.CreateSecDataset(sec, true, true, "y_press", scls.MinPress, scls.MaxPress)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateBuildUpSection( ): -> CreateSecDataset( ) -> p ", p)
	sec.SecDats = append(sec.SecDats, *p)

	ba, err := job.CreateSecDataset(sec, true, false, "y_bat_amp", scls.MinBatAmp, scls.MaxBatAmp)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateBuildUpSection( ): -> CreateSecDataset( ) -> ba ", ba)
	sec.SecDats = append(sec.SecDats, *ba)

	bv, err := job.CreateSecDataset(sec, true, false, "y_bat_volt", scls.MinBatVolt, scls.MaxBatVolt)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateBuildUpSection( ): -> CreateSecDataset( ) -> bv ", bv)
	sec.SecDats = append(sec.SecDats, *bv)

	mv, err := job.CreateSecDataset(sec, true, false, "y_mot_volt", scls.MinMotVolt, scls.MaxMotVolt)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateBuildUpSection( ): -> CreateSecDataset( ) -> mv ", mv)
	sec.SecDats = append(sec.SecDats, *mv)

	/* ADD ANNOTATION FOR EACH EVENT */
	for _, evt := range job.Events {
		/* TODO: HANDLE ALARM EVENTS */
		if evt.EvtTime >= start && evt.EvtTime <= end {
			job.CreateSecAnnotation(sec, true, true, evt)
		}
	}

	/* CALCULATE SSP
	GET SSP START / END
	CREATE EVENT / ANNOTATION SSP START
	CREATE EVENT / ANNOTATION SSP END
	*/
	return
}
func (job *Job) CreateVentSection(rep *Report, start, end int64, name string, cfg Config) {

	sec, err := job.CreateRepSection(rep, start, end, name)
	if err != nil {
		pkg.LogErr(err)
	}
	pkg.Json("CreateVentSection( ): -> CreateRepSection( ) -> sec ", sec)

	/* ADD DATASETS */
	scls := &SecScales{}
	job.AutoScaleSection(scls, start, end)

	ch4, err := job.CreateSecDataset(sec, true, true, "y_ch4", scls.MinCH4, scls.MaxCH4)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> ch4 ", ch4)
	sec.SecDats = append(sec.SecDats, *ch4)

	hf, err := job.CreateSecDataset(sec, true, false, "y_hi_flow", scls.MinHF, scls.MaxHF)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> hf ", hf)
	sec.SecDats = append(sec.SecDats, *hf)

	lf, err := job.CreateSecDataset(sec, true, true, "y_lo_flow", scls.MinLF, scls.MaxLF)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> lf ", lf)
	sec.SecDats = append(sec.SecDats, *lf)

	p, err := job.CreateSecDataset(sec, true, true, "y_press", scls.MinPress, scls.MaxPress)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> p ", p)
	sec.SecDats = append(sec.SecDats, *p)

	ba, err := job.CreateSecDataset(sec, true, false, "y_bat_amp", scls.MinBatAmp, scls.MaxBatAmp)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> ba ", ba)
	sec.SecDats = append(sec.SecDats, *ba)

	bv, err := job.CreateSecDataset(sec, true, false, "y_bat_volt", scls.MinBatVolt, scls.MaxBatVolt)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> bv ", bv)
	sec.SecDats = append(sec.SecDats, *bv)

	mv, err := job.CreateSecDataset(sec, true, false, "y_mot_volt", scls.MinMotVolt, scls.MaxMotVolt)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> mv ", mv)
	sec.SecDats = append(sec.SecDats, *mv)

	/* ADD ANNOTATION FOR EACH EVENT */
	for _, evt := range job.Events {
		/* TODO: HANDLE ALARM EVENTS */
		if evt.EvtTime >= start && evt.EvtTime <= end {
			job.CreateSecAnnotation(sec, true, true, evt)
		}
	}
	return
}
func (job *Job) CreateFlowSection(rep *Report, start, end int64, name string, cfg Config) {

	sec, err := job.CreateRepSection(rep, start, end, name)
	if err != nil {
		pkg.LogErr(err)
	}
	pkg.Json("CreateFlowSection( ): -> CreateRepSection( ) -> sec ", sec)

	/* ADD DATASETS */
	scls := &SecScales{}
	job.AutoScaleSection(scls, start, end)

	ch4, err := job.CreateSecDataset(sec, true, true, "y_ch4", scls.MinCH4, scls.MaxCH4)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateFlowSection( ): -> CreateSecDataset( ) -> ch4 ", ch4)
	sec.SecDats = append(sec.SecDats, *ch4)

	plotHF := false
	if scls.MaxLF > cfg.CfgFlowTog {
		plotHF = true
	}
	hf, err := job.CreateSecDataset(sec, plotHF, plotHF, "y_hi_flow", scls.MinHF, scls.MaxHF)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateFlowSection( ): -> CreateSecDataset( ) -> hf ", hf)
	sec.SecDats = append(sec.SecDats, *hf)

	lf, err := job.CreateSecDataset(sec, !plotHF, !plotHF, "y_lo_flow", scls.MinLF, scls.MaxLF)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateFlowSection( ): -> CreateSecDataset( ) -> lf ", lf)
	sec.SecDats = append(sec.SecDats, *lf)

	p, err := job.CreateSecDataset(sec, true, true, "y_press", scls.MinPress, scls.MaxPress)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateFlowSection( ): -> CreateSecDataset( ) -> p ", p)
	sec.SecDats = append(sec.SecDats, *p)

	ba, err := job.CreateSecDataset(sec, true, false, "y_bat_amp", scls.MinBatAmp, scls.MaxBatAmp)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateFlowSection( ): -> CreateSecDataset( ) -> ba ", ba)
	sec.SecDats = append(sec.SecDats, *ba)

	bv, err := job.CreateSecDataset(sec, true, false, "y_bat_volt", scls.MinBatVolt, scls.MaxBatVolt)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateFlowSection( ): -> CreateSecDataset( ) -> bv ", bv)
	sec.SecDats = append(sec.SecDats, *bv)

	mv, err := job.CreateSecDataset(sec, true, false, "y_mot_volt", scls.MinMotVolt, scls.MaxMotVolt)
	if err != nil {
		pkg.LogErr(err)
	}
	// pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> mv ", mv)
	sec.SecDats = append(sec.SecDats, *mv)

	/* ADD ANNOTATION FOR EACH EVENT */
	for _, evt := range job.Events {
		/* TODO: HANDLE ALARM EVENTS */
		if evt.EvtTime >= start && evt.EvtTime <= end {
			job.CreateSecAnnotation(sec, true, true, evt)
		}
	}

	/* CALCULATE SCVF
	GET SCVF START / END
	CREATE EVENT / ANNOTATION SCVF START
	CREATE EVENT / ANNOTATION SCVF END
	*/
	return
}

/* NOT CURRENTLY IN USE... */
func (job *Job) GetJobData_Limited(limit int) (err error) {
	// db := job.JDB()
	// db.Connect()
	// defer db.Disconnect()
	job.DBClient.DB.Select("*").Table("admins").Limit(limit).Order("adm_time DESC").Scan(&job.Admins)
	job.DBClient.DB.Select("*").Table("headers").Limit(limit).Order("hdr_time DESC").Scan(&job.Headers)
	job.DBClient.DB.Select("*").Table("configs").Limit(limit).Order("cfg_time DESC").Scan(&job.Configs)
	job.DBClient.DB.Select("*").Table("events").Limit(limit).Order("evt_time DESC").Scan(&job.Events)
	job.DBClient.DB.Select("*").Table("samples").Limit(limit).Order("smp_time DESC").Scan(&job.Samples)
	for _, smp := range job.Samples {
		job.XYPoints.AppendXYSample(smp)
	}
	// db.Disconnect()
	// pkg.Json("GetJobData(): job", job)
	return
}

type XYPoint struct {
	X int64   `json:"x"`
	Y float32 `json:"y"`
}

type XYPoints struct {
	CH4     []XYPoint `json:"ch4"`
	HiFlow  []XYPoint `json:"hi_flow"`
	LoFlow  []XYPoint `json:"lo_flow"`
	Press   []XYPoint `json:"press"`
	BatAmp  []XYPoint `json:"bat_amp"`
	BatVolt []XYPoint `json:"bat_volt"`
	MotVolt []XYPoint `json:"mot_volt"`
	VlvTgt  []XYPoint `json:"vlv_tgt"`
	VlvPos  []XYPoint `json:"vlv_pos"`
}

func (xys *XYPoints) AppendXYSample(smp Sample) {
	xys.CH4 = append(xys.CH4, XYPoint{X: smp.SmpTime, Y: smp.SmpCH4})
	xys.HiFlow = append(xys.HiFlow, XYPoint{X: smp.SmpTime, Y: smp.SmpHiFlow})
	xys.LoFlow = append(xys.LoFlow, XYPoint{X: smp.SmpTime, Y: smp.SmpLoFlow})
	xys.Press = append(xys.Press, XYPoint{X: smp.SmpTime, Y: smp.SmpPress})
	xys.BatAmp = append(xys.BatAmp, XYPoint{X: smp.SmpTime, Y: smp.SmpBatAmp})
	xys.BatVolt = append(xys.BatVolt, XYPoint{X: smp.SmpTime, Y: smp.SmpBatVolt})
	xys.MotVolt = append(xys.MotVolt, XYPoint{X: smp.SmpTime, Y: smp.SmpMotVolt})
	xys.VlvTgt = append(xys.VlvTgt, XYPoint{X: smp.SmpTime, Y: float32(smp.SmpVlvTgt)})
	xys.VlvPos = append(xys.VlvPos, XYPoint{X: smp.SmpTime, Y: float32(smp.SmpVlvPos)})
}

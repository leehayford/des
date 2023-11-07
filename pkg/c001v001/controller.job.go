package c001v001

import (
	"fmt"
	"strings"

	"github.com/leehayford/des/pkg"
	"gorm.io/gorm/clause"
)

type Job struct {
	Admins              []Admin  `json:"admins"`
	States				[]State `json:"states"`
	Headers             []Header `json:"headers"`
	Configs             []Config `json:"configs"`
	Events              []Event  `json:"events"`
	Samples             []Sample `json:"samples"`
	XYPoints            XYPoints `json:"xypoints"`
	Reports				[]Report `json:"reports"`
	pkg.DESRegistration `json:"reg"`
	pkg.DBClient        `json:"-"`
}

func (job *Job) JDBX() {
	job.DBClient = pkg.DBClient{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(job.DESJobName))}

}
func (job *Job) JDB() *pkg.DBClient {
	return &pkg.DBClient{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(job.DESJobName))}
}

/* GET THE DESRegistration FOR ALL DEVICES ON THIS DES */
func GetJobList() (jobs []pkg.DESRegistration, err error) {

	qry := pkg.DES.DB.
		Table("des_jobs").
		Select("des_jobs.*, des_devs.*, des_job_searches.*").
		Joins("JOIN des_devs ON des_jobs.des_job_dev_id = des_devs.des_dev_id").
		Joins("JOIN des_job_searches ON des_jobs.des_job_id = des_job_searches.des_job_key").
		Where("des_jobs.des_job_end != 0").
		Order("des_devs.des_dev_id ASC, des_jobs.des_job_start DESC")

	res := qry.Scan(&jobs)
	err = res.Error
	return
}

/* GET THE SEARCHABLE DATA FOR ALL JOBS IN THE LIST OF DESRegistrations */
func GetJobs(regs []pkg.DESRegistration) (jobs []Job) {
	for _, reg := range regs {
		// pkg.Json("GetJobs( ) -> reg", reg)
		job := Job{}
		job.DESRegistration = reg
		jobs = append(jobs, job)
	}
	// pkg.Json("GetJobs(): Jobs", jobs)
	return
}

/* RETURNS ALL DATA ASSOCIATED WITH A JOB */
func (job *Job) GetJobData() (err error) {
	db := job.JDB()
	db.Connect()
	defer db.Disconnect()
	db.Select("*").Table("admins").Order("adm_time ASC").Scan(&job.Admins)
	db.Select("*").Table("states").Order("sta_time ASC").Scan(&job.States)
	db.Select("*").Table("headers").Order("hdr_time ASC").Scan(&job.Headers)
	db.Select("*").Table("configs").Order("cfg_time ASC").Scan(&job.Configs)
	db.Select("*").Table("events").Order("evt_time ASC").Scan(&job.Events)
	db.Select("*").Table("samples").Order("smp_time ASC").Scan(&job.Samples)
	for _, smp := range job.Samples {
		job.XYPoints.AppendXYSample(smp)
	}
	// db.Select("*").Table("reports").Order("rep_created ASC").Scan(&job.Reports)
	db.Preload("RepSecs.SecDats").Preload(clause.Associations).Find(&job.Reports)
	db.Disconnect()
	// pkg.Json("GetJobData(): job.Headers", job.Headers)
	return
}



/* RUNS AUTOMATICALLY WHEN A JOB HAS ENDED */
func (job *Job) CreateDefaultReport(rep *Report) {

	/* GET START & END OF JOB */
	start := job.DESJobStart
	
	job.GetJobData()
	rep.CreateReport(job)
	// pkg.Json("CreateDefaultReport( ): -> rep ", rep)

	buildCount := 0
	ventCount := 0
	flowCount := 0

	secStart := start
	secName := "Job Start"
	curCFG := job.Configs[0]
	pkg.Json("CreateDefaultReport( ): -> curCFG ", curCFG)

	/* CREATE SECTIONS BY CFG */
	for _, cfg := range job.Configs {

		/*  CREATE NEW SECTIONS FOR EACH MODE CHANGE */
		if cfg.CfgVlvTgt != curCFG.CfgVlvTgt {

			switch curCFG.CfgVlvTgt {
	
			case MODE_BUILD:
				buildCount++
				secName = fmt.Sprintf("Pressure Build-Up %d", buildCount)
				job.CreateBuildUpSection(rep, secStart, cfg.CfgTime, secName, curCFG)
	
			case MODE_VENT:
				ventCount++
				secName = fmt.Sprintf("Vent %d", ventCount)
				job.CreateVentSection(rep, secStart, cfg.CfgTime, secName, curCFG)
	
			case MODE_HI_FLOW:
			case MODE_LO_FLOW:
				flowCount++
				secName = fmt.Sprintf("Flow %d", flowCount)
				job.CreateFlowSection(rep, secStart, cfg.CfgTime, secName, curCFG)
			}
			// sec.CreateRepSection(rep, secStart, cfg.CfgTime, secName)
			/* ADD DATASETS */
			// CH4
			// HI FLOW
			// LO FLOW
			// PRESSURE 
			// BAT VOLT
			// MOT AMP
			// VALVE TGT
			// VALVE POS
			
			/* UPDATE START TIME AND CURRENT MODE */
			secStart = cfg.CfgTime
		}

		curCFG = cfg
	}


	/* ADD ANNOTATION FOR EACH EVENT */
	// for _, evt := range job.Events {

	// }


}

func (job *Job) CreateBuildUpSection(rep *Report, start, end int64, name string, cfg Config) {

	sec, err := job.CreateRepSection(rep, start, end, name)
	if err != nil {
		pkg.TraceErr(err)
	}
	pkg.Json("CreateBuildUpSection( ): -> CreateRepSection( ) -> sec ", sec)

	/* ADD DATASETS */
	scls := &SecScales{}
	scls.AutoScaleSection(job, start, end)

	/* CALCULATE SSP 
		GET SSP START / END
		CREATE EVENT / ANNOTATION SSP START
		CREATE EVENT / ANNOTATION SSP END
	*/

}

func (job *Job) CreateVentSection(rep *Report, start, end int64, name string, cfg Config) {

	sec, err := job.CreateRepSection(rep, start, end, name)
	if err != nil {
		pkg.TraceErr(err)
	}
	pkg.Json("CreateVentSection( ): -> CreateRepSection( ) -> sec ", sec)

	/* ADD DATASETS */
	scls := &SecScales{}
	scls.AutoScaleSection(job, start, end)

	ch4, err := job.CreateSecDataset(sec, true, false, "y_ch4", scls.MinCH4, scls.MaxCH4)
	if err != nil {
		pkg.TraceErr(err)
	}
	pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> ch4 ", ch4)
	
	hf, err := job.CreateSecDataset(sec, true, false, "y_hi_flow", scls.MinHF, scls.MaxHF)
	if err != nil {
		pkg.TraceErr(err)
	}
	pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> hf ", hf)
	
	lf, err := job.CreateSecDataset(sec, true, false, "y_lo_flow", scls.MinLF, scls.MaxLF)
	if err != nil {
		pkg.TraceErr(err)
	}
	pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> lf ", lf)

	p, err := job.CreateSecDataset(sec, true, false, "y_press", scls.MinPress, scls.MaxPress)
	if err != nil {
		pkg.TraceErr(err)
	}
	pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> p ", p)

	ba, err := job.CreateSecDataset(sec, true, false, "y_bat_amp", scls.MinBatAmp, scls.MaxBatAmp)
	if err != nil {
		pkg.TraceErr(err)
	}
	pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> ba ", ba)


	bv, err := job.CreateSecDataset(sec, true, false, "y_bat_volt", scls.MinBatVolt, scls.MaxBatVolt)
	if err != nil {
		pkg.TraceErr(err)
	}
	pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> bv ", bv)


	mv, err := job.CreateSecDataset(sec, true, false, "y_mot_volt", scls.MinMotVolt, scls.MaxMotVolt)
	if err != nil {
		pkg.TraceErr(err)
	}
	pkg.Json("CreateVentSection( ): -> CreateSecDataset( ) -> mv ", mv)


}

func (job *Job) CreateFlowSection(rep *Report, start, end int64, name string, cfg Config) {

	sec, err := job.CreateRepSection(rep, start, end, name)
	if err != nil {
		pkg.TraceErr(err)
	}
	pkg.Json("CreateFlowSection( ): -> CreateRepSection( ) -> sec ", sec)

	/* ADD DATASETS */
	scls := &SecScales{}
	scls.AutoScaleSection(job, start, end)

	ch4 := SecDataset{
		DatUserID: rep.RepUserID,
		DatSecID: sec.SecID,
		DatCSV: true,
		DatPlot: true,
		DatYAxis: "y_ch4",
		DatYMin: scls.MinCH4,
		DatYMax: scls.MaxCH4,
	}
	pkg.Json("CreateFlowSection( ): -> SecDataset -> ch4 ", ch4)
	/* WRITE TO DB */

	hf := SecDataset{
		DatUserID: rep.RepUserID,
		DatSecID: sec.SecID,
		DatYAxis: "y_hi_flow",
	}
	lf := SecDataset{
		DatUserID: rep.RepUserID,
		DatSecID: sec.SecID,
		DatYAxis: "y_lo_flow",
	}
	if scls.MaxLF > cfg.CfgFlowTog {
		hf.DatCSV = true
		hf.DatPlot = true
		hf.DatYMin = scls.MinHF
		hf.DatYMax = scls.MaxHF

		lf.DatCSV = false
		lf.DatPlot = false
		lf.DatYMin = scls.MinHF
		lf.DatYMax = scls.MaxHF
	} else {
		hf.DatCSV = false
		hf.DatPlot = false
		hf.DatYMin = scls.MinLF
		hf.DatYMax = scls.MaxLF

		lf.DatCSV = true
		lf.DatPlot = true
		lf.DatYMin = scls.MinLF
		lf.DatYMax = scls.MaxLF
	}
	pkg.Json("CreateFlowSection( ): -> SecDataset -> hf ", hf)
	pkg.Json("CreateFlowSection( ): -> SecDataset -> lf ", lf)
	/* WRITE TO DB */
	
	p := SecDataset{
		DatUserID: rep.RepUserID,
		DatSecID: sec.SecID,
		DatCSV: true,
		DatPlot: true,
		DatYAxis: "y_press",
		DatYMin: scls.MinPress,
		DatYMax: scls.MaxPress,
	}
	pkg.Json("CreateFlowSection( ): -> SecDataset -> p ", p)
	/* WRITE TO DB */

	/* CALCULATE SCVF 
		GET SCVF START / END
		CREATE EVENT / ANNOTATION SCVF START
		CREATE EVENT / ANNOTATION SCVF END
	*/

}







/* NOT CURRENTLY IN USE... */
func (job *Job) GetJobData_Limited(limit int) (err error) {
	db := job.JDB()
	db.Connect()
	defer db.Disconnect()
	db.Select("*").Table("admins").Limit(limit).Order("adm_time DESC").Scan(&job.Admins)
	db.Select("*").Table("headers").Limit(limit).Order("hdr_time DESC").Scan(&job.Headers)
	db.Select("*").Table("configs").Limit(limit).Order("cfg_time DESC").Scan(&job.Configs)
	db.Select("*").Table("events").Limit(limit).Order("evt_time DESC").Scan(&job.Events)
	db.Select("*").Table("samples").Limit(limit).Order("smp_time DESC").Scan(&job.Samples)
	for _, smp := range job.Samples {
		job.XYPoints.AppendXYSample(smp)
	}
	db.Disconnect()
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

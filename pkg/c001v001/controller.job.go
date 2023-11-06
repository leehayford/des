package c001v001

import (
	"fmt"
	"strings"

	"github.com/leehayford/des/pkg"
)

type Job struct {
	Admins              []Admin  `json:"admins"`
	States				[]State `json:"states"`
	Headers             []Header `json:"headers"`
	Configs             []Config `json:"configs"`
	Events              []Event  `json:"events"`
	Samples             []Sample `json:"samples"`
	XYPoints            XYPoints `json:"xypoints"`
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

	// qry := pkg.DES.DB.
	// 	Table("des_jobs").
	// 	Select("des_jobs.*, djs.*, des_devs.*").
	// 	Joins("des_job_searches djs ON des_jobs.des_job_id = djs.des_job_key").
	// 	Joins("JOIN des_devs ON des_jobs.des_job_dev_id = des_devs.des_dev_id").
	// 	Order("des_devs.des_dev_id ASC, des_jobs.des_job_start DESC")

	// qry := pkg.DES.DB.
	// 	Table("des_jobs").
	// 	Select("des_jobs.*, des_devs.*").
	// 	Joins("JOIN des_devs ON des_jobs.des_job_dev_id = des_devs.des_dev_id").
	// 	Where("des_jobs.des_job_end != 0").
	// 	Order("des_devs.des_dev_id ASC, des_jobs.des_job_start DESC")

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
	db.Select("*").Table("admins").Order("adm_time DESC").Scan(&job.Admins)
	db.Select("*").Table("admins").Order("adm_time DESC").Scan(&job.States)
	db.Select("*").Table("headers").Order("hdr_time DESC").Scan(&job.Headers)
	db.Select("*").Table("configs").Order("cfg_time DESC").Scan(&job.Configs)
	db.Select("*").Table("events").Order("evt_time DESC").Scan(&job.Events)
	db.Select("*").Table("samples").Order("smp_time ASC").Scan(&job.Samples)
	for _, smp := range job.Samples {
		job.XYPoints.AppendXYSample(smp)
	}
	db.Disconnect()
	pkg.Json("GetJobData(): job.Headers", job.Headers)
	return
}



/* RUNS AUTOMATICALLY WHEN A JOB HAS ENDED */
func (job *Job) CreateDefaultReport(end_evt Event) {

	/* GET START & END OF JOB */
	start := job.DESJobStart
	end := job.DESJobEnd
	
	rep := &Report{}
	rep.RepUserID = end_evt.EvtUserID 
	rep.CreateReport(job)

	sec := &RepSection{}
	sec.SecUserID = end_evt.EvtUserID
	sec.CreateRepSection(rep, start, end, "Basic")
	
	

	// for _, evt := range job.Events {

	// }

	// for _, cfg := range job.Configs {

	// }

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

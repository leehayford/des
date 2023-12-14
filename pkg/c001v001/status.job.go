package c001v001

import (
	// "fmt"
	// "sync"

	"github.com/leehayford/des/pkg"
)

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

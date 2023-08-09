package c001v001

import (
	"strings"

	"github.com/leehayford/des/pkg"
)

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

	if (!existing) {
		/* CREATE NEW DATABASE */
		db :=  job.JDB()
		db.Connect()
		defer db.Close()
		
		if err = db.Migrator().CreateTable(
			&JobAdmin{},
			&JobConfig{},
			&JobEvent{},
			&JobEventType{},
			&JobSample{},
		); err != nil {
			return err
		}
		/* TODO: CREATE EVENT TYPE RECORDS */
	}

	return
}
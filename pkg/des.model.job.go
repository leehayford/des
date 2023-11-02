/* Data Exchange Server (DES) is a component of the Datacan Data2Desk (D2D) Platform.
License:

	[PROPER LEGALESE HERE...]

	INTERIM LICENSE DESCRIPTION:
	In spirit, this license:
	1. Allows <Third Party> to use, modify, and / or distributre this software in perpetuity so long as <Third Party> understands:
		a. The software is porvided as is without guarantee of additional support from DataCan in any form.
		b. The software is porvided as is without guarantee of exclusivity.

	2. Prohibits <Third Party> from taking any action which might interfere with DataCan's right to use, modify and / or distributre this software in perpetuity.
*/

package pkg

type DESJob struct {
	DESJobID int64 `gorm:"unique; primaryKey" json:"des_job_id"`

	DESJobRegTime   int64  `gorm:"not null" json:"des_job_reg_time"`
	DESJobRegAddr   string `json:"des_job_reg_addr"`
	DESJobRegUserID string `gorm:"not null; varchar(36)" json:"des_job_reg_user_id"`
	DESJobRegApp    string `gorm:"not null; varchar(36)" json:"des_job_reg_app"`

	DESJobName  string  `gorm:"not null; unique; varchar(24)" json:"des_job_name"`
	DESJobStart int64   `gorm:"not null" json:"des_job_start"`
	DESJobEnd   int64   `gorm:"not null" json:"des_job_end"`
	DESJobLng   float64 `json:"des_job_lng"`
	DESJobLat   float64 `json:"des_job_lat"`
	DESJobDevID int64   `json:"des_job_dev_id"`

	DESDev DESDev `gorm:"foreignKey:DESJobDevID" json:"-"`
	User   User   `gorm:"foreignKey:DESJobRegUserID" json:"-"`
}
func WriteDESJob(job *DESJob) (err error) {
	job.DESJobID = 0
	res := DES.DB.Create(&job)
	return res.Error
}
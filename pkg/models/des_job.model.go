
/* Data Exchange Server (DES) is a component of the Datacan Data2Desk (D2D) Platform.
License:

	[PROPER LEGALESE HERE...]

	INTERIM LICENSE DESCRIPTION:
	In spirit, this license:
	1. Allows <Third Party> to use, modify, adn / or distributre this software in perpetuity so long as <Third Party> understands:
		a. The software is porvided as is without guarantee of additional support from DataCan in any form.
		b. The software is porvided as is without guarantee of exclusivity.

	2. Prohibits <Third Party> from taking any action which might interfere with DataCan's right to use, modify, distributre this software in perpetuity.
*/

package models

type DESJob struct {
	DESJobID int64 `gorm:"unique; primaryKey" json:"des_job_id"`

	DESJobRegTime   int64  `gorm:"not null" json:"des_job_reg_time"`
	DESJobRegAddr   string `json:"des_job_reg_addr"`
	DESJobRegUesrID int64  `gorm:"not null" json:"des_job_reg_user_id"`
	DESJobRegApp    string `gorm:"not null" json:"des_job_reg_app"`

	DESJobName  string `gorm:"not null; unique; varchar(27)" json:"des_job_name"`
	DESJobStart int64  `gorm:"not null" json:"des_job_start"`
	DESJobEnd   int64  `gorm:"not null" json:"des_job_end"`
	DESJobDevID int `json:"des_job_dev_id"`
	DESDev DESDev `gorm:"foreignKey:DESJobDevID" json:"-"`
}

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

type DESJobSearch struct {
	DESJobSearchID int64 `gorm:"unique; primaryKey" json:"des_job_search_id"`
	DESJobToken string `gorm:"not null" json:"des_job_token"`
	DESJobKey int64 `json:"des_job_key"`
	DESJob `gorm:"foreignKey:DESJobKey; references:des_job_key" json:"-"`
}

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

type DESDev struct {
	DESDevID int64 `gorm:"unique; primaryKey" json:"des_dev_id"`	
	
	DESDevRegTime   int64  `gorm:"not null" json:"des_dev_reg_time"`
	DESDevRegAddr   string `json:"des_dev_reg_addr"`
	DESDevRegUserID string `gorm:"not null; varchar(36)" json:"des_dev_reg_user_id"`
	DESDevRegApp    string `gorm:"not null; varchar(36)" json:"des_dev_reg_app"`

	DESDevSerial  string   `gorm:"not null; varchar(10)" json:"des_dev_serial"`
	DESDevVersion string   `gorm:"not null; varchar(3)" json:"des_dev_version"`
	DESDevClass   string   `gorm:"not null; varchar(3)" json:"des_dev_class"`
	DESJobs []DESJob `gorm:"foreignKey:DESJobDevID" json:"-"`
	User User `gorm:"foreignKey:DESDevRegUserID" json:"-"`
}


func GetDesDevList(devices *[]DESDev) (err error) {

	/*
		WHERE A DEVICE HAS MORE THAN ONE REGISTRATION RECORD
		WE WANT THE LATEST
	*/
	subQry := DES.DB.
		Table("des_devs").
		Select(`des_dev_serial, MAX(des_dev_reg_time) AS max_time`).
		Group("des_dev_serial")

	qry := DES.DB.
		Select(" * ").
		Joins(`JOIN ( ? ) x 
		ON des_devs.des_dev_serial = x.des_dev_serial 
		AND des_devs.des_dev_reg_time = x.max_time`,
			subQry).
		Order("des_devs.des_dev_serial DESC")

		res := qry.Find(&devices)
		return res.Error
}



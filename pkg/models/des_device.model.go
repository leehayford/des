
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

import (
	"github.com/google/uuid" // go get github.com/google/uuid
)

type DESDev struct {
	DESDevID int `gorm:"unique; primaryKey" json:"des_dev_id"`	
	
	DESDevRegTime   int64  `gorm:"not null" json:"des_dev_reg_time"`
	DESDevRegAddr   string `json:"des_dev_reg_addr"`
	DESDevRegUesrID *uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()" json:"des_dev_reg_user_id"`
	DESDevRegApp    string `gorm:"not null" json:"des_dev_reg_app"`

	DESDevSerial  string   `gorm:"not null; varchar(10)" json:"des_dev_serial"`
	DESDevVersion string   `gorm:"not null; varchar(3)" json:"des_dev_version"`
	DESDevClass   string   `gorm:"not null; varchar(3)" json:"des_dev_class"`
	DESJobs []DESJob `gorm:"foreignKey:DESJobDevID" json:"-"`
}
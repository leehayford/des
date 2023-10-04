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

import (
	"strings"
/* TODO: LOOK INTO USING JSON FIELD FOR TOKEN... */
// 	"database/sql/driver"
// 	"encoding/json"
// 	"errors"
)

type DESRegistration struct {
	DESDev //`json:"des_dev"`
	DESJob //`json:"des_job"`
}

/* TODO: LOOK INTO USING JSON FIELD FOR TOKEN... */
type DESJobSearch struct {
	DESJobSearchID int64  `gorm:"unique; primaryKey" json:"des_job_search_id"`
	DESJobToken    string `gorm:"not null" json:"des_job_token"`
	DESJobKey      int64  `json:"des_job_key"`
	DESJob         DESJob `gorm:"foreignKey:DESJobKey; references:des_job_id" json:"-"`
}

func SearchDESJobsByToken(token string) (regs []DESRegistration, err error) {

	token = "%" + token + "%"

	qry := DES.DB.
		Table("des_job_searches").
		Select("des_devs.*, des_jobs.*").
		Joins("JOIN des_jobs ON des_job_searches.des_job_key = des_jobs.des_job_id").
		Joins("JOIN des_devs ON des_jobs.des_job_dev_id = des_devs.des_dev_id").
		Where("des_job_searches.des_job_token LIKE ?", token)

	res := qry.Scan(&regs)
	err = res.Error
	return
}

func SearchDESJobsByRegion(lngMin, lngMax, latMin, latMax float32) (regs []DESRegistration, err error) {

	// token = "%" + token + "%"

	qry := DES.DB.
		Table("des_job_searches").
		Select("des_devs.*, des_jobs.*").
		Joins("JOIN des_jobs ON des_job_searches.des_job_key = des_jobs.des_job_id").
		Joins("JOIN des_devs ON des_jobs.des_job_dev_id = des_devs.des_dev_id").
		Where("( des_jobs.des_job_lng BETWEEN ? AND ? ) AND ( des_jobs.des_job_lat BETWEEN ? AND ? ) ", lngMin, lngMax, latMin, latMax)

	res := qry.Scan(&regs)
	err = res.Error
	return
}

func SearchDESJobs(p DESSearchParam) (regs []DESRegistration, err error) {

	p.Token = "%" + p.Token + "%"

	qry := DES.DB.
		Table("des_job_searches").
		Select("des_devs.*, des_jobs.*").
		Joins("JOIN des_jobs ON des_job_searches.des_job_key = des_jobs.des_job_id").
		Joins("JOIN des_devs ON des_jobs.des_job_dev_id = des_devs.des_dev_id").
		Where(`
		( des_job_searches.des_job_token LIKE ? ) AND
		( des_jobs.des_job_lng BETWEEN ? AND ? ) AND 
		( des_jobs.des_job_lat BETWEEN ? AND ? ) AND 
		( des_jobs.des_job_name NOT LIKE '%_CMDARCHIVE' )`,
			p.Token, p.LngMin, p.LngMax, p.LatMin, p.LatMax)

	res := qry.Scan(&regs)
	err = res.Error
	return
}


func SearchDESDevices(p DESSearchParam) (regs []DESRegistration, err error) {

	p.Token = "%" + strings.ToUpper(p.Token) + "%"

	/* WHERE MORE THAN ONE JOB IS ACTIVE ( des_job_end = 0 ) WE WANT THE LATEST */
	subQryLatestJob := DES.DB.
		Table("des_jobs").
		Select("des_job_dev_id, MAX(des_job_reg_time) AS max_time").
		Joins("JOIN des_job_searches ON des_jobs.des_job_id = des_job_searches.des_job_key").
		Where(`des_job_end = 0
		AND UPPER( des_job_searches.des_job_token ) LIKE ?
		AND des_jobs.des_job_lng BETWEEN ? AND ?
		AND des_jobs.des_job_lat BETWEEN ? AND ?
		`,p.Token, p.LngMin, p.LngMax, p.LatMin, p.LatMax ).
		Group("des_job_dev_id")

	qry := DES.DB.
		Table("des_jobs").
		Distinct("des_devs.*, des_jobs.*").
		Joins(`JOIN ( ? ) j ON des_jobs.des_job_dev_id = j.des_job_dev_id AND des_job_reg_time = j.max_time`, subQryLatestJob).
		Joins("JOIN des_devs ON des_devs.des_dev_id = j.des_job_dev_id").
		Order("des_devs.des_dev_serial DESC")


	res := qry.Scan(&regs)
	err = res.Error
	return
}

type DESSearchParam struct {
	Token  string  `json:"token"`
	LngMin float32 `json:"lng_min"`
	LngMax float32 `json:"lng_max"`
	LatMin float32 `json:"lat_min"`
	LatMax float32 `json:"lat_max"`
}

/* TODO: LOOK INTO USING JSON FIELD FOR TOKEN... */
/* https://blog.davidvassallo.me/2022/12/14/inserting-reading-and-updating-json-data-in-postgres-using-golang-gorm/ */
// type JSONB map[string]interface{}
// func (jsonField JSONB) Value() (driver.Value, error) {
// 	return json.Marshal(jsonField)
// }
// func (jsonField *JSONB) Scan(value interface{}) error {
//     data, ok := value.([]byte)
//     if !ok {
//         return errors.New("type assertion to []byte failed")
//     }
//     return json.Unmarshal(data,&jsonField )
// }

// type DESJobSearchJSON struct {
// 	DESJobSearchID int64  `gorm:"unique; primaryKey" json:"des_job_search_id"`
// 	DESJobToken    JSONB `gorm:"type:jsonb" json:"des_job_token"`
// 	DESJobKey      int64  `json:"des_job_key"`
// 	DESJob         DESJob `gorm:"foreignKey:DESJobKey; references:des_job_id" json:"-"`
// }

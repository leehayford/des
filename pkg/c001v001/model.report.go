package c001v001

import (
	"fmt"
)

/*
REPORT - AS WRITTEN TO DATABASE
*/

type Report struct {
	RepID    int64  `gorm:"unique; primaryKey" json:"rep_id"`
	RepUserID string `gorm:"not null; varchar(36)" json:"rep_user_id"`
	RepCreated   int64  `gorm:"not null" json:"rep_created"`
	RepModified   int64  `gorm:"not null" json:"rep_modified"`

	RepJobID int64  `json:"rep_job_id"` // DES Job ID
	RepTitle string `json:"rep_title"`

	RepSecs []RepSection `json:"rep_secs"`
}
func (rep *Report) CreateReport(job *Job) {
	if rep.RepTitle == "" {
		rep.RepTitle = fmt.Sprintf("%s-%d", job.DESDevSerial, job.DESJobEnd)
		// rep.RepTitle = job.Headers[len(job.Headers)-1].HdrWellName
	}
	/* WRITE TO JOB DB ( ??? WRITE TO DES DB ??? ) */
	job.JDB().Create(rep)
}


type RepSection struct {
	SecID    int64  `gorm:"unique; primaryKey" json:"sec_id"`
	SecUserID string `gorm:"not null; varchar(36)" json:"sec_user_id"`
	SecCreated   int64  `gorm:"not null" json:"sec_created"`
	SecModified   int64  `gorm:"not null" json:"sec_modified"`

	SecRepID int64  `json:"sec_rep_id"`
	SecStart int64  `gorm:"not null" json:"sec_start"`
	SecEnd   int64  `gorm:"not null" json:"sec_end"`
	SecName  string `json:"sec_name"`

	SecDats []SecDataset `json:"sec_dats"`
}
func (sec *RepSection) CreateRepSection(rep *Report, start, end int64, name string) {	
	
	sec.SecRepID = rep.RepID
	/* WRITE TO JOB DB ( ??? WRITE TO DES DB ??? ) */
}

type SecDataset struct {
	DatID    int64   `gorm:"unique; primaryKey" json:"dat_id"`
	DatUserID string `gorm:"not null; varchar(36)" json:"dat_user_id"`
	DatCreated   int64  `gorm:"not null" json:"dat_created"`
	DatModified   int64  `gorm:"not null" json:"dat_modified"`

	DatSecID int64   `json:"dat_sec_id"`
	DatCSV   bool    `json:"dat_csv"`
	DatPlot  bool    `json:"dat_plot"`
	DatYAxis string  `json:"dat_y_axis"`
	DatYMin  float64 `json:"dat_y_min"`
	DatYMax  float64 `json:"dat_y_max"`
}
func (dat *SecDataset) CreateSecDataset(sec *RepSection) {

}

type SecAnnotation struct {
	AnnID int64 `gorm:"unique; primaryKey" json:"ann_id"`
	AnnUserID string `gorm:"not null; varchar(36)" json:"ann_user_id"`
	AnnCreated   int64  `gorm:"not null" json:"ann_created"`
	AnnModified   int64  `gorm:"not null" json:"ann_modified"`
	
	AnnSecID int64   `json:"ann_sec_id"`
	AnnCSV   bool    `json:"ann_csv"`
	AnnPlot  bool    `json:"ann_plot"`

	AnnEvtID int64	`json:"ann_evt_id"` 
	AnnEvt Event `gorm:"foreignKey:EvtID; references:Ann_evt_id" json:"evt"`
}
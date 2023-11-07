package c001v001

import (
	"fmt"
	"time"

	"github.com/leehayford/des/pkg"
)

/*
REPORT - AS WRITTEN TO DATABASE
*/
type Report struct {
	RepID    int64  `gorm:"unique; primaryKey" json:"rep_id"`
	RepUserID string `gorm:"not null; varchar(36)" json:"rep_user_id"`
	RepCreated   int64  `gorm:"autoCreateTime:milli" json:"rep_created"`
	RepModified   int64  `gorm:"autoUpdateTime:milli" json:"rep_modified"`

	RepTitle string `json:"rep_title"`
	RepSecs []RepSection `gorm:"foreignKey:SecID" json:"rep_secs"`

	// RepJobID int64  `json:"rep_job_id"` // DES Job ID
	pkg.DESRegistration `gorm:"-" json:"reg"`
}
func (rep *Report) CreateReport(job *Job) {
	if rep.RepTitle == "" {
		rep.RepTitle = fmt.Sprintf("%s-%d", job.DESJobName, time.Now().UTC().UnixMilli())
	}

	/* WRITE TO JOB DB ( ??? WRITE TO DES DB ??? ) */
	db := job.JDB()
	db.Connect()
	defer db.Disconnect()
	db.Create(rep)

}

/* CALC STRUCTURE - NOT WRITTEN TO DB */
type SecScales struct {
	MinCH4 float32 `json:"min_ch4"`
	MaxCH4 float32 `json:"max_ch4"`
	
	MinHF float32 `json:"min_hf"`
	MaxHF float32 `json:"max_hf"`
	
	MinLF float32 `json:"min_lf"`
	MaxLF float32 `json:"max_lf"`

	MinPress float32 `json:"min_press"`
	MaxPress float32 `json:"max_press"`

	MinBatAmp float32  `json:"min_bat_amp"`
	MaxBatAmp float32  `json:"max_bat_amp"`

	MinBatVolt float32  `json:"min_bat_volt"`
	MaxBatVolt float32  `json:"max_bat_volt"`

	MinMotVolt float32  `json:"min_mot_volt"`
	MaxMotVolt float32  `json:"max_mot_volt"`
}
func (scls *SecScales) AutoScaleSection(job *Job, start, end int64) (err error) {

	/* GET MIN / MAX FOR EACH VALUE IN THE SECTION */
	db := job.JDB()
	db.Connect()
	defer db.Disconnect()
	qry := db.
		Table("samples").
		Select(`
			MIN(smp_ch4) min_ch4, 
			MAX(smp_ch4) max_ch4, 

			MIN(smp_hi_flow) min_hf, 
			MAX(smp_hi_flow) max_hf, 

			MIN(smp_lo_flow) min_lf, 
			MAX(smp_lo_flow) max_lf, 
 
			MIN(smp_press) min_press,
			MAX(smp_press) max_press
			`).
		Where(`smp_time >= ? AND smp_time <= ?`, start, end) 
	
		res := qry.Scan(&scls)
		db.Disconnect()
		// pkg.Json("AutoScaleSection(): SecScales", scls)
		err = res.Error
		if err != nil {
			return 
		}

		/* ADD A MARGIN TO THE QUERIED RESULTS */
		m := float32(0.1)
		margin := ( scls.MaxCH4 - scls.MinCH4 ) * m
		scls.MinCH4 -= margin
		scls.MaxCH4 += margin

		margin = ( scls.MaxHF - scls.MinHF ) *m
		scls.MinHF -= margin
		scls.MaxHF += margin

		margin = ( scls.MaxLF - scls.MinLF ) *m
		scls.MinLF -= margin
		scls.MaxLF += margin

		margin = ( scls.MaxPress - scls.MinPress ) *m
		scls.MinPress -= margin
		scls.MaxPress += margin

		/* FIXED SCALES ( FOR THIS PURPOSE ) */
		scls.MinBatAmp = 0
		scls.MaxBatAmp = 1.5

		scls.MinBatVolt = 0
		scls.MaxBatVolt = 15

		scls.MinMotVolt = 0
		scls.MaxMotVolt = 15

		// pkg.Json("AutoScaleSection(): SecScales -> with margin ", scls)

		return
}

type RepSection struct {
	SecID    int64  `gorm:"unique; primaryKey" json:"sec_id"`
	SecUserID string `gorm:"not null; varchar(36)" json:"sec_user_id"`
	SecCreated   int64  `gorm:"autoCreateTime:milli" json:"sec_created"`
	SecModified   int64  `gorm:"autoUpdateTime:milli" json:"sec_modified"`

	SecRepID int64  `json:"sec_rep_id"`
	SecStart int64  `gorm:"not null" json:"sec_start"`
	SecEnd   int64  `gorm:"not null" json:"sec_end"`
	SecName  string `json:"sec_name"`

	SecDats []SecDataset `gorm:"foreignKey:DatID" json:"sec_dats"`
	SecAnns []SecAnnotation `gorm:"foreignKey:AnnID" json:"sec_anns"`
}
func (job *Job) CreateRepSection(rep *Report, start, end int64, name string) (sec *RepSection, err error){	
	
	sec.SecRepID = rep.RepID
	sec.SecUserID = rep.RepUserID

	return
}

type SecDataset struct {
	DatID    int64   `gorm:"unique; primaryKey" json:"dat_id"`
	DatUserID string `gorm:"not null; varchar(36)" json:"dat_user_id"`
	DatCreated   int64  `gorm:"autoCreateTime:milli" json:"dat_created"`
	DatModified   int64  `gorm:"autoUpdateTime:milli" json:"dat_modified"`

	DatSecID int64   `json:"dat_sec_id"`
	DatCSV   bool    `json:"dat_csv"`
	DatPlot  bool    `json:"dat_plot"`
	DatYAxis string  `json:"dat_y_axis"`
	DatYMin  float32 `json:"dat_y_min"`
	DatYMax  float32 `json:"dat_y_max"`
}
func (job *Job) CreateSecDataset(sec *RepSection, ymin, ymax float32) (dat *SecDataset, err error) {

	dat.DatSecID = sec.SecID
	
	return
}

type SecAnnotation struct {
	AnnID int64 `gorm:"unique; primaryKey" json:"ann_id"`
	AnnUserID string `gorm:"not null; varchar(36)" json:"ann_user_id"`
	AnnCreated   int64  `gorm:"autoCreateTime:milli" json:"ann_created"`
	AnnModified   int64  `gorm:"autoUpdateTime:milli" json:"ann_modified"`
	
	AnnSecID int64   `json:"ann_sec_id"`
	AnnCSV   bool    `json:"ann_csv"`
	AnnPlot  bool    `json:"ann_plot"`

	AnnEvtID int64	`json:"ann_evt_id"` 
	AnnEvt Event `gorm:"foreignKey:AnnEvtID; references:EvtID" json:"evt"`
}
func (job *Job) CreateSecAnnotation(sec *RepSection) (ann *SecAnnotation, err error) {

	ann.AnnSecID = sec.SecID

	return
}
package c001v001

import (
	"github.com/leehayford/des/pkg"
)

/*
REPORT - AS WRITTEN TO DATABASE
*/
type Report struct {
	// RepID   int64  `gorm:"unique; primaryKey" json:"rep_id"` // POSTGRES
	RepID       int64  `gorm:"autoIncrement" json:"rep_id"` // SQLITE
	RepUserID   string `gorm:"not null; varchar(36)" json:"rep_user_id"`
	RepCreated  int64  `gorm:"autoCreateTime:milli" json:"rep_created"`
	RepModified int64  `gorm:"autoUpdateTime:milli" json:"rep_modified"`

	RepTitle string `json:"rep_title"`
	// RepSecs  []RepSection `gorm:"foreignKey:SecRepID" json:"rep_secs"` // POSTGRESS
	RepSecs []RepSection `gorm:"foreignKey:SecRepID; references:RepID" json:"rep_secs"`

	pkg.DESRegistration `gorm:"-" json:"reg"`
}

type RepSection struct {
	// SecID   int64  `gorm:"unique; primaryKey" json:"sec_id"` // POSTGRES
	SecID       int64  `gorm:"autoIncrement" json:"sec_id"` // SQLITE
	SecUserID   string `gorm:"not null; varchar(36)" json:"sec_user_id"`
	SecCreated  int64  `gorm:"autoCreateTime:milli" json:"sec_created"`
	SecModified int64  `gorm:"autoUpdateTime:milli" json:"sec_modified"`

	SecRepID int64  `json:"sec_rep_id"`
	SecStart int64  `gorm:"not null" json:"sec_start"`
	SecEnd   int64  `gorm:"not null" json:"sec_end"`
	SecName  string `json:"sec_name"`

	// SecDats []SecDataset `gorm:"foreignKey:DatSecID" json:"sec_dats"` // POSTGRES
	// SecAnns []SecAnnotation `gorm:"foreignKey:AnnSecID" json:"sec_anns"` // POSTGRES
	SecDats []SecDataset    `gorm:"foreignKey:DatSecID; references:SecID" json:"sec_dats"`
	SecAnns []SecAnnotation `gorm:"foreignKey:AnnSecID; references:SecID" json:"sec_anns"`
}

type SecDataset struct {
	// DatID   int64  `gorm:"unique; primaryKey" json:"dat_id"` // POSTGRES
	DatID       int64  `gorm:"autoIncrement" json:"dat_id"` // SQLITE
	DatUserID   string `gorm:"not null; varchar(36)" json:"dat_user_id"`
	DatCreated  int64  `gorm:"autoCreateTime:milli" json:"dat_created"`
	DatModified int64  `gorm:"autoUpdateTime:milli" json:"dat_modified"`

	DatSecID int64   `json:"dat_sec_id"`
	DatCSV   bool    `json:"dat_csv"`
	DatPlot  bool    `json:"dat_plot"`
	DatYAxis string  `json:"dat_y_axis"`
	DatYMin  float32 `json:"dat_y_min"`
	DatYMax  float32 `json:"dat_y_max"`
}

type SecAnnotation struct {
	// AnnID   int64  `gorm:"unique; primaryKey" json:"ann_id"` // POSTGRES
	AnnID       int64  `gorm:"autoIncrement" json:"ann_id"` // SQLITE
	AnnUserID   string `gorm:"not null; varchar(36)" json:"ann_user_id"`
	AnnCreated  int64  `gorm:"autoCreateTime:milli" json:"ann_created"`
	AnnModified int64  `gorm:"autoUpdateTime:milli" json:"ann_modified"`

	AnnSecID int64 `json:"ann_sec_id"`
	AnnCSV   bool  `json:"ann_csv"`
	AnnPlot  bool  `json:"ann_plot"`

	AnnEvtFK int64 `json:"ann_evt_id"`
	AnnEvt   Event `gorm:"foreignKey:AnnEvtFK; references:EvtID" json:"evt"`
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

	MinBatAmp float32 `json:"min_bat_amp"`
	MaxBatAmp float32 `json:"max_bat_amp"`

	MinBatVolt float32 `json:"min_bat_volt"`
	MaxBatVolt float32 `json:"max_bat_volt"`

	MinMotVolt float32 `json:"min_mot_volt"`
	MaxMotVolt float32 `json:"max_mot_volt"`
}


// type StableFlowAnnotation struct {
//  	// SfaID   int64  `gorm:"unique; primaryKey" json:"sfa_id"` // POSTGRES
// 	SfaID    int64   `gorm:"autoIncrement" json:"sfa_id"` // SQLITE
// 	SfaUserID string `gorm:"not null; varchar(36)" json:"sfa_user_id"`
// 	SfaCreated   int64  `gorm:"autoCreateTime:milli" json:"sfa_created"`
// 	SfaModified   int64  `gorm:"autoUpdateTime:milli" json:"sfa_modified"`

// 	SfaStart int64  `gorm:"not null" json:"sfa_start"`
// 	SfaEnd   int64  `gorm:"not null" json:"sfa_end"`
// 	SfaDur   int64  `gorm:"not null" json:"sfa_dur"` // min
// 	SfaTAC float32 `json:"sfa_tac"` // Total Annual metric ton (1000 kg)

// 	SfaYAxis string  `json:"sfa_y_axis"`

// 	SfaCSV   bool    `json:"ann_csv"`
// 	SfaPlot  bool    `json:"ann_plot"`
// 	SfaSecID int64   `json:"sfa_sec_id"`
// 	SfaSec RepSection `gorm:"foreignKey:SfaSecID; references:SecID" json:"sec"`

// 	SfaEvtID int64   `json:"sfa_evt_id"`
// 	SfaEvt Event `gorm:"foreignKey:SfaEvtID; references:EvtID" json:"evt"`
// }

package c001v001

import (
	"github.com/leehayford/des/pkg"
)

/*
CONFIG - AS WRITTEN TO JOB DATABASE
*/
type Config struct {
	CfgID int64 `gorm:"unique; primaryKey" json:"cfg_id"`

	CfgTime   int64  `gorm:"not null" json:"cfg_time"`
	CfgAddr   string `json:"cfg_addr"`
	CfgUserID string `gorm:"not null; varchar(36)" json:"cfg_user_id"`
	CfgApp    string `gorm:"not null; varchar(36)" json:"cfg_app"`

	/*JOB*/
	// CfgJobName  string  `gorm:"not null; unique; varchar(24)" json:"cfg_job_name"`
	// CfgJobStart int64   `json:"cfg_job_start"`
	// CfgJobEnd   int64   `json:"cfg_job_end"`
	CfgSCVD     float32 `json:"cfg_scvd"`
	CfgSCVDMult float32 `json:"cfg_scvd_mult"`
	CfgSSPRate  float32 `json:"cfg_ssp_rate"`
	CfgSSPDur   float32 `json:"cfg_ssp_dur"`
	CfgHiSCVF   float32 `json:"cfg_hi_scvf"`
	CfgFlowTog  float32 `json:"cfg_flow_tog"`

	/*VALVE*/
	CfgVlvTgt int32 `json:"cfg_vlv_tgt"`
	CfgVlvPos int32 `json:"cfg_vlv_pos"`

	/*OP PERIODS*/
	CfgOpSample int32 `json:"cfg_op_sample"`
	CfgOpLog    int32 `json:"cfg_op_log"`
	CfgOpTrans  int32 `json:"cfg_op_trans"`

	/*DIAG PERIODS*/
	CfgDiagSample int32 `json:"cfg_diag_sample"`
	CfgDiagLog    int32 `json:"cfg_diag_log"`
	CfgDiagTrans  int32 `json:"cfg_diag_trans"`
}

/*
CONFIG - MQTT MESSAGE STRUCTURE
*/
type MQTT_JobConfig struct {
	CfgTime   int64  `json:"cfg_time"`
	CfgAddr   string `json:"cfg_addr"`
	CfgUserID string `json:"cfg_user_id"`
	CfgApp    string `json:"cfg_app"`

	/*JOB*/
	CfgSCVD     float32 `json:"cfg_scvd"`
	CfgSCVDMult float32 `json:"cfg_scvd_mult"`
	CfgSSPRate  float32 `json:"cfg_ssp_rate"`
	CfgSSPDur   float32 `json:"cfg_ssp_dur"`
	CfgHiSCVF   float32 `json:"cfg_hi_scvf"`
	CfgFlowTog  float32 `json:"cfg_flow_tog"`

	/*VALVE*/
	CfgVlvTgt int32 `json:"cfg_vlv_tgt"`
	CfgVlvPos int32 `json:"cfg_vlv_pos"`

	/*OP PERIODS*/
	CfgOpSample int32 `json:"cfg_op_sample"`
	CfgOpLog    int32 `json:"cfg_op_log"`
	CfgOpTrans  int32 `json:"cfg_op_trans"`

	/*DIAG PERIODS*/
	CfgDiagSample int32 `json:"cfg_diag_sample"`
	CfgDiagLog    int32 `json:"cfg_diag_log"`
	CfgDiagTrans  int32 `json:"cfg_diag_trans"`
}

func (cfg *Config) FilterCfgRecord() MQTT_JobConfig {
	return MQTT_JobConfig{
		CfgTime:   cfg.CfgTime,
		CfgAddr:   cfg.CfgAddr,
		CfgUserID: cfg.CfgUserID,
		CfgApp:    cfg.CfgApp,

		CfgSCVD:     cfg.CfgSCVD,
		CfgSCVDMult: cfg.CfgSCVDMult,
		CfgSSPRate:  cfg.CfgSSPRate,
		CfgSSPDur:   cfg.CfgSSPDur,
		CfgHiSCVF:   cfg.CfgHiSCVF,
		CfgFlowTog:  cfg.CfgFlowTog,

		CfgVlvTgt: cfg.CfgVlvTgt,
		CfgVlvPos: cfg.CfgVlvPos,

		CfgOpSample: cfg.CfgOpSample,
		CfgOpLog:    cfg.CfgOpLog,
		CfgOpTrans:  cfg.CfgOpTrans,

		CfgDiagSample: cfg.CfgDiagSample,
		CfgDiagLog:    cfg.CfgDiagLog,
		CfgDiagTrans:  cfg.CfgDiagTrans,
	}
}

func (cfg *Config) FilterCfgBytes() (out []byte) {

	out = append(out, pkg.Int64ToBytes(cfg.CfgTime)...)
	out = append(out, pkg.StringToNBytes(cfg.CfgAddr, 36)...)
	out = append(out, pkg.StringToNBytes(cfg.CfgUserID, 36)...)
	out = append(out, pkg.StringToNBytes(cfg.CfgApp, 36)...)

	out = append(out, pkg.Float32ToBytes(cfg.CfgSCVD)...)
	out = append(out, pkg.Float32ToBytes(cfg.CfgSCVDMult)...)
	out = append(out, pkg.Float32ToBytes(cfg.CfgSSPRate)...)
	out = append(out, pkg.Float32ToBytes(cfg.CfgSSPDur)...)
	out = append(out, pkg.Float32ToBytes(cfg.CfgHiSCVF)...)
	out = append(out, pkg.Float32ToBytes(cfg.CfgFlowTog)...)

	out = append(out, pkg.Int32ToBytes(cfg.CfgVlvTgt)...)
	out = append(out, pkg.Int32ToBytes(cfg.CfgVlvPos)...)

	out = append(out, pkg.Int32ToBytes(cfg.CfgOpSample)...)
	out = append(out, pkg.Int32ToBytes(cfg.CfgOpLog)...)
	out = append(out, pkg.Int32ToBytes(cfg.CfgOpTrans)...)
	
	out = append(out, pkg.Int32ToBytes(cfg.CfgDiagSample)...)
	out = append(out, pkg.Int32ToBytes(cfg.CfgDiagLog)...)
	out = append(out, pkg.Int32ToBytes(cfg.CfgDiagTrans)...)

	return
}

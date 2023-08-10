package c001v001

import (
	"fmt"
	"strings"
	// "time"

	// "github.com/gofiber/fiber/v2"

	"github.com/leehayford/des/pkg"
)
// 28b118b6-14d0-4404-914a-7ded89d1e5c6
type Job struct {
	pkg.DESJob
}

func (job *Job) JDB() *pkg.DBI {
	return &pkg.DBI{ ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(job.DESJobName)), }
}

type JobAdmin struct {
	AdmID int64 `gorm:"unique; primaryKey" json:"adm_id"`
}
type JobConfig struct {
	CfgID int64 `gorm:"unique; primaryKey" json:"cfg_id"`

	CfgTime   int64  `gorm:"not null" json:"cfg_time"`
	CfgAddr   string `json:"cfg_addr"`
	CfgUserID string `gorm:"not null; varchar(36)" json:"cfg_user_id"`
	CfgApp    string `gorm:"not null; varchar(36)" json:"cfg_app"`

	/*JOB*/
	CfgJobName  string  `json:"cfg_job_name"`
	CfgJobStart int64   `json:"cfg_job_start"`
	CfgJobEnd   int64   `json:"cfg_job_end"`
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
type JobEvent struct {
	EvtID int64 `gorm:"unique; primaryKey" json:"evt_id"`
}
type JobEventType struct {
	ETypeID int64 `gorm:"unique; primaryKey" json:"etype_id"`
}
type JobSample struct {
	SmpID int64 `gorm:"unique; primaryKey" json:"smp_id"`
}
package c001v001

import (
	"fmt"
	"strings"
	// "time"

	// "github.com/gofiber/fiber/v2"

	"github.com/leehayford/des/pkg"
	desModels "github.com/leehayford/des/pkg/models"
)

type Job struct {
	desModels.DESJob
}

func (job *Job) JDB() *pkg.DBI {
	return &pkg.DBI{ ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(job.DESJobName)), }
}

type JobAdmin struct {
	AdmID int64 `gorm:"unique; primaryKey" json:"adm_id"`
}
type JobConfig struct {
	CfgID int64 `gorm:"unique; primaryKey" json:"cfg_id"`
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
package c001v001

import (
	"fmt"
	"strings"

	// "time"

	// "github.com/gofiber/fiber/v2"

	"github.com/leehayford/des/pkg"
)

type Job struct {
	Admins  []JobAdmin  `json:"admins"`
	Configs []JobConfig `json:"configs"`
	Events  []Evt       `json:"events"`
	Samples []JobSample `json:"samples"`
	pkg.DESRegistration
}

func (job *Job) JDB() *pkg.DBI {
	return &pkg.DBI{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(job.DESJobName))}
}

type JobSample struct {
	SmpID int64 `gorm:"unique; primaryKey" json:"smp_id"`
}

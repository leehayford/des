package c001v001

import (
	"fmt"
	"strings"

	"github.com/leehayford/des/pkg"
)

type Job struct {
	Admins  []Admin     `json:"admins"`
	Configs []Config    `json:"configs"`
	Events  []Event     `json:"events"`
	Samples []Sample `json:"samples"`
	pkg.DESRegistration `json:"reg"`
}

func (job *Job) JDB() *pkg.DBI {
	return &pkg.DBI{ConnStr: fmt.Sprintf("%s%s", pkg.DB_SERVER, strings.ToLower(job.DESJobName))}
}


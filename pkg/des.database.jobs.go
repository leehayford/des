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
	"fmt"
	"os"
	"strings"

	/* https://gorm.io/docs/ */
	"gorm.io/gorm" // go get gorm.io/gorm
	// "github.com/glebarez/sqlite" // go get github.com/glebarez/sqlite
	"gorm.io/driver/sqlite" // go get gorm.io/driver/sqlite
	"gorm.io/gorm/logger"
)

type JobDBClient struct {
	ConnStr string
	*gorm.DB

	/* TODO: ADD RWMUTEXT */
}

func CheckDatabaseExists(db_name string) (exits bool) {
	_, err := os.Stat(fmt.Sprintf("%s/%s", DES_JOB_DATABASES, db_name))
	if !os.IsNotExist(err) {
		exits = true
	}
	return
}
func MakeDBClient(db_name string) (dbc JobDBClient) {
	dbc = JobDBClient{ConnStr: fmt.Sprintf("%s/%s", DES_JOB_DATABASES, db_name)}
	return
}

func (jdbc *JobDBClient) GetDBName() string {
	str := strings.Split(jdbc.ConnStr, "/")
	if len(str) == 2 {
		/* THIS IS A VALID CONNECTION STRING */
		return str[1] /* TODO: IMPROVE VALIDATION ? */
	} else {
		return ""
	}
}
func (jdbc *JobDBClient) Connect() (err error) {

	if jdbc.DB, err = gorm.Open(sqlite.Open(jdbc.ConnStr), &gorm.Config{}); err != nil {
		// fmt.Printf("\n(*JobDBClient) Connect() -> %s -> FAILED! \n", jdbc.GetDBName())
		return LogErr(err)
	}
	// jdbc.DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
	jdbc.DB.Logger = logger.Default.LogMode(logger.Error)

	// fmt.Printf("\n(*JobDBClient) Connect() -> %s -> connected... \n", jdbc.GetDBName())
	return err
}
func (jdbc *JobDBClient) Disconnect() (err error) {

	db, err := jdbc.DB.DB()
	if err != nil {
		return LogErr(err)
	}
	if err = db.Close(); err != nil {
		return LogErr(err)
	}
	// fmt.Printf("\n(*JobDBClient) Disconnect() -> %s -> connection closed. \n", jdbc.GetDBName())
	jdbc = &JobDBClient{}
	return
}

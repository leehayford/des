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

func GetJobDBClient(db_name string) (dbc JobDBClient, err error) {
	dbc = JobDBClient{ConnStr: fmt.Sprintf("%s/%s", DES_JOB_DATABASES, db_name)}
	err = dbc.ConfirmDBFile()
	return
}
func (jdbc *JobDBClient) ConfirmDBFile() (err error) {
	/* WE AVOID CREATING IF THE DATABASE WAS PRE-EXISTING, LOG TO CMDARCHIVE  */
	_, err = os.Stat(fmt.Sprintf(jdbc.ConnStr))
	if os.IsNotExist(err) {
		f, os_err := os.Create(jdbc.ConnStr)
		if os_err != nil {
			return os_err
		}
		f.Close()
		err = nil
	}
	return
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
func (jdbc *JobDBClient) GetDBNameFromConnStr() string {
	str := strings.Split(jdbc.ConnStr, "/")
	if len(str) == 2 {
		/* THIS IS A VALID CONNECTION STRING */
		return str[1] /* TODO: IMPROVE VALIDATION ? */
	} else {
		return ""
	}
}

// func CheckDatabaseExists(connStr string) (exits bool) {
// 	// _, err := os.Stat(fmt.Sprintf("%s/%s", DES_JOB_DATABASES, db_name))
// 	_, err := os.Stat(fmt.Sprintf(connStr))
// 	if !os.IsNotExist(err) {
// 		exits = true
// 	}
// 	return
// }

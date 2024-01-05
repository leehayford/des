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
	// "os"
	"strings"
	"sync"

	/* https://gorm.io/docs/ */
	"gorm.io/gorm" // go get gorm.io/gorm
	// "github.com/glebarez/sqlite" // go get github.com/glebarez/sqlite
	"gorm.io/driver/sqlite" // go get gorm.io/driver/sqlite
	"gorm.io/gorm/logger"
)

type JobDBClient struct {
	ConnStr string
	*gorm.DB

	RWM *sync.RWMutex
}

func GetJobDBClient(db_name string) (jdbc JobDBClient, err error) {
	jdbc = JobDBClient{
		ConnStr: fmt.Sprintf("%s/%s/%s", DATA_DIR, JOB_DB_DIR, db_name),
		RWM: &sync.RWMutex{},
	}
	return
}

func (jdbc *JobDBClient) Connect() (err error) {

	if jdbc.ConnStr == "" {
		return fmt.Errorf("JobDBClient connection string is empty")
	}

	/* THIS JobDBClient SHOULD ALREADY HAVE A RWMutex 
	BUT WE'LL JUST MAKE SURE BEFORE TRYING TO LOCK IT */
	if jdbc.RWM == nil {
		jdbc.RWM = &sync.RWMutex{}
	}

	jdbc.RWM.Lock()
	if jdbc.DB, err = gorm.Open(sqlite.Open(jdbc.ConnStr), &gorm.Config{}); err != nil {
		// fmt.Printf("\n(*JobDBClient) Connect() -> %s -> FAILED! \n", jdbc.GetDBName())
		return LogErr(err)
	}
	// jdbc.DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
	jdbc.DB.Logger = logger.Default.LogMode(logger.Error)
	jdbc.RWM.Unlock()

	// fmt.Printf("\n(*JobDBClient) Connect() -> %s -> connected... \n", jdbc.GetDBName())
	return err
}
func (jdbc *JobDBClient) Disconnect() (err error) {

	/* THIS JobDBClient SHOULD ALREADY HAVE A RWMutex 
	BUT WE'LL JUST MAKE SURE BEFORE TRYING TO LOCK IT */
	if jdbc.RWM != nil {
		jdbc.RWM.Lock()
	}
	
	db, err := jdbc.DB.DB()
	if err != nil {
		return LogErr(err)
	}
	if err = db.Close(); err != nil {
		return LogErr(err)
	}
	// fmt.Printf("\n(*JobDBClient) Disconnect() -> %s -> connection closed. \n", jdbc.GetDBName())

	/*  WE DON'T UNLOCK BECAUSE THE OLD RWMutex IS GONE, 
	ALONG WITH ANY PENDING DB OPERATIONS  */
	jdbc = &JobDBClient{RWM: &sync.RWMutex{}}

	return
}
func (jdbc *JobDBClient) GetDBNameFromConnStr() string {
	str := strings.Split(jdbc.ConnStr, "/")
	if len(str) == 3 {
		/* THIS IS A VALID CONNECTION STRING */
		return str[2] /* TODO: IMPROVE VALIDATION ? */
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

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
	// "encoding/json"
	"fmt"
	"time"

	// "log"
	"os"
	"strings"
	"sync"

	// "time"

	/* https://gorm.io/docs/ */
	"golang.org/x/crypto/bcrypt" // go get golang.org/x/crypto/bcrypt

	"gorm.io/driver/postgres" // go get gorm.io/driver/postgres
	"gorm.io/gorm"            // go get gorm.io/gorm
	"gorm.io/gorm/logger"
)

/* TYPE DEFINITION TO ALLOW GORM TO WORK WITH JSONB DATA */
// type JSONB map[string]interface{}

/*
	DATABASE CLIENT

ALL DATABASES IN THE DES ARE ACCESSED VIA A DBClient
*/
type DBClient struct {
	ConnStr string
	*gorm.DB

	/* TODO: FINISH IMPLEMENTATION & TESTING
	WAIT GROUP USED TO PREVENT CONCURRENT ACCESS  OF MAPPED DEVICE STATE */
	WG *sync.WaitGroup
}

func (dbc *DBClient) GetDBName() string {
	str := strings.Split(dbc.ConnStr, "/")
	if len(str) == 4 {
		/* THIS IS A VALID CONNECTION STRING */
		return str[3] /* TODO: IMPROVE VALIDATION ? */
	} else {
		return ""
	}
}

func (dbc *DBClient) Connect( /* TODO: CONNECTION POOL OPTIONS */ ) (err error) {

	if dbc.DB, err = gorm.Open(postgres.Open(dbc.ConnStr), &gorm.Config{}); err != nil {
		// fmt.Printf("\n(dbc *DBClient) Connect() -> %s -> FAILED! \n", dbc.GetDBName())
		return LogErr(err)
	}
	dbc.DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
	dbc.DB.Logger = logger.Default.LogMode(logger.Error)
	dbc.WG = &sync.WaitGroup{}

	// fmt.Printf("\n(dbc *DBClient) Connect() -> %s -> connected... \n", dbc.GetDBName())
	return err
}
func (dbc *DBClient) Disconnect() (err error) {

	/* ENSURE ALL PENDING WRITES TO JOB DB ARE COMPLETE BEFORE DISCONNECTION */
	// fmt.Printf("\n(dbc *DBClient) Disconnect() -> %s -> waiting for final write ops... \n", dbc.GetDBName())
	dbc.WG.Wait()

	db, err := dbc.DB.DB()
	if err != nil {
		return LogErr(err)
	}
	if err = db.Close(); err != nil {
		return LogErr(err)
	}
	// fmt.Printf("\n(dbc *DBClient) Disconnect() -> %s -> connection closed. \n", dbc.GetDBName())
	dbc = &DBClient{}
	return
}

/*
	ADMIN DATABASE

USED TO MANAGE ALL OTHER DATABASES ON THIS DES
*/
var ADB ADMINDatabase = ADMINDatabase{DBClient: DBClient{ConnStr: ADMIN_DB_CONNECTION_STRING}}

type ADMINDatabase struct{ DBClient }

func (adb ADMINDatabase) CreateDatabase(db_name string) (err error) {
	db_name = strings.ToLower(db_name)
	createDBCommand := fmt.Sprintf(`CREATE DATABASE %s WITH OWNER = datacan 
		ENCODING = 'UTF8' LC_COLLATE = 'C.UTF-8' LC_CTYPE = 'C.UTF-8' TABLESPACE = pg_default CONNECTION LIMIT = -1 IS_TEMPLATE = False;`,
		db_name,
	)
	fmt.Printf("\n(adb *ADMINDatabase) CreateDatabase( ): Creating %s...\n", db_name)
	res := adb.DB.Exec(createDBCommand)
	err = res.Error
	return
}
func (adb ADMINDatabase) CheckDatabaseExists(db_name string) (exists bool) {
	db_name = strings.ToLower(db_name)
	checkExistsCommand := `SELECT EXISTS ( SELECT datname FROM pg_catalog.pg_database WHERE datname=? )`
	adb.DB.Raw(checkExistsCommand, db_name).Scan(&exists)
	return
}
func (adb ADMINDatabase) DropDatabase(db_name string) {
	db_name = strings.ToLower(db_name)
	dropDBCommand := fmt.Sprintf(`DROP DATABASE %s WITH (FORCE)`, db_name)
	adb.DB.Exec(dropDBCommand)
}
func (adb ADMINDatabase) DropAllDatabases() {
	databases := []string{}
	adb.Raw("SELECT datname FROM pg_catalog.pg_database WHERE datdba != 10").Scan(&databases)
	for _, db := range databases {
		fmt.Printf("\nDROPPING: %s\n", db)
		adb.DropDatabase(db)
	}
}

/* CREATE OR MIGRATE DES DATABASE */
func (adb ADMINDatabase) CreateDESDatabase() (err error) {
	exists := adb.CheckDatabaseExists(DES_DB)

	if !exists {
		if err = adb.CreateDatabase(DES_DB); err != nil {
			return LogErr(err)
		}
	}

	/* CREATE TABLES OR MIGRATE */
	DES.Connect()
	defer DES.Disconnect()
	if err = DES.CreateDESTables(exists); err != nil {
		return LogErr(err)
	}

	return
}

/* IF ANY REQURIRED DIRECTORY FAILS TO EXIST, ACTIVELY DISAGREE */
func ConfirmDESDirectories() (err error) {

	/* THIS IS HERE FOR TESTING **************************/
	ents, err := os.ReadDir(DES_JOB_DATABASES)
	if err != nil {
		LogErr(err)
	}
	for _, ent := range ents {
		fmt.Println(ent.Name())
	}
	/********************************************************/

	if err = ConfirmDirectory(DES_JOB_DATABASES); err != nil {
		return LogErr(err)
	}

	if err = ConfirmDirectory(DES_JOB_DATABASES_ARCHIVE); err != nil {
		return LogErr(err)
	}

	if err = ConfirmDirectory(DES_JOB_FILES); err != nil {
		return LogErr(err)
	}

	if err = ConfirmDirectory(DES_JOB_FILES_ARCHIVE); err != nil {
		return LogErr(err)
	}

	if err = ConfirmDirectory(DES_JOB_DATABASES); err != nil {
		return LogErr(err)
	}

	if err = ConfirmDirectory(DES_DEVICE_FILES); err != nil {
		return LogErr(err)
	}

	if err = ConfirmDirectory(DES_DEVICE_FILES_ARCHIVE); err != nil {
		return LogErr(err)
	}

	return
}
func ConfirmDirectory(name string) (err error) {
	_, err = os.Stat(name)
	if os.IsNotExist(err) {
		if err = os.Mkdir(name, os.ModePerm); err != nil {
			return LogErr(err)
		}
	}
	
	fmt.Printf("ConfirmDirectory( %s ): CONFIRMED\n", name)
	return
}

/* MOVE ALL EXISTING JOB / DEVICE DATA TO ARCHIVE DIRECTORIES */
func ArchiveDESDirectories() (err error) {

	/* TIME OF ARCHIVING ALL EXISTING JOB / DEVICE DATA */
	arc_time := time.Now().UTC().UnixMilli()

	/* ARCHIVE ALL EXISTING JOB DATABASES */
	jdba := fmt.Sprintf("%s/%d", DES_JOB_DATABASES_ARCHIVE, arc_time)
	if err := ArchiveDirectory(DES_JOB_DATABASES, jdba); err != nil {
		return LogErr(err)
	}

	/* ARCHIVE ALL EXISTING JOB FILEES */
	jfa := fmt.Sprintf("%s/%d", DES_JOB_FILES_ARCHIVE, arc_time)
	if err := ArchiveDirectory(DES_JOB_FILES, jfa); err != nil {
		return LogErr(err)
	}

	/* ARCHIVE ALL EXISTING DEVICE FILES */
	dfa := fmt.Sprintf("%s/%d", DES_DEVICE_FILES_ARCHIVE, arc_time)
	if err := ArchiveDirectory(DES_DEVICE_FILES, dfa); err != nil {
		return LogErr(err)
	}

	/* DEMO -> NOT FOR PRODUCTION */
	if err := os.RemoveAll("demo"); err != nil {
		return LogErr(err)
	}
	return
}
func ArchiveDirectory(dir, arc string) (err error) {
	if err := os.Rename(dir, arc); err != nil {
		if strings.Contains(err.Error(), "cannot find the file") {
			/* TODO: IDENTIFY CORRECT os.Err...
			- IT'S NONE OF THESE, AND THAT'S THE WHOLE LIST...
			*/
			// fmt.Printf("%s : %s\n", dir, err.Error())
			// fmt.Printf("%s : %s\n", dir, os.ErrClosed.Error())
			// fmt.Printf("%s : %s\n", dir, os.ErrDeadlineExceeded.Error())
			// fmt.Printf("%s : %s\n", dir, os.ErrExist.Error())
			// fmt.Printf("%s : %s\n", dir, os.ErrInvalid.Error())
			// fmt.Printf("%s : %s\n", dir, os.ErrNoDeadline.Error())
			// fmt.Printf("%s : %s\n", dir, os.ErrNotExist.Error())
			// fmt.Printf("%s : %s\n", dir, os.ErrPermission.Error())
			// fmt.Printf("%s : %s\n", dir, os.ErrProcessDone.Error())
			err = nil
		}

	}
	return
}

/*
	DES DATABASE

USED TO MANAGE DEVICES AND RELATED JOBS ON THIS DES
CURRENTLY MANAGES USERS BUT THIS WILL CHANGE IN THE NEXT VERSION
*/
var DES DESDatabase = DESDatabase{DBClient: DBClient{ConnStr: DES_DB_CONNECTION_STRING}}

type DESDatabase struct{ DBClient }

func (des DESDatabase) CreateDESTables(exists bool) (err error) {

	if exists {
		// fmt.Printf("\nMigrating DES: %s\n", DES.ConnStr)
		err = des.DB.AutoMigrate(
			&User{},
			&DESDev{},
			&DESJob{},
			&DESJobSearch{},
			&DESError{},
		)
	} else {
		// fmt.Printf("\nCreating DES Tables: %s\n", DES.ConnStr)
		if err = des.DB.Migrator().CreateTable(
			&User{},
			&DESDev{},
			&DESJob{},
			&DESJobSearch{},
			&DESError{},
		); err != nil {
			return err
		}

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(SPR_PW), bcrypt.DefaultCost)
		role := ROLE_SUPER
		newUser := User{
			Name:     SPR_USER,
			Email:    strings.ToLower(SPR_EMAIL),
			Password: string(hashedPassword),
			Role:     role,
		}
		if result := des.DB.Create(&newUser); result.Error != nil {
			fmt.Printf("\nCreate admin user failed...\n%s\n", result.Error.Error())
			err = result.Error
		} // Json("(des DESDatabase) CreateDESDatabase(): -> newUser", newUser)
	}

	return err
}
func (des DESDatabase) GetAllTables() (err error) {

	return
}
func (des DESDatabase) GetAllRows() (err error) {

	return
}

// type WGTestThing struct {
// 	Name string
// 	WG *sync.WaitGroup
// }

// func (wgt *WGTestThing) whatever(n int) {
// 	wgt.WG.Add(1)
// 	defer wgt.WG.Done()
// 	time.Sleep(time.Duration(time.Second * 5))
// 	fmt.Printf("\n%s func# %d: done.",wgt.Name , n)
// }

// func (wgt *WGTestThing) waitGroupTest() {

// 	go wgt.whatever(1)
// 	go wgt.whatever(2)
// 	go wgt.whatever(3)

// 	fmt.Printf("\n%s Waiting...", wgt.Name)
// 	wgt.WG.Wait()
// 	fmt.Printf("\n%s Done.\n", wgt.Name)
// }

// type WGTestThingMap map[string]WGTestThing
// var WGTs = make(WGTestThingMap)

// func mappedWaitGroupThingTest() {

// 	WGTs["Jeff"] = *&WGTestThing{Name: "Jeff", WG: &sync.WaitGroup{}}
// 	WGTs["Mia"] = *&WGTestThing{Name: "Mia", WG: &sync.WaitGroup{}}
// 	WGTs["Pat"] = *&WGTestThing{Name: "Pat", WG: &sync.WaitGroup{}}
// 	for name := range WGTs {
// 		wgt := WGTs[name]
// 		go wgt.waitGroupTest()
// 	}
// }

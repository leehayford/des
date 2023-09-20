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
	// "log"
	"os"
	"strings"
	"sync"

	// "time"

	/* https://gorm.io/docs/ */
	"golang.org/x/crypto/bcrypt" // go get golang.org/x/crypto/bcrypt

	"gorm.io/gorm" // go get gorm.io/gorm
	// "gorm.io/plugin/dbresolver" // go get "gorm.io/plugin/dbresolver"
	"gorm.io/driver/postgres" // go get gorm.io/driver/postgres
	"gorm.io/gorm/logger"
)

/*
	DATABASE CLIENT

ALL DATABASES IN THE DES ARE ACCESSED VIA A DBClient
*/
type DBClient struct {
	ConnStr string
	*gorm.DB

	/* WAIT GROUP USED TO ENSURE ALL PENDING WRITES HAVE COMPLETED BEFORE DISCONNECT */
	WG *sync.WaitGroup
}


func (dbc *DBClient) Connect( /* TODO: CONNECTION POOL OPTIONS */ ) (err error) {
	/* TODO: SETUP CONNECTION POOLING FOR DIFERENT TYPES OF CONNECTIONS
			? "gorm.io/plugin/dbresolver" ?

			INITIALLY AIMING FOR 100 ACTIVE DEVICES / DES
			
			ADMIN DB POOL: USED WHEN CREATING NEW JOBS -> TYPICALLY LESS THAN ONE /DAY / DEVICE
					MAX OPEN CONNECTIONS = 50
					MAX IDLE CONNECTIONS = 100

			DES DB: USED WHEN CREATING NEW JOBS -> TYPICALLY LESS THAN ONE /DAY / DEVICE
					MAX OPEN CONNECTIONS = 50
					MAX IDLE CONNECTIONS = 100
			
			DEVICE DB(s): 
				CMDARCHIVE: USED BY DEVICE CLIENT -> A FEW TRANSACTIONS / PER DAY
					MAX OPEN CONNECTIONS < 5
					MAX IDLE CONNECTIONS < 5
				ACTIVE JOB: USED BY DEVICE CLIENT -> ~ ONE TRANSACTION / SECOND
					MAX OPEN CONNECTIONS < 5
					MAX IDLE CONNECTIONS < 5

			JOB DB: USED WHEN USERS ARE CREATING / VIEWING REPORTS/JOB DATA -> A FEW TRANSACTIONS / PER MINUTE
				MAX OPEN CONNECTIONS = 50
				MAX IDLE CONNECTIONS = 100
	*/
	if dbc.DB, err = gorm.Open(postgres.Open(dbc.ConnStr), &gorm.Config{}); err != nil {
		return err
	}
	dbc.DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
	dbc.DB.Logger = logger.Default.LogMode(logger.Error)
	dbc.WG = &sync.WaitGroup{}
	return err
}
func (dbc DBClient) Disconnect() (err error) {

	/* ENSURE ALL PENDING WRITES TO JOB DB ARE COMPLETE BEFORE DISCONNECTION */
	dbc.WG.Wait()

	db, err := dbc.DB.DB()
	if err != nil {
		return TraceErr(err)
	}
	if err = db.Close(); err != nil {
		return TraceErr(err)
	}
	dbc.ConnStr = ""
	dbc.DB = nil
	fmt.Printf("\n (dbc DBClient) Disconnect() -> Connection closed. \n")
	return
}
func (dbc *DBClient) Write(model interface{}) (err error) {
	dbc.WG.Add(1)
	defer dbc.WG.Done()
	res := dbc.Create(model)
	return res.Error
}


/*
	ADMIN DATABASE

USED TO MANAGE ALL OTHER DATABASES ON THIS DES
*/
var ADB ADMINDatabase = ADMINDatabase{DBClient: DBClient{ConnStr: ADMIN_DB_CONNECTION_STRING}}

type ADMINDatabase struct{ DBClient }

func (adb ADMINDatabase) CreateDatabase(db_name string) {
	createDBCommand := fmt.Sprintf(`CREATE DATABASE %s WITH OWNER = datacan 
		ENCODING = 'UTF8' LC_COLLATE = 'C.UTF-8' LC_CTYPE = 'C.UTF-8' TABLESPACE = pg_default CONNECTION LIMIT = -1 IS_TEMPLATE = False;`,
		db_name,
	)
	fmt.Printf("\n(adb *ADMINDatabase) CreateDatabase( ): Creating %s...\n", db_name)
	adb.DB.Exec(createDBCommand)
}
func (adb ADMINDatabase) CheckDatabaseExists(db_name string) (exists bool) {
	checkExistsCommand := `SELECT EXISTS ( SELECT datname FROM pg_catalog.pg_database WHERE datname=? )`
	adb.DB.Raw(checkExistsCommand, db_name).Scan(&exists)
	return
}
func (adb ADMINDatabase) DropDatabase(db_name string) {
	dropDBCommand := fmt.Sprintf(`DROP DATABASE %s WITH (FORCE)`, db_name)
	adb.DB.Exec(dropDBCommand)
}
func (adb ADMINDatabase) DropAllDatabases() {
	databases := &[]string{}
	adb.Raw("SELECT datname FROM pg_catalog.pg_database WHERE datdba != 10").Scan(databases)
	for _, db := range *databases {
		fmt.Printf("\nDROPPING: %s\n", db)
		adb.DropDatabase(db)
	}
	/* DEMO -> NOT FOR PRODUCTION */
	if err := os.RemoveAll("demo"); err != nil {
		TraceErr(err)
	}
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
		)
	} else {
		// fmt.Printf("\nCreating DES Tables: %s\n", DES.ConnStr)
		if err = des.DB.Migrator().CreateTable(
			&User{},
			&DESDev{},
			&DESJob{},
		); err != nil {
			return err
		}

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(ADM_PW), bcrypt.DefaultCost)
		role := string("admin")
		newUser := User{
			Name:     ADM_USER,
			Email:    strings.ToLower(ADM_EMAIL),
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
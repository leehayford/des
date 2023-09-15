
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
	"encoding/json"
	"fmt"
	// "log"
	"os"
	"strings"
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
}

func (dbc *DBClient) Connect() (err error) {
	if dbc.DB, err = gorm.Open(postgres.Open(dbc.ConnStr), &gorm.Config{}); err != nil {
		return err
	}
	dbc.DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
	dbc.DB.Logger = logger.Default.LogMode(logger.Error)
	return err
}
func (dbc DBClient) Close() (err error) {
	db, err := dbc.DB.DB()
	if err != nil {
		return err
	}
	return db.Close()
}
func (dbc *DBClient) Write(model interface{}) (err error) {
	res := dbc.Create(model)
	return res.Error
}
func (dbc *DBClient) WriteMQTT(msg []byte, model interface{}) (err error) {

	if err = json.Unmarshal(msg, model); err != nil {
		return TraceErr(err)
	}
	return dbc.Write(model)
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

func (des DESDatabase)CreateDESTables(exists bool) (err error) {

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

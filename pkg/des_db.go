/* Data Exchange Server (DES) is a component of the Datacan Data2Desk (D2D) Platform.
License:

	[PROPER LEGALESE HERE...]

	INTERIM LICENSE DESCRIPTION:
	In spirit, this license:
	1. Allows <Third Party> to use, modify, adn / or distributre this software in perpetuity so long as <Third Party> understands:
		a. The software is porvided as is without guarantee of additional support from DataCan in any form.
		b. The software is porvided as is without guarantee of exclusivity.

	2. Prohibits <Third Party> from taking any action which might interfere with DataCan's right to use, modify, distributre this software in perpetuity.
*/

package pkg

import (
	"fmt"
	// "log"
	// "os"
	"strings"
	// "time"

	/* https://gorm.io/docs/ */
	"golang.org/x/crypto/bcrypt" // go get golang.org/x/crypto/bcrypt

	"gorm.io/gorm" // go get gorm.io/gorm
	// "gorm.io/plugin/dbresolver" // go get "gorm.io/plugin/dbresolver"
	"gorm.io/driver/postgres" // go get gorm.io/driver/postgres
	"gorm.io/gorm/logger"
)

/* All databases in the DES system have the following structure */
type DBI struct {
	ConnStr string
	*gorm.DB
}

func (dbi *DBI) Connect() (err error) {

	if dbi.DB, err = gorm.Open(postgres.Open(dbi.ConnStr), &gorm.Config{}); err != nil {
		return err
	}

	dbi.DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
	dbi.DB.Logger = logger.Default.LogMode(logger.Error)

	return err
}
func (dbi *DBI) Close() (err error) {
	db, err := dbi.DB.DB()
	if err != nil {
		return err
	}
	return db.Close()
}


/* ADMIN DATABASE */
type ADMINDatabase struct {
	DBI
}
var ADB ADMINDatabase = ADMINDatabase{
	DBI: DBI{ConnStr: ADMIN_DB_CONNECTION_STRING},
}
func (adb *ADMINDatabase) DropDatabase(db_name string) {
	dropDBCommand := fmt.Sprintf(`DROP DATABASE %s WITH (FORCE)`, db_name)
	adb.DB.Exec(dropDBCommand)
}
func(adb *ADMINDatabase) DropAllDatabases() {
	databases := &[]string{}
	adb.Raw("SELECT datname FROM pg_catalog.pg_database WHERE datdba != 10").Scan(databases)
	for _, db := range *databases {
		fmt.Printf("\nDROPPING: %s\n", db)
		adb.DropDatabase(db)
	}
}
func (adb *ADMINDatabase) CreateDatabase(db_name string) {
	createDBCommand := fmt.Sprintf(`CREATE DATABASE %s WITH OWNER = datacan 
		ENCODING = 'UTF8' LC_COLLATE = 'C.UTF-8' LC_CTYPE = 'C.UTF-8' TABLESPACE = pg_default CONNECTION LIMIT = -1 IS_TEMPLATE = False;`,
		db_name,
	)
	fmt.Printf("\n(adb *ADMINDatabase) CreateDatabase( ): Creating %s...\n", db_name)
	adb.DB.Exec(createDBCommand)
}
func (adb *ADMINDatabase) CheckDatabaseExists(db_name string) (exists bool) {
	checkExistsCommand := `SELECT EXISTS ( SELECT datname FROM pg_catalog.pg_database WHERE datname=? )`
	adb.DB.Raw(checkExistsCommand, db_name).Scan(&exists)
	return
}


/* DES DATABASE */
type DESDatabase struct {
	DBI
}
var DES DESDatabase = DESDatabase{
	DBI: DBI{ConnStr: DES_DB_CONNECTION_STRING},
}

func CreateDESDatabase(exists bool) (err error) {

	if exists {
		fmt.Printf("\nMigrating DES: %s\n", DES.ConnStr)
		err = DES.DB.AutoMigrate(
			&User{},
			&DESDev{},
			&DESJob{},
		)
	} else {
		fmt.Printf("\nCreating DES Tables: %s\n", DES.ConnStr)
		if err = DES.DB.Migrator().CreateTable(
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
			// Photo:    &payload.Photo,
		}
		if result := DES.DB.Create(&newUser); result.Error != nil {
			fmt.Printf("\nCreate admin user failed...\n%s\n", result.Error.Error())
			err = result.Error
		} // Json("(des *DESDatabase) CreateDESDatabase(): -> newUser", newUser)
	}

	return err
}


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
	"strings"

	"gorm.io/driver/postgres" // go get gorm.io/driver/postgres
	"gorm.io/gorm"            // go get gorm.io/gorm
	"gorm.io/gorm/logger"
	"golang.org/x/crypto/bcrypt" // go get golang.org/x/crypto/bcrypt
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
	dbi.DB.Logger = logger.Default.LogMode(logger.Info)

	return err
}
func (dbi *DBI) Close() (err error) {
	db, err := dbi.DB.DB()
	if err != nil {
		return err
	}
	return db.Close()
}
func (dbi *DBI) DropDatabase(db_name string) {
	dropDBCommand := fmt.Sprintf(`DROP DATABASE %s`, db_name)
	dbi.DB.Exec(dropDBCommand)
}
func (dbi *DBI) CheckDatabaseExists(db_name string, exists *bool) {
	checkExistsCommand := `SELECT EXISTS ( SELECT datname FROM pg_catalog.pg_database WHERE datname=? )`
	dbi.DB.Raw(checkExistsCommand, db_name).Scan(exists)
}
func (dbi *DBI) CreateDatabase(db_name string, drop bool) (exists bool) {

	exists = false
	dbi.CheckDatabaseExists(db_name, &exists)
	fmt.Printf("\n(dbi *DBI) CreateDatabase: %s exists: %v\n", db_name, exists)

	if exists && drop {
		fmt.Printf("\n(dbi *DBI) CreateDatabase: Dropping %s...", db_name)
		dbi.DropDatabase(db_name)
		dbi.CheckDatabaseExists(db_name, &exists)
		fmt.Printf("\n(dbi *DBI) CreateDatabase: %s exists: %v", db_name, exists)
	}

	if !exists {
		createDBCommand := fmt.Sprintf(`CREATE DATABASE %s WITH OWNER = datacan 
			ENCODING = 'UTF8' LC_COLLATE = 'C.UTF-8' LC_CTYPE = 'C.UTF-8' TABLESPACE = pg_default CONNECTION LIMIT = -1 IS_TEMPLATE = False;`,
			db_name,
		)
		fmt.Printf("\nCreating %s...", db_name)
		dbi.DB.Exec(createDBCommand)
		// dbi.CheckDatabaseExists(db_name, &exists)
		// fmt.Printf("\n(dbi *DBI) CreateDatabase: %s exists: %v\n", db_name, exists)
	}
	return exists
}

/* DES DATABASE */
type DESDatabase struct {
	DBI
}

var DES DESDatabase = DESDatabase{
	DBI: DBI{ConnStr: DES_DB_CONNECTION_STRING},
}

func (des *DESDatabase) CreateDESDatabase(drop bool) (err error) {

	/* ADMIN DB - CONNECT TO THE ADMIN DATABASE */
	adb := DBI{ConnStr: ADMIN_DB_CONNECTION_STRING}
	adb.Connect()
	defer adb.Close()
	exists := adb.CreateDatabase(DES_DB, drop)
	adb.Close()

	/* DES DATABASE - CONNECT AND CREATE TABLES */
	fmt.Printf("\n(des *DESDatabase) CreateDESDatabase: ConnStr: %s\n", des.ConnStr)
	DES.Connect()

	if (!exists) {
		err = DES.DB.Migrator().CreateTable(
			&User{},
			&DESDev{},
			&DESJob{},
		)	

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(ADM_PW), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		role := string("admin")
		newUser := User{
			Name:     ADM_USER,
			Email:    strings.ToLower(ADM_EMAIL),
			Password: string(hashedPassword),
			Role: role,
			// Photo:    &payload.Photo,
		}
		if result := DES.DB.Create(&newUser); result.Error != nil {
			fmt.Printf("\nCreate admin user failed...\n%s\n", result.Error.Error())
		}

	} else {
		err = DES.DB.AutoMigrate(
			&User{},
			&DESDev{},
			&DESJob{},
		)
	}
	if err != nil {
		fmt.Println(err.Error())
	}
	return err
}

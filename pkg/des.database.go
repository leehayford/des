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
	"strings"
	"sync"

	/* https://gorm.io/docs/ */
	"golang.org/x/crypto/bcrypt" // go get golang.org/x/crypto/bcrypt

	"gorm.io/driver/postgres" // go get gorm.io/driver/postgres
	"gorm.io/gorm"            // go get gorm.io/gorm
	"gorm.io/gorm/logger"
)

/*
	DATABASE CLIENT

ALL DATABASES IN THE DES ARE ACCESSED VIA A DBClient
*/
type DBClient struct {
	ConnStr string
	*gorm.DB
	
	/* WE DON'T USE THIS RIGHT NOW BUT SHOULD WE EVER SWITCH BACK TO 
		- USING THIS TYPE OF DBClient FOR JOB DATABASES, WE'LL WANT THE SWAP TO GO SMOOTHLY */
	RWM *sync.RWMutex 
}

func (dbc *DBClient) Connect() (err error) {

	if dbc.ConnStr == "" {
		return fmt.Errorf("DBClient connection string is empty")
	}

	/* THIS DBClient SHOULD ALREADY HAVE A RWMutex 
	BUT WE'LL JUST MAKE SURE BEFORE TRYING TO LOCK IT */
	if dbc.RWM == nil {
		dbc.RWM = &sync.RWMutex{} /* WE DON'T USE THIS RIGHT NOW BUT LET'S KEEP OUR PANTS UP JUST IN CASE */
	}

	// dbc.RWM.Lock()
	if dbc.DB, err = gorm.Open(postgres.Open(dbc.ConnStr), &gorm.Config{}); err != nil {
		// fmt.Printf("\n(dbc *DBClient) Connect() -> %s -> FAILED! \n", dbc.GetDBName())
		return LogErr(err)
	}
	dbc.DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
	dbc.DB.Logger = logger.Default.LogMode(logger.Error)
	// dbc.RWM.Unlock()

	// fmt.Printf("\n(dbc *DBClient) Connect() -> %s -> connected... \n", dbc.GetDBName())
	return err
}
func (dbc *DBClient) Disconnect() (err error) {

	// /* THIS DBClient SHOULD ALREADY HAVE A RWMutex 
	// BUT WE'LL JUST MAKE SURE BEFORE TRYING TO LOCK IT */
	// if dbc.RWM != nil {
	// 	dbc.RWM.Lock()
	// }

	db, err := dbc.DB.DB()
	if err != nil {
		return LogErr(err)
	}
	if err = db.Close(); err != nil {
		return LogErr(err)
	}
	// fmt.Printf("\n(dbc *DBClient) Disconnect() -> %s -> connection closed. \n", dbc.GetDBName())

	/*  WE DON'T UNLOCK BECAUSE THE OLD RWMutex IS GONE, 
	ALONG WITH ANY PENDING DB OPERATIONS  */
	dbc = &DBClient{
		RWM: &sync.RWMutex{}, /* WE DON'T USE THIS RIGHT NOW BUT LET'S KEEP OUR PANTS UP JUST IN CASE */
	}

	return
}
func (dbc *DBClient) GetDBNameFromConnStr() string {
	str := strings.Split(dbc.ConnStr, "/")
	if len(str) == 4 {
		/* THIS IS A VALID CONNECTION STRING */
		return str[3] /* TODO: IMPROVE VALIDATION ? */
	} else {
		return ""
	}
}

/*
	ADMIN DATABASE

USED TO MANAGE ALL OTHER DATABASES ON THIS DES
*/
var ADB ADMINDatabase = ADMINDatabase{
	DBClient: DBClient{
		ConnStr: ADMIN_DB_CONNECTION_STRING,
		RWM: &sync.RWMutex{}, /* WE DON'T USE THIS RIGHT NOW BUT LET'S KEEP OUR PANTS UP JUST IN CASE */
	},
}

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

/*
	DES DATABASE

USED TO MANAGE DEVICES AND RELATED JOBS ON THIS DES
CURRENTLY MANAGES USERS BUT THIS WILL CHANGE IN THE NEXT VERSION
*/
var DES DESDatabase = DESDatabase{
	DBClient: DBClient{
		ConnStr: DES_DB_CONNECTION_STRING,
		RWM: &sync.RWMutex{}, /* WE DON'T USE THIS RIGHT NOW BUT LET'S KEEP OUR PANTS UP JUST IN CASE */
	},
}

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

/* TODO:  */
// func WriteDES_XXX(xxx SomeModel, dbc *DBClient) (err error) {

// 	/* WHEN Write IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING
// 	WE WANT TO PREVENT CONCURRENT WRITES / DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
// 	*/
// 	if dbc.RWM == nil {
// 		dbc.RWM = &sync.RWMutex{}
// 	}
// 	dbc.RWM.Lock()
// 	res := dbc.Create(&xxx)
// 	dbc.RWM.Unlock()

// 	return res.Error
// }


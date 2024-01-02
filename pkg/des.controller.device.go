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
	"regexp"
	"strings"
	"time"
)

/* ENSURE SERIAL IS:  
		- NOT BLANK
		- UPPERCASE
		- UNIQUE
		- 10 OR LESS ALPHANUMERICA CHARACHTERS
*/
func ValidateSerialNumber(serial string ) (err error) { 
	

	/* ENSURE SERIAL IS NOT BLANK */
	if serial == "" {
		return fmt.Errorf("Serial number is blank.")
	}

	/* ENSURE SERIAL IS UPPERCASE */
	serial = strings.ToUpper(serial)
	
	/* ENSURE SERIAL HAS NO MORE THAN 10 CHARACTERS */
	if len(serial) > 10 {
		return fmt.Errorf("Serial number may contain upto 10 alphanumeric values.")
	}

	/* ENSURE SERIAL HAS ONLY ALPHANUMERIC CHARACTERS */
	if alphaNum := regexp.MustCompile(`^[A-Za-z0-9]*$`).MatchString(serial); !alphaNum {
		return fmt.Errorf("Serial number may contain only alphanumeric values.")
	}

	/* ENSURE SERIAL IS UNIQUE */
	regs, err :=  GetDESDeviceList()
	if err != nil {
		return // WHTEVER THE ERROR WAS...
	}

	for _, reg := range regs {
		if serial == reg.DESDevSerial {
			return fmt.Errorf("Serial number already exists.")
		} 
	} 

	return
}

/* NOT IMPLEMENTED: INTENDED AS API ENDPOINT FOR D2D CORE ************************************************/

/* NOT IMPLEMENTED: INTENDED AS API ENDPOINT FOR D2D CORE  */
func RegisterDESDevice(src string, dev DESDev) ( reg DESRegistration, err error) {

	/*
		CREATE A DEVICE RECORD IN THE DES DB FOR THIS DEVICE
		 - Creates a new DESevice in the DES database
		 - Gets the C001V001Device's DeviceID from the DES Database
	*/
	
	dev.DESDevRegTime = time.Now().UTC().UnixMilli()
	dev.DESDevRegAddr = src
	if dev_res := DES.DB.Create(&dev); dev_res.Error != nil {
		err =  dev_res.Error
		return
	}

	/*
		CREATE THE DEFAULT JOB FOR THIS DEVICE
		 - SERIALNUM-0000000000000000
		 - Create the registration job record in the DES database
		 - Create a new Job database for the job data
		 - Sets the the Device's active job
		 -
	*/
	job := DESJob{
		DESJobRegTime:   dev.DESDevRegTime,
		DESJobRegAddr:   dev.DESDevRegAddr,
		DESJobRegUserID: dev.DESDevRegUserID,
		DESJobRegApp:    dev.DESDevRegApp,

		DESJobName:  fmt.Sprintf("%s_CMDARCHIVE", dev.DESDevSerial),
		DESJobStart: dev.DESDevRegTime,
		DESJobEnd:   0,

		DESJobDevID: dev.DESDevID,
	}
	if job_res := DES.DB.Create(&job); job_res.Error != nil {
		err = job_res.Error
		return
	}

	reg = DESRegistration{DESDev: dev, DESJob: job}

	return
}

/* NOT IMPLEMENTED: INTENDED AS API ENDPOINT FOR D2D CORE  */
func GetDESDeviceList() (devs []DESDev, err error) {

	qry := DES.DB.
		Table("des_devs").
		Select("des_devs.*").
		Order("des_devs.des_dev_serial ASC")
	
	res := qry.Scan(&devs)
	// pkg.Json("GetDESDeviceList(): DESDevs", devs)
	err = res.Error
	return
}

/* NOT IMPLEMENTED: INTENDED AS API ENDPOINT FOR D2D CORE  */
func GetDESDeviceBySerial(serial string) (dev DESDev,  err error) {
	
	// qry := DES.DB.Table("des_devs").Select().Where()

	res := DES.DB.Order("des_dev_reg_time desc").First(&dev, "des_dev_serial =?", dev.DESDevSerial) 
	err = res.Error
	return 
}

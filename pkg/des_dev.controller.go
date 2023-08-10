
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
	"time"

	"github.com/gofiber/fiber/v2"
)

/*
USED WHEN:
 - DATACAN ADMIN WEB CLIENTS REGISTER NEW DEVICES ON THIS DES
 - WEB CLIENTS REQUEST ACCESS TO DEVICES/JOBS REGISTERED ON THIS DES

CLASS & VERSION AGNOSTIC
*/
type DESRegistration struct {
	DESDev //`json:"des_device"`
	DESJob    //`json:"des_job"`
}

func RegisterDesDev(c *fiber.Ctx) (err error) {

	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status": "fail", 
			"message": "You must be an administrator to register devices",
		})
	}

	device := DESDev{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail", 
			"message": err.Error(),
		})
	}
	if errors := ValidateStruct(device); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail", 
			"errors": errors,
		})
	}

	/*
		CREATE A DEVICE RECORD IN THE DES DB FOR THIS DEVICE
		 - Creates a new DESevice in the DES database
		 - Gets the C001V001Device's DeviceID from the DES Database
	*/
	device.DESDevRegTime = time.Now().UTC().UnixMilli()
	device.DESDevRegAddr = c.IP()
	if device_res := DES.DB.Create(&device); device_res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message": device_res.Error.Error(),
		})
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
		DESJobRegTime: device.DESDevRegTime,
		DESJobRegAddr: device.DESDevRegAddr,
		DESJobRegUserID: device.DESDevRegUserID,
		DESJobRegApp: device.DESDevRegApp,

		DESJobName: fmt.Sprintf("%s_0000000000000", device.DESDevSerial),
		DESJobStart: device.DESDevRegTime,
		DESJobEnd: 0,

		DESJobDevID: device.DESDevID,
	}
	if job_res := DES.DB.Create(&job); job_res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message": job_res.Error.Error(),
		})
	}

	reg := DESRegistration{
		DESDev: device,
		DESJob: job,
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success", 
		"data": fiber.Map{"device": &reg},
	})
} 

func GetDesDevList(c *fiber.Ctx) (err error) {

	devices := []DESDev{} 

	/* 
	WHERE A DEVICE HAS MORE THAN ONE REGISTRATION RECORD
	WE WANT THE LATEST
	*/
	subQry := DES.DB.
	Table("des_devs").
	Select(`des_dev_serial, MAX(des_dev_reg_time) AS max_time`).
	Group("des_dev_serial")

	qry := DES.DB.
	Select(" * ").
	Joins(`JOIN ( ? ) x 
		ON des_devs.des_dev_serial = x.des_dev_serial 
		AND des_devs.des_dev_reg_time = x.max_time`, 
		subQry,).
	Order("des_devs.des_dev_serial DESC")

	if res := qry.Find(&devices); res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message":  fmt.Sprintf("GetDesDevList(...) -> query failed:\n%s\n", res.Error.Error()),
			"data": fiber.Map{"devices": devices},
		})
	}
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success", 
		"message": "You are a tolerable person!",
		"data": fiber.Map{"devices": devices},
	})
}

func GetDesDevBySerial(c *fiber.Ctx) (err error) {

	reg := DESRegistration{}
	if err = c.BodyParser(&reg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail", 
			"message": fmt.Sprintf("GetDesDevBySerial(...) -> BodyParser failed:\n%s\n", err.Error()),
		})
	}

	if res := DES.DB.Order("des_dev_reg_time desc").First(&reg.DESDev, "des_dev_serial =?", reg.DESDev.DESDevSerial); res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message":  fmt.Sprintf("GetDesDevBySerial(...) -> query failed:\n%s\n", res.Error.Error()),
			"data": fiber.Map{"device": reg},
		})
	}
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success", 
		"message": "You are a tolerable person!",
		"data": fiber.Map{"device": reg},
	})
}

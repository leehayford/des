
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
USED WHEN DATACAN ADMIN WEB CLIENTS REGISTER NEW C001V001 JOBS ON THIS DES
*/
func RegisterDesJob(c *fiber.Ctx) (err error) {
	
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status": "fail", 
			"message": "You must be an administrator to register jobs",
		})
	}

	job := DESJob{}
	if err := c.BodyParser(&job); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail", 
			"message": err.Error(),
		})
	}

	if errors := ValidateStruct(job); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail", 
			"errors": errors,
		})
	}

	job.DESJobRegTime = time.Now().UTC().UnixMicro()
	job.DESJobRegAddr = c.IP()
	if job_res := DES.DB.Create(&job); job_res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message": job_res.Error.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "success", "data": fiber.Map{"job": &job}})
}

func GetDesJobList(c *fiber.Ctx) (err error) {

	jobs := []DESJob{}

	if res := DES.DB.Find(&jobs); res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message":  fmt.Sprintf("GetDesJobList(...) -> query failed:\n%s\n", res.Error.Error()),
			"data": fiber.Map{"jobs": jobs},
		})
	}
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success", 
		"message": "You are a tolerable person!",
		"data": fiber.Map{"jobs": jobs},
	})
}

func GetDesJobByName(c *fiber.Ctx) (err error) {

	reg := DESRegistration{}
	if err = c.BodyParser(&reg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail", 
			"message": fmt.Sprintf("GetDesJobByName(...) -> BodyParser failed:\n%s\n", err.Error()),
		})
	}

	if res := DES.DB.First(&reg.DESJob, "des_job_name =?", reg.DESJob.DESJobName); res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message":  fmt.Sprintf("GetDesJobByName(...) -> get job query failed:\n%s\n", res.Error.Error()),
			"data": fiber.Map{"job": reg},
		})
	}
	
	if res := DES.DB.First(&reg.DESDev, "des_dev_id =?", reg.DESJob.DESJobDevID); res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message":  fmt.Sprintf("GetDesJobByName(...) -> get job device query failed:\n%s\n", res.Error.Error()),
			"data": fiber.Map{"job": reg},
		})
	}
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success", 
		"message": "You are a tolerable person!",
		"data": fiber.Map{"job": reg},
	})
}
package c001v001

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/leehayford/des/pkg"
)

/*
RETURNS THE LIST OF JOBS REGISTERED TO THIS DES

ALONG WITH THE DEVICE FOR EACH OF THOSE JOBS
IN THE FORM OF A DESRegistration
*/
func HandleGetJobList(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleGetJobList( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != pkg.ROLE_ADMIN && role != pkg.ROLE_OPERATOR && role != pkg.ROLE_USER{
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be a registered user to view job list",
		})
	}

	regs, err := GetJobList()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("GetJobList(...) -> query failed:\n%s\n", err),
			"data":    fiber.Map{"jobs": regs},
		})
	}

	stat := "fail"
	msg := fmt.Sprintf("GetJobs(...) -> NO JOBS.\n")
	jobs := GetJobs(regs)
	if len(jobs) > 0 {
		stat = "success"
		msg = "You are a tolerable person!"
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  stat,
		"message": msg,
		"data":    fiber.Map{"jobs": jobs},
	})
}

/*
RETURNS ALL DATA ASSOCIATED WITH A JOB

ALONG WITH THE DEVICE AND ANY REPORT INFORMATION
*/
func HandleGetJobData(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleGetJobData( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != pkg.ROLE_ADMIN && role != pkg.ROLE_OPERATOR && role != pkg.ROLE_USER{
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be a registered user to view job data",
		})
	}

	reg := pkg.DESRegistration{}
	if err = c.BodyParser(&reg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleGetJobData(): -> c.BodyParser(&reg) -> reg", reg)

	job := Job{DESRegistration: reg}
	job.GetJobData()

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"job": job},
	})
}

/*
RETURNS THE LIST OF EVENT TYPES FOR A CLASS 001 VERSION 001 DEVICE / JOB
*/
func HandleGetEventTypeLists(c *fiber.Ctx) (err error) {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"event_types": EVENT_TYPES},
	})
}

/*
CREATES A REPORT INSTANCE IN THE REQUESTED JOB DB
RETURNS THE REPORT
*/
func HandleNewReport(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleNewReport( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != pkg.ROLE_ADMIN && role != pkg.ROLE_OPERATOR {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an operator user to create a report",
		})
	}

	rep := &Report{}
	if err = c.BodyParser(rep); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleCreateReport(): -> c.BodyParser(&rep) -> rep", rep)

	job := &Job{DESRegistration: rep.DESRegistration}
	// pkg.Json("HandleCreateReport(): job", job)

	// scls := &SecScales{}
	// if err = scls.AutoScaleSection(job, job.DESJobStart, job.DESJobEnd); err != nil {
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
	// 		"status":  "fail",
	// 		"message": err.Error(),
	// 	})
	// } // pkg.Json("AutoScaleSection(): SecScales", scls)

	job.CreateDefaultReport(rep)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"report": rep},
	})
}

/*

*/
func HandleJobNewHeader(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleJobNewHeader( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != pkg.ROLE_ADMIN && role != pkg.ROLE_OPERATOR {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be a registered user to edit job data",
		})
	}

	/*
	ADD TO JOB DB
	UPDATE REG JSON / TOKEN
	RETURN HDR & REG
	*/

	return
}

/*

*/
func HandleJobNewEvent(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleJobNewEvent( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != pkg.ROLE_ADMIN && role != pkg.ROLE_OPERATOR {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be a registered user to edit job data",
		})
	}

	/*
	ADD TO JOB DB
	RETURN EVT
	*/

	return
}
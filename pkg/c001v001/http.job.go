package c001v001

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
)


/*
	RETURNS THE LIST OF JOBS REGISTERED TO THIS DES

	ALONG WITH THE DEVICE FOR EACH OF THOSE JOBS
	IN THE FORM OF A DESRegistration
*/
func HandleGetJobList(c *fiber.Ctx) (err error) {

	fmt.Printf("\nHandleGetJobList( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to view job list",
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

	jobs := GetJobs(regs)
	if len(jobs) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("GetJobList(...) -> NO JOBS.\n"),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"jobs": jobs},
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

/* NOT CURRENTLY IN USE... */
func (job *Job) GetJobData(limit int) (err error) {
	db := job.JDB()
	db.Connect()
	defer db.Disconnect()
	db.Select("*").Table("admins").Limit(limit).Order("adm_time DESC").Scan(&job.Admins)
	db.Select("*").Table("headers").Limit(limit).Order("hdr_time DESC").Scan(&job.Headers)
	db.Select("*").Table("configs").Limit(limit).Order("cfg_time DESC").Scan(&job.Configs)
	db.Select("*").Table("events").Limit(limit).Order("evt_time DESC").Scan(&job.Events)
	db.Select("*").Table("samples").Limit(limit).Order("smp_time DESC").Scan(&job.Samples)
	for _, smp := range job.Samples {
		job.XYPoints.AppendXYSample(smp)
	}
	db.Disconnect()
	// pkg.Json("GetJobData(): job", job)
	return
}

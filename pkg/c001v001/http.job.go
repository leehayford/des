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

package c001v001

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/leehayford/des/pkg"
)

func InitializeJobRoutes(app, api *fiber.App) (err error) {

	api.Route("/001/001/job", func(router fiber.Router) {
		router.Get("/event/list", HandleGetEventTypeLists)

		router.Get("/list", pkg.DesAuth, HandleGetJobList)
		router.Post("/data", pkg.DesAuth, HandleGetJobData)
		router.Post("/new_report", pkg.DesAuth, HandleNewReport)
		router.Post("/new_header", pkg.DesAuth, HandleJobNewHeader)
		router.Post("/new_event", pkg.DesAuth, HandleNewReportEvent)
		router.Post("/event_list", pkg.DesAuth, HandleGetJobEvents)
	})
	return
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
RETURNS THE LIST OF JOBS REGISTERED TO THIS DES

ALONG WITH THE DEVICE FOR EACH OF THOSE JOBS
IN THE FORM OF A DESRegistration
*/
func HandleGetJobList(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleGetJobList( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Viewer(c.Locals("role")) {
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
	if !pkg.UserRole_Viewer(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be a registered user to view job data.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	job := Job{}
	if err = c.BodyParser(&job); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleGetJobData(): -> c.BodyParser(&job) -> job", job)

	/* OPEN A JOB DATABASE CONNECTION FOR THIS REQUEST */
	if err = job.ConnectDBC(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	/* ENSURE DATABASE CONNECTION CLOSES AFTER THIS REQUEST */
	defer job.DBClient.Disconnect()

	/* QUERY JOB DATABASE */
	if err = job.GetJobData(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"job": job},
	})
}

/*
CREATES A REPORT INSTANCE IN THE REQUESTED JOB DB
RETURNS THE REPORT
*/
func HandleNewReport(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleNewReport( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an operator user to create a report",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	rep := Report{}
	if err = c.BodyParser(&rep); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleCreateReport(): -> c.BodyParser(&rep) -> rep", rep)

	/* OPEN A JOB DATABASE CONNECTION FOR THIS REQUEST */
	job := Job{DESRegistration: rep.DESRegistration}
	if err = job.ConnectDBC(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	/* ENSURE DATABASE CONNECTION CLOSES AFTER THIS REQUEST */
	defer job.DBClient.Disconnect()

	/* GENERATE THE NEW REPORT RECORDS IN THE JOB DATABASE */
	job.GenerateReport(&rep)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"report": rep},
	})
}

/* TODO:
 */
func HandleJobNewHeader(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleJobNewHeader( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
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
func HandleGetJobEvents(c *fiber.Ctx) (err error) {
	fmt.Printf("\nHandleGetJobEvents( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Viewer(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be a registered user to view job data",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	job := Job{}
	if err = c.BodyParser(&job); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	pkg.Json("HandleGetJobEvents(): -> c.BodyParser(&job) -> job.DESRegistration", job.DESRegistration)

	/* OPEN A JOB DATABASE CONNECTION FOR THIS REQUEST */
	if err = job.ConnectDBC(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	/* ENSURE DATABASE CONNECTION CLOSES AFTER THIS REQUEST */
	defer job.DBClient.Disconnect()

	if err = job.GetJobEvents(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	pkg.Json("HandleGetJobEvents(): -> job.GetJobEvents() -> job.Events", job.Events)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"evts": job.Events},
	})

}

/*
 */
func HandleNewReportEvent(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleNewReportEvent( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an operator to edit job data",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	job := Job{}
	if err = c.BodyParser(&job); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	pkg.Json("HandleNewReportEvent(): -> c.BodyParser(&job) -> job", job)

	/* OPEN A JOB DATABASE CONNECTION FOR THIS REQUEST */
	if err = job.ConnectDBC(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	/* ENSURE DATABASE CONNECTION CLOSES AFTER THIS REQUEST */
	defer job.DBClient.Disconnect()

	if err = job.NewReportEvent(c.IP(), &job.Events[0]); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	pkg.Json("HandleNewReportEvent(): -> job.CreateReportEvent -> job.Events[0]", job.Events[0])

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"evt": job.Events[0]},
	})
}

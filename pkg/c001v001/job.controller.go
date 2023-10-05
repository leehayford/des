package c001v001

import (
	"github.com/gofiber/fiber/v2"
)

func HandleGetEventTypeLists(c *fiber.Ctx) (err error) {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"event_types": EVENT_TYPES},
	})
}

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

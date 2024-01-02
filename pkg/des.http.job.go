package pkg

import (
	// "fmt"
	"github.com/gofiber/fiber/v2"
)

func InitializeDESJobRoutes(app, api *fiber.App) {
	api.Route("/job", func(router fiber.Router) {
			

	})
}

func ValidatePostRequestBody_Job(c *fiber.Ctx, reg *DESRegistration) (err error) {

	if err = ParseRequestBody(c, &reg); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	/*  TODO: ADDITIONAL JOB VALIDATION */
	return
}

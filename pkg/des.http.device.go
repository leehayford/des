package pkg

import (
	"github.com/gofiber/fiber/v2"
)

func InitializeDESDeviceRoutes(app, api *fiber.App) {
	api.Route("/device", func(router fiber.Router) {
			
		router.Post("/validate_serial", DesAuth, HandleValidateSerialNumber)

	})
}

func HandleValidateSerialNumber(c * fiber.Ctx) (err error) {

	/* CHECK USER PERMISSION */
	if !UserRole_Viewer(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be a registered user to validate serial numbers.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	serial := ""
	if err = c.BodyParser(&serial); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} 
	Json("HandleValidateSerialNumber(): -> c.BodyParser(&serial) -> serial", serial)

	if err = ValidateSerialNumber(serial); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Serial number OK.",
	})
}

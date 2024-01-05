package pkg

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
)

func InitializeDESDeviceRoutes(app, api *fiber.App) {
	api.Route("/device", func(router fiber.Router) {
			
		router.Post("/validate_serial", DesAuth, HandleValidateSerialNumber)

	})
}

/********************************************************************************************************/

/* NOT IMPLEMENTED: INTENDED AS API ENDPOINT FOR D2D CORE  *******************************/
func HandleValidateSerialNumber(c * fiber.Ctx) (err error) {

	/* CHECK USER PERMISSION */
	if !UserRole_Viewer(c.Locals("role")) {
		txt := "You must be a registered user to validate serial numbers."
		return c.Status(fiber.StatusForbidden).SendString(txt)
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	serial := ""
	if err := c.BodyParser(&serial); err != nil {
		txt := fmt.Sprintf("Invalid request body: %s", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(txt)
	} // Json("HandleValidateSerialNumber(): -> c.BodyParser(&serial) -> serial", serial)

	if err = ValidateSerialNumber(serial); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Serial number OK.",
	})
}

/* NOT IMPLEMENTED: INTENDED AS API ENDPOINT FOR D2D CORE  */
func HandleRegisterDESDevice(c *fiber.Ctx) (err error) {

	/* CHECK USER PERMISSION */
	if !UserRole_Admin(c.Locals("role")) {
		txt := "You must be an administrator to register DES devices"
		return c.Status(fiber.StatusForbidden).SendString(txt)
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := DESDev{}
	if err := c.BodyParser(&device); err != nil {
		txt := fmt.Sprintf("Invalid request body: %s", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(txt)
	}

	/*
		CREATE A DEVICE RECORD IN THE DES DB FOR THIS DEVICE
		 CREATE A JOB RECORD IN THE DES DB FOR THIS DEVICE CMDARCHIVE
	*/
	reg, err := RegisterDESDevice(c.IP(), device)
	if err != nil {
		txt := fmt.Sprintf("Registration failed: %s", err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString(txt)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"reg": &reg})
}

/* NOT IMPLEMENTED: INTENDED AS API ENDPOINT FOR D2D CORE  */
func HandleGetDESDeviceList(c *fiber.Ctx) (err error) {

	/* CHECK USER PERMISSION */
	if !UserRole_Admin(c.Locals("role")) {
		txt := "You must be an administrator to view DES device list"
		return c.Status(fiber.StatusForbidden).SendString(txt)
	}

	des_devs, err := GetDESDeviceList()
	if err != nil {
		txt := fmt.Sprintf("Failed to retrieve device list: %s", err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString(txt)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"devices": des_devs})
}

/* NOT IMPLEMENTED: INTENDED AS API ENDPOINT FOR D2D CORE  */
func HandleGetDESDeviceBySerial(c *fiber.Ctx) (err error) {

	/* CHECK USER PERMISSION */
	if !UserRole_Admin(c.Locals("role")) {
		txt := "You must be an administrator to view DES devices"
		return c.Status(fiber.StatusForbidden).SendString(txt)
	}

	reg := DESRegistration{}
	if err := c.BodyParser(&reg); err != nil {
		txt := fmt.Sprintf("Invalid request body: %s", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(txt)
	}

	/* TODO: MOVE THIS OUT OF THE HANDLER */
	if res := DES.DB.Order("des_dev_reg_time desc").First(&reg.DESDev, "des_dev_serial =?", reg.DESDev.DESDevSerial); res.Error != nil {
		txt := fmt.Sprintf("Device was not found: %s", res.Error.Error())
		return c.Status(fiber.StatusInternalServerError).SendString(txt)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"device": reg})
}

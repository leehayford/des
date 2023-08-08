
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

package controllers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/leehayford/des/pkg"
	"github.com/leehayford/des/pkg/models"
)

func RegisterDesDev(c *fiber.Ctx) (err error) {

	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status": "fail", 
			"message": "You must be an administrator to register devices",
		})
	}

	device := models.DESDev{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail", 
			"message": err.Error(),
		})
	}
	if errors := models.ValidateStruct(device); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail", 
			"errors": errors,
		})
	}

	device.DESDevRegTime = time.Now().UTC().UnixMicro()
	device.DESDevRegAddr = c.IP()
	res := pkg.DES.DB.Create(&device)
	fmt.Println(res.Error)
	fmt.Println(res.RowsAffected)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success", 
		"data": fiber.Map{"device": &device},
	})
} 

func GetDesDevList(c *fiber.Ctx) (err error) {

	devices := []models.DESDev{} // make([]models.DESDev, 0)

	if res := pkg.DES.DB.Order("des_dev_serial desc").Find(&devices); res.Error != nil {
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

type Serial struct {
	Serial string `json:"serial"`
}
func GetDesDevBySerial(c *fiber.Ctx) (err error) {

	sn := Serial{}
	if err = c.BodyParser(&sn); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail", 
			"message": fmt.Sprintf("GetDesDevBySerial(...) -> BodyParser failed:\n%s\n", err.Error()),
		})
	}

	device := models.DESDev{} 

	if res := pkg.DES.DB.First(&device, "des_dev_serial =?", sn.Serial); res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message":  fmt.Sprintf("GetDesDevBySerial(...) -> query failed:\n%s\n", res.Error.Error()),
			"data": fiber.Map{"device": device},
		})
	}
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success", 
		"message": "You are a tolerable person!",
		"data": fiber.Map{"device": device},
	})
}
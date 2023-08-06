
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
	var device *models.DESDev

	if err := c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "fail", "message": err.Error()})
	}

	errors := models.ValidateStruct(device)
	if errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "fail", "errors": errors})
	}

	device.DESDevRegTime = time.Now().UTC().UnixMicro()
	device.DESDevRegAddr = c.IP()

	res := pkg.DES.DB.Create(device)
	fmt.Println(res.Error)
	fmt.Println(res.RowsAffected)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "success", "data": fiber.Map{"user": &device}})
} 
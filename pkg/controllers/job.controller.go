
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
	"github.com/golang-jwt/jwt" // go get github.com/golang-jwt/jwt

	"github.com/leehayford/des/pkg"
	"github.com/leehayford/des/pkg/models"
)

/*
USED WHEN DATACAN ADMIN WEB CLIENTS REGISTER NEW C001V001 JOBS ON THIS DES
*/
func RegisterDesJob(c *fiber.Ctx) (err error) {
	
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status": "fail", 
			"message": "You must be an administrator to register jobs",
		})
	}

	job := models.DESJob{}
	if err := c.BodyParser(&job); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail", 
			"message": err.Error(),
		})
	}

	if errors := models.ValidateStruct(job); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail", 
			"errors": errors,
		})
	}

	job.DESJobRegTime = time.Now().UTC().UnixMicro()
	job.DESJobRegAddr = c.IP()
	if job_res := pkg.DES.DB.Create(&job); job_res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message": job_res.Error.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "success", "data": fiber.Map{"job": &job}})
}

func GetDesJobList(c *fiber.Ctx) (err error) {



	/* TODO: MOVE TO des_job.middleware.go */
	tokenString := c.Cookies("token")
	fmt.Printf("TOKEN RECEIVED: \n%s\n", tokenString)
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "fail", "message": "You are not logged in"})
	}
	tokenByte, err := jwt.Parse(tokenString, func(jwtToken *jwt.Token) (interface{}, error) {
		if _, ok := jwtToken.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %s", jwtToken.Header["alg"])
		}
		return []byte(pkg.JWT_SECRET), nil
	})
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "fail", "message": fmt.Sprintf("invalidate token: %v", err)})
	}
	claims, ok := tokenByte.Claims.(jwt.MapClaims)
	if !ok || !tokenByte.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "fail", "message": "invalid token claim"})
	}
	fmt.Printf("TOKEN CLAIMS: \n%s\n", claims)
	/* TODO: MOVE TO des_job.middleware.go */



	var jobs []models.DESJob

	if res := pkg.DES.DB.Find(&jobs); res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message":  fmt.Sprintf("GetDesJobList(...) -> query failed: %v", err),
		})
	}
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "data": fiber.Map{"jobs": jobs}})
}
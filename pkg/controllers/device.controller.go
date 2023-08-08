
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
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt" // go get github.com/golang-jwt/jwt

	"github.com/leehayford/des/pkg"
	"github.com/leehayford/des/pkg/models"
)

func RegisterDesDev(c *fiber.Ctx) (err error) {
	var device models.DESDev

	if err := c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "fail", "message": err.Error()})
	}

	errors := models.ValidateStruct(device)
	if errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "fail", "errors": errors})
	}

	device.DESDevRegTime = time.Now().UTC().UnixMicro()
	device.DESDevRegAddr = c.IP()

	res := pkg.DES.DB.Create(&device)
	fmt.Println(res.Error)
	fmt.Println(res.RowsAffected)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "success", "data": fiber.Map{"device": &device}})
} 

func GetDesDevList(c *fiber.Ctx) (err error) {

	var tokenString string
	authorization := c.Get("Authorization")
	tokenString = c.Cookies("token")
	fmt.Printf("TOKEN RECEIVED: \n%s\n", tokenString)
	if strings.HasPrefix(authorization, "Bearer ") {
		tokenString = strings.TrimPrefix(authorization, "Bearer ")
	// } else if c.Cookies("token") != "" {
	// 	tokenString = c.Cookies("token")
	}

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

	var devices []models.DESDev

	if res := pkg.DES.DB.Find(&devices); res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message":  fmt.Sprintf("GetDesDevList(...) -> query failed: %v", err),
		})
	}
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "data": fiber.Map{"devices": devices}})
}
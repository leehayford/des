
/* Data Exchange Server (DES) is a component of the Datacan Data2Desk (D2D) Platform.
License:

	[PROPER LEGALESE HERE...]

	INTERIM LICENSE DESCRIPTION:
	In spirit, this license:
	1. Allows <Third Party> to use, modify, and / or distributre this software in perpetuity so long as <Third Party> understands:
		a. The software is porvided as is without guarantee of additional support from DataCan in any form.
		b. The software is porvided as is without guarantee of exclusivity.

	2. Prohibits <Third Party> from taking any action which might interfere with DataCan's right to use, modify and / or distributre this software in perpetuity.
*/

package pkg

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2" // go get github.com/gofiber/fiber/v2
	"github.com/golang-jwt/jwt"   // go get github.com/golang-jwt/jwt
)

/* AUTHENTICATE USER AND GET THEIR ROLE */
func DesAuth(c *fiber.Ctx) (err error) {

	authorization := c.Get("Authorization")
	// fmt.Printf("AUTHORIZATION: \n%s\n", authorization)
	// fmt.Printf("ACCESS_TOKEN: \n%s\n", c.Query("access_token"))

	tokenString := ""
	if strings.HasPrefix(authorization, "Bearer ") {
		tokenString = strings.TrimPrefix(authorization, "Bearer ")
	} else if c.Cookies("token") != "" {
		tokenString = c.Cookies("token")
	} else if c.Query("access_token") != "" {
		tokenString = c.Query("access_token")
	}
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "fail",
			"message": "You are not logged in",
		})
	}

	tokenByte, err := jwt.Parse(tokenString, func(jwtToken *jwt.Token) (interface{}, error) {
		if _, jwt_err := jwtToken.Method.(*jwt.SigningMethodHMAC); !jwt_err {
			return nil, fmt.Errorf("unexpected signing method: %s", jwtToken.Header["alg"])
		}
		return []byte(JWT_SECRET), nil
	})
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("invalid token: %v", err),
		})
	}

	/* GET THE USER ROLE & PASS ALONG TO THE NEXT HANDLER */
	claims, ok := tokenByte.Claims.(jwt.MapClaims)
	if !ok || !tokenByte.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "fail",
			"message": "invalid token claim",
		})
	}
	c.Locals("role", claims["rol"])
	c.Locals("sub", claims["sub"])

	return c.Next()
}

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
	"time"

	"github.com/gofiber/fiber/v2" // go get github.com/gofiber/fiber/v2
	"github.com/golang-jwt/jwt"   // go get github.com/golang-jwt/jwt
	"golang.org/x/crypto/bcrypt"  // go get golang.org/x/crypto/bcrypt
)
/* https://codevoweb.com/how-to-properly-use-jwt-for-authentication-in-golang/ */
func SignUpUser(c *fiber.Ctx) error {
	var payload *SignUpInput

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "fail", "message": err.Error()})
	}

	errors := ValidateStruct(payload)
	if errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "fail", "errors": errors})

	}

	if payload.Password != payload.PasswordConfirm {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail", 
			"message": "Passwords do not match",
		})

	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "fail", "message": err.Error()})
	}

	user := User{
		Name:     payload.Name,
		Email:    strings.ToLower(payload.Email),
		Password: string(hashedPassword),
		Photo:    payload.Photo,
	}

	result := DES.DB.Create(&user)

	if result.Error != nil && strings.Contains(result.Error.Error(), "duplicate key value violates unique") {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"status": "fail", "message": "User with that email already exists"})
	} else if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Something bad happened"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success", 
		"data": fiber.Map{"user": user.FilterUserRecord()},
	})
}

func SignInUser(c *fiber.Ctx) error {
	payload := SignInInput{} // fmt.Println("SignInUser(c *fiber.Ctx)")

	if err := c.BodyParser(&payload); err != nil {
		fmt.Println("SignInUser(c *fiber.Ctx) -> c.BodyParser")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	if errors := ValidateStruct(payload); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": errors,
		})
	}

	user := User{}
	if result := DES.DB.First(&user, "email = ?", strings.ToLower(payload.Email)); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": "Invalid email or Password",
		})
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.Password)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": "Invalid email or Password",
		})
	}

	now := time.Now().UTC()

	claims := jwt.MapClaims{
		"sub": user.ID, // SUBJECT
		"rol": user.Role, // ROLE
		"exp": now.Add(JWT_EXPIRED_IN).Unix(),
		"iat": now.Unix(), // ISSUED AT
		"nbf": now.Unix(), // NOT VALID BEFORE
	}
	tokenByte := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := tokenByte.SignedString([]byte(JWT_SECRET))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("failed to generate access token: %v", err),
		})
	}

	// c.Cookie(&fiber.Cookie{
	// 	Name:     "token",
	// 	Value:    tokenString,
	// 	Path:     "/",
	// 	Expires:   now.Add(JWT_EXPIRED_IN),
	// 	Secure:   false,
	// 	HTTPOnly: true,
	// 	Domain:   "localhost",
	// })

	// refTokenByte := jwt.New(jwt.SigningMethodES256)
	// refClaims := refTokenByte.Claims.(jwt.MapClaims)
	// refClaims["sub"] = user.ID // SUBJECT
	// // refClaims["rol"] = user.Role // ROLE
	// refClaims["exp"] = now.Add(time.Duration(time.Hour * 24)).Unix()
	// // refClaims["iat"] = now.Unix() // ISSUED AT
	// // refClaims["nbf"] = now.Unix() // NOT VALID BEFORE

	// refTokenString, err := refTokenByte.SignedString([]byte("SOMETINGELSE"))
	// if err != nil {
	// 	return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
	// 		"status":  "fail",
	// 		"message": fmt.Sprintf("generate refresh token failed: %v", err),
	// 	})
	// }

	// c.Cookie(&fiber.Cookie{
	// 	Name:     "refresh",
	// 	Value:    refTokenString,
	// 	Path:     "/",
	// 	// MaxAge:   JWT_MAXAGE * 60,
	// 	Secure:   false,
	// 	HTTPOnly: true,
	// 	Domain:   "localhost",
	// })

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"token":  tokenString,
		// "refresh": refTokenString,
	})
}

func LogoutUser(c *fiber.Ctx) error {
	expired := time.Now().Add(-time.Hour * 24)
	c.Cookie(&fiber.Cookie{
		Name:    "token",
		Value:   "",
		Expires: expired,
	})
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
}

func GetMe(c *fiber.Ctx) error {
	id := c.Locals("sub") // fmt.Printf("\nID:\t%s\n", id)

	user := User{}
	DES.DB.First(&user, "id = ?", id)
	if user.ID.String() != id {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "the user belonging to this token no logger exists",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   fiber.Map{"user": user.FilterUserRecord()},
	})
}

func GetUserList(c *fiber.Ctx) error {

	fmt.Printf("\nGetUsers( ):\n")

	users := []User{}
	DES.DB.Find(&users)
	fmt.Printf("\nusrs: %d\n", len(users))

	userList := []UserResponse{}
	for _, user := range users {
		userList = append(userList, user.FilterUserRecord())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "These are all tolerable people!",
		"data":    fiber.Map{"users": userList},
	})
}

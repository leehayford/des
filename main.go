
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

package main

import (
    "fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	
	"github.com/leehayford/des/pkg"
	"github.com/leehayford/des/pkg/controllers"
	"github.com/leehayford/des/pkg/middleware"
	// "github.com/leehayford/des/pkg/models"
)

func main() {

	pkg.DES.CreateDESDatabase(false)
	pkg.DES.Connect()
	defer pkg.DES.Close()

	/* MAIN SER$VER */
	app := fiber.New()
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:8080, http://localhost:5173, http://localhost:58714",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST",
		AllowCredentials: true,
	}))	
	app.Get("app/healthchecker", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON( fiber.Map{
			"status": "success",
			"message": "Data Exchange Server",
		})
	})

	/* API ROUTES */
	api := fiber.New()
	app.Mount("/api", api)
	app.Get("/user/me", middleware.DeserializeUser, controllers.GetMe)

	/* AUTH & USER ROUTES */
	api.Route("/auth", func(router fiber.Router) {
		router.Post("/register", controllers.SignUpUser)
		router.Post("/login", controllers.SignInUser)
		router.Get("/logout", middleware.DeserializeUser, controllers.LogoutUser)
	})

	/* DEVICE ROUTES */
	api.Route("/device", func(router fiber.Router) {
		router.Post("/register", controllers.RegisterDesDev)
		router.Get("/list", controllers.GetDesDevList)
	})

	/* JOB ROUTES */
	api.Route("/job", func(router fiber.Router) {
		
	})

	api.All("*", func(c *fiber.Ctx) error {
		path := c.Path()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("Path: %v does not exists on this server", path),
		})
	})

	log.Fatal(app.Listen("127.0.0.1:8007"))

}
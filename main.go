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
	"flag"
	"fmt"
	"log"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/leehayford/des/pkg"
	"github.com/leehayford/des/pkg/c001v001"
)

func main() {

	/* ADMIN DB - CONNECT TO THE ADMIN DATABASE */
	pkg.ADB.Connect()
	defer pkg.ADB.Close()

	cleanDB := flag.Bool("clean", false, "Drop and recreate databases")
	flag.Parse()

	if (*cleanDB) {
		/* CLEAN DATABASE - DROP ALL */
		pkg.ADB.DropAllDatabases()
	}

	/* CREATE OR MIGRATE DES DATABASE & CONNECT */
	exists := pkg.ADB.CheckDatabaseExists(pkg.DES_DB)
	if !exists {
		pkg.ADB.CreateDatabase(pkg.DES_DB)
	}
	pkg.DES.Connect()
	defer pkg.DES.Close()

	/* IF DES DATABASE DIDN'T ALREADY EXIST, CREATE TABLES, OTHERWISE MIGRATE */
	if err := pkg.DES.CreateDESTables(exists); err != nil {
		pkg.TraceErr(err)
	}

	
	/********************************************************************************************/
	/* DEMO DEVICES -> NOT FOR PRODUCTION */
	fmt.Println("\n\nConnecting all C001V001 MQTT DemoDevice Clients...")
	c001v001.DemoDeviceClient_ConnectAll()
	defer c001v001.DemoDeviceClient_DisconnectAll()
	/********************************************************************************************/


	/* MQTT - C001V001 - SUBSCRIBE TO ALL REGISTERED DEVICES */
	/* DATABASE - C001V001 - CONNECT ALL DEVICES TO JOB DATABASES */
	fmt.Println("\n\nConnecting all C001V001 Device Clients...")
	c001v001.DeviceClient_ConnectAll()
	defer c001v001.DeviceClient_DisconnectAll()


	/* MAIN SER$VER */
	app := fiber.New()
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		/* TODO: LIMIT ALLOWED ORIGINS FOR PRODUCTION DEPLOYMENT */
		AllowOrigins:     "http://localhost:8080, http://localhost:4173, http://localhost:5173, http://localhost:58714",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, Cache-Control",
		AllowMethods:     "GET, POST",
		AllowCredentials: true,
	}))

	/* API ROUTES */
	api := fiber.New()
	app.Mount("/api", api)
	app.Get("/user", pkg.GetUsers)
	app.Get("/user/me", pkg.DesAuth, pkg.GetMe)

	/* AUTH & USER ROUTES */
	api.Route("/auth", func(router fiber.Router) {
		router.Post("/register", pkg.SignUpUser)
		router.Post("/login", pkg.SignInUser)
		router.Get("/logout", pkg.DesAuth, pkg.LogoutUser)
	})

	/* C001V001 DEVICE ROUTES */
	api.Route("/001/001/device", func(router fiber.Router) {
		router.Post("/register", pkg.DesAuth, (&c001v001.Device{}).HandleRegisterDevice)
		router.Post("/start", pkg.DesAuth, (&c001v001.Device{}).HandleStartJob)
		router.Post("/end", pkg.DesAuth, (&c001v001.Device{}).HandleEndJob)
		router.Post("/admin", pkg.DesAuth, (&c001v001.Device{}).HandleSetAdmin)
		router.Post("/header", pkg.DesAuth, (&c001v001.Device{}).HandleSetHeader)
		router.Post("/config", pkg.DesAuth, (&c001v001.Device{}).HandleSetConfig)
		router.Get("/list", pkg.DesAuth, c001v001.HandleGetDeviceList)
		router.Get("/ws", pkg.DesAuth, websocket.New(
			(&c001v001.DeviceUserClient{}).WSDeviceUserClient_Connect,
		))
	})

	/* C001V001 JOB ROUTES */
	api.Route("/001/001/job", func(router fiber.Router) {
		router.Get("/event/list", c001v001.HandleGetEventTypeLists)
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

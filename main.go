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

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"

	"github.com/leehayford/des/pkg"
	"github.com/leehayford/des/pkg/c001v001"
)

func main() {

	/* ADMIN DB - CONNECT TO THE ADMIN DATABASE */
	pkg.ADB.Connect()
	defer pkg.ADB.Disconnect()

	cleanDB := flag.Bool("clean", false, "Drop and recreate databases")
	sim := flag.Bool("sim", false, "Run as device simulator only")
	demoQty := flag.Int("demos", 5, "Create n demo devices if there are none currently") /* DEMO -> NOT FOR PRODUCTION */
	flag.Parse()

	if *cleanDB {
		/* CLEAN DATABASE - DROP ALL */
		pkg.ADB.DropAllDatabases()
	}

	/* CREATE OR MIGRATE DES DATABASE & CONNECT */
	exists := pkg.ADB.CheckDatabaseExists(pkg.DES_DB)
	if !exists {
		pkg.ADB.CreateDatabase(pkg.DES_DB)
	}
	pkg.DES.Connect()
	defer pkg.DES.Disconnect()

	/* IF DES DATABASE DIDN'T ALREADY EXIST, CREATE TABLES, OTHERWISE MIGRATE */
	if err := pkg.DES.CreateDESTables(exists); err != nil {
		pkg.TraceErr(err)
	}

	/* MAIN SERVER */
	app := fiber.New()
	api := fiber.New()

	if *sim {

		/********************************************************************************************/
		/* DEMO DEVICES -> NOT FOR PRODUCTION */
		fmt.Println("\n\nConnecting all C001V001 MQTT DemoDevice Clients...")
		c001v001.DemoDeviceClient_ConnectAll(*demoQty)
		defer c001v001.DemoDeviceClient_DisconnectAll()
		/********************************************************************************************/

	} else {

		/* MQTT - C001V001 - SUBSCRIBE TO ALL REGISTERED DEVICES */
		/* DATABASE - C001V001 - CONNECT ALL DEVICES TO JOB DATABASES */
		fmt.Println("\n\nConnecting all C001V001 Device Clients...")
		c001v001.DeviceClient_ConnectAll()
		defer c001v001.DeviceClient_DisconnectAll()

		/* MAIN SERVER - LOGGING AND CORS */
		app.Use(logger.New())
		app.Use(cors.New(cors.Config{
			/* TODO: LIMIT ALLOWED ORIGINS FOR PRODUCTION DEPLOYMENT */
			AllowOrigins:     "https://vw1.data2desk.com, http://localhost:8080, http://localhost:4173, http://localhost:5173, http://localhost:58714",
			AllowHeaders:     "Origin, Content-Type, Accept, Authorization, Cache-Control",
			AllowMethods:     "GET, POST",
			AllowCredentials: true,
		}))

		/* AUTH & USER ROUTES */
		api.Route("/user", func(router fiber.Router) {
			router.Post("/signup", pkg.SignUpUser)
			router.Post("/login", pkg.SignInUser)
			router.Get("/list", pkg.GetUserList) /* TODO: AUTH */
			router.Get("/me", pkg.DesAuth, pkg.GetMe)
			router.Get("/logout", pkg.DesAuth, pkg.LogoutUser)
		})

		/* C001V001 DEVICE ROUTES */
		api.Route("/001/001/device", func(router fiber.Router) {
			// router.Post("/register", pkg.DesAuth, c001v001.HandleRegisterDevice)

			router.Post("/start", pkg.DesAuth, c001v001.HandleStartJob)
			router.Post("/cancel_start", pkg.DesAuth, c001v001.HandleCancelStartJob)

			router.Post("/end", pkg.DesAuth, c001v001.HandleEndJob)
			
			router.Post("/admin", pkg.DesAuth, c001v001.HandleSetAdmin)
			router.Post("/state", pkg.DesAuth, c001v001.HandleSetState)
			router.Post("/header", pkg.DesAuth, c001v001.HandleSetHeader)
			router.Post("/config", pkg.DesAuth, c001v001.HandleSetConfig)
			router.Post("/event", pkg.DesAuth, c001v001.HandleCreateDeviceEvent)
			router.Post("/search", pkg.DesAuth, c001v001.HandleSearchDevices)
			router.Get("/list", pkg.DesAuth, c001v001.HandleGetDeviceList)

			app.Use("/ws", func(c *fiber.Ctx) error {
				if websocket.IsWebSocketUpgrade(c) {
					c.Locals("allowed", true)
					return c.Next()
				}
				return fiber.ErrUpgradeRequired
			})
			router.Get("/ws", pkg.DesAuth, websocket.New(
				(&c001v001.DeviceUserClient{}).WSDeviceUserClient_Connect,
			))
		})

		/* C001V001 JOB / REPORTING ROUTES */
		api.Route("/001/001/job", func(router fiber.Router) {
			router.Get("/event/list", c001v001.HandleGetEventTypeLists)

			router.Get("/list", pkg.DesAuth, c001v001.HandleGetJobList)
			router.Post("/data", pkg.DesAuth, c001v001.HandleGetJobData)
			router.Post("/new_report", pkg.DesAuth, c001v001.HandleNewReport)
			router.Post("/new_header", pkg.DesAuth, c001v001.HandleJobNewHeader)
			router.Post("/new_event", pkg.DesAuth, c001v001.HandleJobNewEvent)
		})

	}

	/* API ROUTES */
	app.Mount("/api", api)

	api.All("*", func(c *fiber.Ctx) error {
		path := c.Path()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("Path: %v does not exists on this server", path),
		})
	})

	log.Fatal(app.Listen(pkg.APP_HOST))
}

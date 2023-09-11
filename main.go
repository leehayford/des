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

	"time"

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

	/* CLEAN DATABASE - DROP ALL */
	pkg.ADB.DropAllDatabases()

	/* CREATE / MIGRATE & CONNECT DES DATABASE */
	exists := pkg.ADB.CheckDatabaseExists(pkg.DES_DB)
	if !exists {
		pkg.ADB.CreateDatabase(pkg.DES_DB)
	}

	pkg.DES.Connect()
	defer pkg.DES.Close()
	/* IF DES DATABASE DIDN'T ALREADY EXIST, CREATE TABLES, OTHERWISE MIGRATE */
	if err := pkg.CreateDESDatabase(exists); err != nil {
		pkg.TraceErr(err)
	}

	// /* DEMO DEVICES -> NOT FOR PRODUCTION */

	regs, err := c001v001.GetDemoDeviceList()
	if err != nil {
		pkg.TraceErr(err)
	}

	if len(regs) == 0 {
		user := pkg.User{}
		pkg.DES.DB.Last(&user)
		regs = append(regs, c001v001.MakeDemoC001V001("DEMO000000", user.ID.String()))
		regs = append(regs, c001v001.MakeDemoC001V001("DEMO000001", user.ID.String()))
		regs = append(regs, c001v001.MakeDemoC001V001("DEMO000002", user.ID.String()))
		// regs = append(regs, c001v001.MakeDemoC001V001("DEMO000003", user.ID.String()))
		// regs = append(regs, c001v001.MakeDemoC001V001("DEMO000004", user.ID.String()))
		c001v001.MakeDemoC001V001("RENE123456", user.ID.String())
	}

	fmt.Println("\n\nConnecting all C001V001 MQTT DemoDevice Clients...")
	for _, reg := range regs {
		demo := c001v001.DemoDeviceClient{
			Device: c001v001.Device{
				DESRegistration: reg,
				Job:             c001v001.Job{DESRegistration: reg},
			},
		}

		demo.MQTTDemoDeviceClient_Connect()
		defer demo.MQTTDemoDeviceClient_Disconnect()

		// pkg.TraceFunc("Call -> device.GetDeviceStatus( )")
		demo.GetDeviceStatus()
		go demo.Demo_Simulation(time.Now().UTC())
	}

	/* MQTT - C001V001 - SUBSCRIBE TO ALL REGISTERES DEVICES */
	fmt.Println("\n\nConnecting all C001V001 MQTT Device Clients...")
	c001v001.MQTTDeviceClient_CreateAndConnectAll()

	/* MAIN SER$VER */
	app := fiber.New()
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:8080, http://localhost:4173, http://localhost:5173, http://localhost:58714",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, Cache-Control",
		AllowMethods:     "GET, POST",
		AllowCredentials: true,
	}))
	app.Get("app/healthchecker", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "success",
			"message": "Data Exchange Server",
		})
	})

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

	// /* DES DEVICE ROUTES */
	// api.Route("/device", func(router fiber.Router) {
	// 	// router.Post("/register", pkg.DesAuth, controllers.RegisterDesDev)
	// 	router.Get("/list", pkg.DesAuth, pkg.HandleGetDesDevList)
	// 	router.Post("/serial", pkg.DesAuth, pkg.HandleGetDesDevBySerial)
	// })

	/* DES JOB ROUTES */
	api.Route("/job", func(router fiber.Router) {
		// router.Post("/register", pkg.DesAuth, pkg.RegisterDesJob)
		router.Get("/list", pkg.DesAuth, pkg.GetDesJobList)
		router.Post("/name", pkg.DesAuth, pkg.GetDesJobByName)
	})

	/* C001V001 DEVICE ROUTES */
	api.Route("/001/001/device", func(router fiber.Router) {
		router.Post("/register", pkg.DesAuth, (&c001v001.Device{}).HandleRegisterDevice)
		router.Post("/start", pkg.DesAuth, (&c001v001.Device{}).HandleStartJob)
		router.Post("/end", pkg.DesAuth, (&c001v001.Device{}).HandleEndJob)
		router.Get("/list", pkg.DesAuth, c001v001.HandleGetDeviceList)
		router.Get("/ws", pkg.DesAuth, websocket.New(
			(&c001v001.DeviceUserClient{}).WSDeviceUserClient_Connect,
		))
	})

	/* C001V001 JOB ROUTES */
	api.Route("/001/001/job", func(router fiber.Router) {
		// router.Post("/config", pkg.DesAuth, (&c001v001.Job{}).Configure)
		router.Get("/event/list", c001v001.HandleGetEventTypeLists)
	})

	// /* C001V001 DEMO ROUTES */
	// api.Route("/001/001/demo", func(router fiber.Router) {
	// 	router.Get("/sim", pkg.DesAuth, websocket.New(
	// 		(&c001v001.DemoDeviceClient{}).WSDemoDeviceClient_Connect,
	// 	))
	// })

	api.All("*", func(c *fiber.Ctx) error {
		path := c.Path()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("Path: %v does not exists on this server", path),
		})
	})

	log.Fatal(app.Listen("127.0.0.1:8007"))
}

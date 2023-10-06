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
	// demoQty := flag.Int("demos", 5, "Create n demo devices if there are none currently") /* DEMO -> NOT FOR PRODUCTION */
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

	// test1, err := pkg.SearchDESJobsByToken("4")
	// if err != nil {
	// 	pkg.TraceErr(err)
	// }
	// pkg.Json("main -> SearchDESJobsByToken( )", test1)

	// test2, err := pkg.SearchDESJobsByRegion(-110.9, -110.7, 52.5, 52.8)
	// test2, err := pkg.SearchDESJobsByRegion(-180, 180, -90, 90)
	// if err != nil {
	// 	pkg.TraceErr(err)
	// }
	// pkg.Json("main -> SearchDESJobsByRegion( )", test2)

	// test3, err := pkg.SearchDESJobs("3", -180, 180, -90, 90)
	// test3, err := pkg.SearchDESJobs("3", -110.9, -110.7, 52.5, 52.8)
	// if err != nil {
	// 	pkg.TraceErr(err)
	// }
	// pkg.Json("main -> SearchDESJobs( )", test3)

	// params := pkg.DESSearchParam{
	// 	Token:  "",
	// 	LngMin: -110.9,
	// 	LngMax: -110.7,
	// 	LatMin: 52.5,
	// 	LatMax: 52.8,
	// }
	
	// params := pkg.DESSearchParam{
	// 	Token:  "",
	// 	LngMin: -180,
	// 	LngMax: 180,
	// 	LatMin: -90,
	// 	LatMax: 90,
	// }
	// test4, err := pkg.SearchDESDevices(params)
	// if err != nil {
	// 	pkg.TraceErr(err)
	// }
	// pkg.Json("main -> SearchDESDevices( )", test4)

	// /********************************************************************************************/
	// /* DEMO DEVICES -> NOT FOR PRODUCTION */
	// fmt.Println("\n\nConnecting all C001V001 MQTT DemoDevice Clients...")
	// c001v001.DemoDeviceClient_ConnectAll(*demoQty)
	// defer c001v001.DemoDeviceClient_DisconnectAll()
	// /********************************************************************************************/

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

	/* AUTH & USER ROUTES */
	api.Route("/user", func(router fiber.Router) {
		router.Get("/list", pkg.GetUserList)
		router.Post("/signup", pkg.SignUpUser)
		router.Post("/login", pkg.SignInUser)
		router.Get("/me", pkg.DesAuth, pkg.GetMe)
		router.Get("/logout", pkg.DesAuth, pkg.LogoutUser)
	})

	/* C001V001 DEVICE ROUTES */
	api.Route("/001/001/device", func(router fiber.Router) {
		// router.Post("/register", pkg.DesAuth, c001v001.HandleRegisterDevice)
		router.Post("/start", pkg.DesAuth, c001v001.HandleStartJob)
		router.Post("/end", pkg.DesAuth, c001v001.HandleEndJob)
		router.Post("/admin", pkg.DesAuth, c001v001.HandleSetAdmin)
		router.Post("/header", pkg.DesAuth, c001v001.HandleSetHeader)
		router.Post("/config", pkg.DesAuth, c001v001.HandleSetConfig)
		router.Post("/search", c001v001.HandleSearchDevices)
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

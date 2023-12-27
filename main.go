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
		pkg.LogErr(err)
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
			AllowOrigins:     "https://vw1.data2desk.com, http://localhost:8080, http://localhost:5173",
			AllowHeaders:     "Origin, Content-Type, Accept, Authorization, Cache-Control",
			AllowMethods:     "GET, POST",
			AllowCredentials: true,
		}))

		/* DES ROUTES *************************************************************************************/
		/*DES AUTH & USER ROUTES */
		pkg.InitializeDESUserRoutes(app, api)

		/* DES DEVICE ROUTES */
		pkg.InitializeDESDeviceRoutes(app, api)
		/****************************************************************************************************/


		/* C001V001 ROUTES ******************************************************************************/
		/* C001V001 DEVICE ROUTES */
		c001v001.InitializeDeviceRoutes(app, api)

		/* C001V001 JOB / REPORTING ROUTES */
		c001v001.InitializeJobRoutes(app, api)
		/****************************************************************************************************/

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

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

	// "math/rand"
	"os"
	// "path/filepath"

	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/leehayford/des/pkg"
	"github.com/leehayford/des/pkg/c001v001"
)

func MakeDemoC001V001(serial, userID string) pkg.DESRegistration {

	t := time.Now().UTC().UnixMilli()
	/* CREATE DEMO DEVICES */
	des_dev := pkg.DESDev{
		DESDevRegTime:   t,
		DESDevRegAddr:   "DEMO",
		DESDevRegUserID: userID,
		DESDevRegApp:    "DEMO",
		DESDevSerial:    serial,
		DESDevVersion:   "001",
		DESDevClass:     "001",
	}
	pkg.DES.DB.Create(&des_dev)

	job := &c001v001.Job{
		DESRegistration: pkg.DESRegistration{
			DESDev: des_dev,
			DESJob: pkg.DESJob{
				DESJobRegTime:   t,
				DESJobRegAddr:   "DEMO",
				DESJobRegUserID: userID,
				DESJobRegApp:    "DEMO",

				DESJobName:  fmt.Sprintf("%s_0000000000000", serial),
				DESJobStart: 0,
				DESJobEnd:   0,
				DESJobLng:   -180, // -114.75 + rand.Float32() * ( -110.15 + 114.75 ),
				DESJobLat:   90,   // 51.85 + rand.Float32() * ( 54.35 - 51.85 ),
				DESJobDevID: des_dev.DESDevID,
			},
		},
	}
	job.Admins = []c001v001.Admin{(job).RegisterJob_Default_JobAdmin()}
	job.Headers = []c001v001.Header{(job).RegisterJob_Default_JobHeader()}
	job.Configs = []c001v001.Config{(job).RegisterJob_Default_JobConfig()}
	job.Events = []c001v001.Event{(job).RegisterJob_Default_JobEvent()}
	job.Samples = []c001v001.Sample{ { SmpTime: t, SmpJobName: job.DESJobName } }
	job.RegisterJob()

	demo := c001v001.DemoDeviceClient{
		Device: c001v001.Device{
			DESRegistration: job.DESRegistration,
			Job:             c001v001.Job{DESRegistration: job.DESRegistration},
		},
	}

	/* WRITE TO FLASH - JOB_0 */
	demo.WriteAdmToFlash(*job, job.Admins[0])
	demo.WriteHdrToFlash(*job, job.Headers[0])
	demo.WriteCfgToFlash(*job, job.Configs[0])
	demo.WriteEvtToFlash(*job, job.Events[0])
	demo.WriteEvtToFlash(*job, c001v001.Event{
		EvtTime:   time.Now().UTC().UnixMilli(),
		EvtAddr:   "DEMO",
		EvtUserID: userID,
		EvtApp:    "DEMO",

		EvtCode:  1,
		EvtTitle: "Intitial State",
		EvtMsg:   "End Job event to ensure this newly registered demo device is ready to start a new demo job.",
	})

	return job.DESRegistration
}

func DemoSimFlashTest() {

	cfg := c001v001.Config{
		CfgTime:   time.Now().UTC().UnixMilli(),
		CfgAddr:   "aaaabbbbccccddddeeeeffffgggghhhh",
		CfgUserID: "ef0589a4-5ad4-45ea-9575-5aaee0568b0c",
		CfgApp:    "aaaabbbbccccddddeeeeffffgggghhhh",

		/* JOB */
		CfgSCVD:     596.8, // m
		CfgSCVDMult: 10.5,  // kPa / m
		CfgSSPRate:  1.95,  // kPa / hour
		CfgSSPDur:   6.0,   // hour
		CfgHiSCVF:   201.4, //  L/min
		CfgFlowTog:  1.85,  // L/min

		/* VALVE */
		CfgVlvTgt: 2, // vent
		CfgVlvPos: 2, // vent

		/* OP PERIODS*/
		CfgOpSample: 1000, // millisecond
		CfgOpLog:    1000, // millisecond
		CfgOpTrans:  1000, // millisecond

		/* DIAG PERIODS */
		CfgDiagSample: 10000,  // millisecond
		CfgDiagLog:    100000, // millisecond
		CfgDiagTrans:  600000, // millisecond
	}

	cfgBytes := cfg.FilterCfgBytes()
	fmt.Printf("\ncfgBytes ( %d ) : %v\n", len(cfgBytes), cfgBytes)

	dir := fmt.Sprintf("demo/%s", "test")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		pkg.Trace(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/cfg.bin", dir), os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		pkg.Trace(err)
	}
	defer f.Close()

	_, err = f.Write(cfg.FilterCfgBytes())
	if err != nil {
		pkg.Trace(err)
	}

	f.Close()
}

func main() {

	if err := pkg.DES.CreateDESDatabase(false); err != nil {
		pkg.Trace(err)
	}
	pkg.DES.Connect()
	defer pkg.DES.Close()

	// /* DEMO DEVICES -> NOT FOR PRODUCTION */

	regs, err := c001v001.GetDemoDeviceList()
	if err != nil {
		pkg.Trace(err)
	}

	if len(regs) == 0 {
		user := pkg.User{}
		pkg.DES.DB.Last(&user)
		regs = append(regs, MakeDemoC001V001("DEMO000000", user.ID.String()))
		regs = append(regs, MakeDemoC001V001("DEMO000001", user.ID.String()))
		regs = append(regs, MakeDemoC001V001("DEMO000002", user.ID.String()))
		// regs = append(regs, MakeDemoC001V001("DEMO000003", user.ID.String()))
		// regs = append(regs, MakeDemoC001V001("DEMO000004", user.ID.String()))
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

		demo.GetDeviceStatus()
		go demo.Demo_Simulation(time.Now().UTC())

		c001v001.DemoDeviceClients[demo.DESDevSerial] = demo
		// d := c001v001.DemoDeviceClients[demo.DESDevSerial]
		// fmt.Printf("\nCached DemoDeviceClient %s, current event code: %d\n", d.DESDevSerial, d.EVT.EvtCode)
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

	/* C001V001 DEMO ROUTES */
	api.Route("/001/001/demo", func(router fiber.Router) {
		router.Get("/sim", pkg.DesAuth, websocket.New(
			(&c001v001.DemoDeviceClient{}).WSDemoDeviceClient_Connect,
		))
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

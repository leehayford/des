package c001v001

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/leehayford/des/pkg"
)

/*
	RETURNS THE LIST OF DEVICES REGISTERED TO THIS DES

ALOZNG WITH THE ACTIVE JOB FOR EACH DEVICE
IN THE FORM OF A DESRegistration
*/
func HandleGetDeviceList(c *fiber.Ctx) (err error) {

	fmt.Printf("\nHandleGetDeviceList( )\n")

	params := pkg.DESSearchParam{
		Token:  "",
		LngMin: -180,
		LngMax: 180,
		LatMin: -90,
		LatMax: 90,
	}

	// regs, err := GetDeviceList()
	regs, err := pkg.SearchDESDevices(params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("GetDesDevList(...) -> query failed:\n%s\n", err),
			"data":    fiber.Map{"regs": regs},
		})
	}
	// pkg.Json("GetDeviceList(): DESRegistrations", regs)

	// var wg sync.WaitGroup
	// wg.Add(len(regs)) // fmt.Printf("\nWait Group: %d\n", len(regs))

	devices := []Device{}
	for _, reg := range regs {
		// pkg.Json("HandleGetDeviceList( ) -> reg", reg)
		device := (&Device{}).ReadDevicesMap(reg.DESDevSerial)
		device.DESRegistration = reg
		devices = append(devices, device)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"devices": devices},
	})
}

func HandleSearchDevices(c *fiber.Ctx) (err error) {

	fmt.Printf("\nHandleSearchDevices( )\n")

	/* TODO: CHECK USER PERMISSION */

	/* PARSE AND VALIDATE REQUEST DATA */
	params := pkg.DESSearchParam{}
	if err = c.BodyParser(&params); err != nil {
		pkg.TraceErr(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	pkg.Json("HandleSearchDevices( )", params)

	/* SEARCH ACTIVE DEVICES BASED ON params */
	regs, err := pkg.SearchDESDevices(params)
	if err != nil {
		pkg.TraceErr(err)
	}

	devices := []Device{}
	for _, reg := range regs {
		pkg.Json("HandleSearchDevices( ) -> reg", reg)
		device := (&Device{}).ReadDevicesMap(reg.DESDevSerial)
		device.DESRegistration = reg
		devices = append(devices, device)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"devices": devices},
	})
}

/*
	USED WHEN DEVICE OPERATOR WEB CLIENTS WANT TO START A NEW JOB ON THIS DEVICE

SEND AN MQTT JOB ADMIN, HEADER, CONFIG, & EVENT TO THE DEVICE
UPON MQTT MESSAGE AT '.../CMD/EVENT, DEVICE CLIENT PERFORMS

	DES JOB REGISTRATION
	CLASS/VERSION SPECIFIC JOB START ACTIONS
*/
func HandleStartJob(c *fiber.Ctx) (err error) {
	fmt.Printf("\nHandleStartJob( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to start a job",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleStartJob(): -> c.BodyParser(&device) -> dev", device)

	/* SYNC DEVICE WITH DevicesMap */
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DESMQTTClient.WG = &sync.WaitGroup{}
	device.GetMappedClients()

	/* START NEW JOB
	MAKE ADM, HDR, CFG, EVT ( START JOB )
	ENSURE ADM, HDR, CFG, & EVT HAVE THE SAME TIME STAMP / SIGNATURE
	*/
	startTime := time.Now().UTC().UnixMilli()

	device.DESRegistration.DESJobRegTime = startTime
	// device.Job.DESRegistration = device.DESRegistration

	device.ADM.AdmTime = startTime
	device.ADM.AdmAddr = c.IP()
	device.ADM.AdmUserID = device.DESJobRegUserID
	device.ADM.AdmDefHost = pkg.MQTT_HOST
	device.ADM.AdmDefPort = pkg.MQTT_PORT
	device.ADM.AdmOpHost = pkg.MQTT_HOST
	device.ADM.AdmOpPort = pkg.MQTT_PORT
	device.ADM.AdmSerial = device.DESDevSerial
	// pkg.Json("HandleStartJob(): -> device.ADM", device.ADM)

	device.HDR.HdrTime = startTime
	device.HDR.HdrAddr = c.IP()
	device.HDR.HdrUserID = device.DESJobRegUserID
	device.HDR.HdrJobName = device.CmdArchiveName()
	device.HDR.HdrJobStart = startTime // This is displays the time/date of the request while pending
	device.HDR.HdrJobEnd = -1          // This means there is a pending request for the device to start a new job
	device.HDR.HdrGeoLng = -180
	device.HDR.HdrGeoLat = 90
	// pkg.Json("HandleStartJob(): -> device.HDR", device.HDR)

	device.CFG.CfgTime = startTime
	device.CFG.CfgAddr = c.IP()
	device.CFG.CfgUserID = device.DESJobRegUserID
	device.ValidateCFG()
	// pkg.Json("HandleStartJob(): -> device.CFG", device.CFG)

	device.EVT = Event{
		EvtTime:   startTime,
		EvtAddr:   c.IP(),
		EvtUserID: device.DESJobRegUserID,
		EvtApp:    device.DESJobRegApp,
		EvtCode:   STATUS_JOB_START_REQ,
		EvtTitle:  "Job Start Request",
		EvtMsg:    "Job start sequence initiated.",
	}

	/* LOG START JOB REQUEST TO CMDARCHIVE */
	device.CmdDBC.Create(&device.ADM)
	device.CmdDBC.Create(&device.HDR)
	device.CmdDBC.Create(&device.CFG)
	device.CmdDBC.Create(&device.EVT)

	// /* MQTT PUB CMD: ADM, HDR, CFG, EVT */
	fmt.Printf("\nHandleStartJob( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDAdmin(device.ADM)
	device.MQTTPublication_DeviceClient_CMDHeader(device.HDR)
	device.MQTTPublication_DeviceClient_CMDConfig(device.CFG)
	device.MQTTPublication_DeviceClient_CMDEvent(device.EVT)

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, device)
	// Devices[device.DESDevSerial] = device

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 Job Start Reqest sent to device.",
	})
}

/*
	USED WHEN DEVICE OPERATOR WEB CLIENTS WANT TO END A JOB ON THIS DEVICE

SEND AN MQTT END JOB EVENT TO THE DEVICE
UPON MQTT MESSAGE AT '.../CMD/EVENT, DEVICE CLIENT PERFORMS

	DES JOB REGISTRATION ( UPDATE CMDARCHIVE START DATE )
	CLASS/VERSION SPECIFIC JOB END ACTIONS
*/
func HandleEndJob(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleEndtJob( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to end a job",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("(dev *Device) HandleEndJob(): -> c.BodyParser(&device) -> dev", device)

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.GetMappedHDR()
	device.GetMappedCFG()
	device.GetMappedSMP()
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DESMQTTClient.WG = &sync.WaitGroup{}
	device.GetMappedClients()

	device.EVT = Event{
		EvtTime:   time.Now().UTC().UnixMilli(),
		EvtAddr:   c.IP(),
		EvtUserID: device.DESJobRegUserID,
		EvtApp:    device.DESJobRegApp,
		EvtCode:   STATUS_JOB_END_REQ,
		EvtTitle:  "Job End Request",
		EvtMsg:    "Job end sequence initiated.",
	}

	/* LOG END JOB REQUEST TO CMDARCHIVE */ // fmt.Printf("\nHandleEndJob( ) -> Write to %s \n", device.CmdArchiveName())
	device.CmdDBC.Create(&device.EVT)

	/* LOG END JOB REQUEST TO ACTIVE JOB */ // fmt.Printf("\nHandleEndJob( ) -> Write to %s \n", device.DESJobName)
	device.JobDBC.Create(&device.EVT)

	/* MQTT PUB CMD: EVT */
	fmt.Printf("\nHandleEndJob( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDEvent(device.EVT)

	/* UPDATE THE DEVICES CLIENT MAP */
	UpdateDevicesMap(device.DESDevSerial, device)
	// Devices[device.DESDevSerial] = device

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 Job End Reqest sent to device.",
	})
}

/*
	USED TO ALTER THE ADMIN SETTINGS FOR A GIVEN DEVICE

BOTH DURING A JOB OR WHEN SENT TO CMDARCHIVE, TO ALTER THE DEVICE DEFAULTS
*/
func HandleSetAdmin(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleSetAdmin( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to alter device administration data.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleSetAdmin(): -> c.BodyParser(&device) -> device.ADM", device.ADM)

	/* SYNC DEVICE WITH DevicesMap */
	device.ADM.AdmTime = time.Now().UTC().UnixMilli()
	device.ADM.AdmAddr = c.IP()
	device.GetMappedHDR()
	device.GetMappedCFG()
	device.GetMappedEVT()
	device.GetMappedSMP()
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DESMQTTClient.WG = &sync.WaitGroup{}
	device.GetMappedClients()

	/* LOG ADM CHANGE REQUEST TO  CMDARCHIVE */
	device.CmdDBC.Create(device.ADM)

	/* MQTT PUB CMD: ADM */
	fmt.Printf("\nHandleSetConfig( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDAdmin(device.ADM)

	/* UPDATE DevicesMap */
	UpdateDevicesMap(device.DESDevSerial, device)
	// Devices[device.DESDevSerial] = device

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 SET ADMIN Reqest sent to device.",
	})
}

/*
	USED TO ALTER THE HEADER SETTINGS FOR A GIVEN DEVICE

BOTH DURING A JOB OR WHEN SENT TO CMDARCHIVE, TO ALTER THE DEVICE DEFAULTS
*/
func HandleSetHeader(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleSetHeader( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to alter job header data.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleSetHeader(): -> c.BodyParser(&device) -> device.HDR", device.HDR)

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.HDR.HdrTime = time.Now().UTC().UnixMilli()
	device.HDR.HdrAddr = c.IP()
	device.GetMappedCFG()
	device.GetMappedEVT()
	device.GetMappedSMP()
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DESMQTTClient.WG = &sync.WaitGroup{}
	device.GetMappedClients()

	/* LOG HDR CHANGE REQUEST TO CMDARCHIVE */
	device.CmdDBC.Create(device.HDR)

	/* MQTT PUB CMD: HDR */
	fmt.Printf("\nHandleSetConfig( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDHeader(device.HDR)

	/* UPDATE DevicesMap */
	UpdateDevicesMap(device.DESDevSerial, device)
	// Devices[device.DESDevSerial] = device

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 SET HEADER Reqest sent to device.",
	})
}

/*
	USED TO ALTER THE CONFIG SETTINGS FOR A GIVEN DEVICE

BOTH DURING A JOB OR WHEN SENT TO CMDARCHIVE, TO ALTER THE DEVICE DEFAULTS
*/
func HandleSetConfig(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleSetConfig( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to alter job configuration data.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleSetConfig(): -> c.BodyParser(&device) -> device.CFG", device.CFG)

	/* SYNC DEVICE WITH DevicesMap */
	device.GetMappedADM()
	device.GetMappedHDR()
	device.CFG.CfgTime = time.Now().UTC().UnixMilli()
	device.CFG.CfgAddr = c.IP()
	device.ValidateCFG()
	device.GetMappedEVT()
	device.GetMappedSMP()
	device.DESMQTTClient = pkg.DESMQTTClient{}
	device.DESMQTTClient.WG = &sync.WaitGroup{}
	device.GetMappedClients()

	/* LOG CFG CHANGE REQUEST TO CMDARCHIVE */
	device.CmdDBC.Create(device.CFG)

	/* MQTT PUB CMD: CFG */
	fmt.Printf("\nHandleSetConfig( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDConfig(device.CFG)

	/* UPDATE DevicesMap */
	UpdateDevicesMap(device.DESDevSerial, device)
	// Devices[device.DESDevSerial] = device

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 SET CONFIG Reqest sent to device.",
	})
}

/* TODO: TEST *** DO NOT USE ***
	USED WHEN DATACAN ADMIN WEB CLIENTS REGISTER NEW C001V001 DEVICES ON THIS DES

PERFORMS DES DEVICE REGISTRATION
PERFORMS CLASS/VERSION SPECIFIC REGISTRATION ACTIONS
*/
// func (dev *Device) HandleRegisterDevice(c *fiber.Ctx) (err error) {

// 	role := c.Locals("role")
// 	if role != "admin" {
// 		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
// 			"status":  "fail",
// 			"message": "You must be an administrator to register devices",
// 		})
// 	}

// 	reg := pkg.DESRegistration{}
// 	if err = c.BodyParser(&reg); err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 			"status":  "fail",
// 			"message": err.Error(),
// 		})
// 	}

// 	/*
// 		CREATE A DEVICE RECORD IN THE DES DB FOR THIS DEVICE
// 		 - Creates a new DESevice in the DES database
// 		 - Gets the C001V001Device's DeviceID from the DES Database
// 	*/
// 	reg.DESDevRegTime = time.Now().UTC().UnixMilli()
// 	reg.DESDevRegAddr = c.IP()
// 	if device_res := pkg.DES.DB.Create(&reg.DESDev); device_res.Error != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"status":  "fail",
// 			"message": device_res.Error.Error(),
// 		})
// 	}

// 	/*
// 		CREATE THE DEFAULT JOB FOR THIS DEVICE
// 	*/
// 	job := Job{
// 		DESRegistration: pkg.DESRegistration{
// 			DESDev: reg.DESDev,
// 			DESJob: pkg.DESJob{
// 				DESJobRegTime:   reg.DESDevRegTime,
// 				DESJobRegAddr:   reg.DESDevRegAddr,
// 				DESJobRegUserID: reg.DESDevRegUserID,
// 				DESJobRegApp:    reg.DESDevRegApp,

// 				DESJobName:  fmt.Sprintf("%s_0000000000000", reg.DESDevSerial),
// 				DESJobStart: 0,
// 				DESJobEnd:   0,
// 				DESJobLng:   reg.DESJobLng,
// 				DESJobLat:   reg.DESJobLat,
// 				DESJobDevID: reg.DESDevID,
// 			},
// 		},
// 		Admins:  []Admin{(&Job{}).RegisterJob_Default_JobAdmin()},
// 		Headers: []Header{(&Job{}).RegisterJob_Default_JobHeader()},
// 		Configs: []Config{(&Job{}).RegisterJob_Default_JobConfig()},
// 		Events:  []Event{(&Job{}).RegisterJob_Default_JobEvent()},
// 	}
// 	if err = job.RegisterJob(); err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"status":  "fail",
// 			"message": err.Error(),
// 		})
// 	}

// 	reg = pkg.DESRegistration{
// 		DESDev: reg.DESDev,
// 		DESJob: job.DESJob,
// 	}

// 	device := Device{
// 		DESRegistration: reg,
// 		Job:             Job{DESRegistration: reg},
// 		DESMQTTClient:   pkg.DESMQTTClient{},
// 	}
// 	if err = device.MQTTDeviceClient_Connect(); err != nil {
// 		return pkg.TraceErr(err)
// 	}

// 	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
// 		"status":  "success",
// 		"data":    fiber.Map{"device": &reg},
// 		"message": "C001V001 Device Registered.",
// 	})
// }

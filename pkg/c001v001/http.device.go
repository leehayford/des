package c001v001

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/leehayford/des/pkg"
)

/* NOT TESTED
	RETURNS THE LIST OF DEVICES REGISTERED TO THIS DES

	ALONG WITH THE ACTIVE JOB FOR EACH DEVICE
	IN THE FORM OF A DESRegistration
*/
func HandleGetDeviceList(c *fiber.Ctx) (err error) {

	fmt.Printf("\nHandleGetDeviceList( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to view device list",
		})
	}

	regs, err := GetDeviceList()
	if err != nil {
		pkg.TraceErr(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("GetDeviceList(...) -> query failed:\n%s\n", err),
			"data":    fiber.Map{"regs": regs},
		})
	}

	devices := GetDevices(regs)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"devices": devices},
	})
}

/* NOT TESTED --> CURRENTLY HANDLED ON FRONT END...*/
func HandleSearchDevices(c *fiber.Ctx) (err error) {

	fmt.Printf("\nHandleSearchDevices( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to search devices",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	params := pkg.DESSearchParam{}
	if err = c.BodyParser(&params); err != nil {
		pkg.TraceErr(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleSearchDevices( )", params)

	/* SEARCH ACTIVE DEVICES BASED ON params */
	regs, err := pkg.SearchDESDevices(params)
	if err != nil {
		pkg.TraceErr(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("pkg.SearchDESDevices(...) -> query failed:\n%s\n", err),
			"data":    fiber.Map{"regs": regs},
		})
	}

	devices := GetDevices(regs)

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
	} // pkg.Json("HandleStartJob(): -> c.BodyParser(&device) -> device", device)

	/* SEND START JOB REQUEST */
	if err = device.StartJobRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleStartJob(): -> device.StartJobRequest(...) -> device", device)

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

	/* SEND END JOB REQUEST */
	if err = device.EndJobRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleStartJob(): -> device.EndJobRequest(...) -> device", device)

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

	/* SEND SET ADMIN REQUEST */
	if err = device.SetAdminRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleSetAdmin(): -> device.SetAdminRequest(...) -> device.ADM", device.ADM)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 SET ADMIN Reqest sent to device.",
	})
}
/* TODO: DO NOT USE 
TEST EVENT DRIVEN STATUS VS .../cmd/topic/report DRIVEN STATUS 
*/
func  HandleGetAdmin(c *fiber.Ctx) (err error) {

	fmt.Printf("\nHandleGetAdmin( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to see device administration data.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} 
	pkg.Json("HandleGetAdmin(): -> c.BodyParser(&device) -> device.ADM", device.ADM)

	/* SEND GET ADMIN REQUEST */
	if err = device.GetAdminRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleGetAdmin(): -> device.GetAdminRequest(...) -> device", device)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 GET ADMIN Reqest sent to device.",
	})
}

/* TODO: DO NOT USE 
TEST EVENT DRIVEN STATUS VS .../cmd/topic/report DRIVEN STATUS 
*/
func  HandleGetState(c *fiber.Ctx) (err error) {

	fmt.Printf("\nHandleGetState( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to see device hardware ID data.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} 
	pkg.Json("HandleGetState(): -> c.BodyParser(&device) -> device.STA", device.STA)

	/* SEND GET STATE REQUEST */
	if err = device.GetHwIDRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleGetState(): -> device.GetStateRequest(...) -> device", device)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 GET STATE Reqest sent to device.",
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

	/* SEND SET HEADER REQUEST */
	if err = device.SetHeaderRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}  // pkg.Json("HandleSetHeader(): -> device.SetHeaderRequest(...) -> device.HDR", device.HDR)

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

	/* SEND SET CONFIG REQUEST */
	if err = device.SetConfigRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleSetConfig(): -> device.SetConfigRequest(...) -> device.CFG", device.CFG)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 SET CONFIG Reqest sent to device.",
	})
}


/*
	USED TO CREATE AN EVENT FOR A GIVEN DEVICE, BOTH: 
	- DURING A JOB AND 
	- TO MAKE NOTE OF NON-JOB SPECIFIC ... STUFF ( MAINTENANCE ETC. ) 
*/
func HandleCreateDeviceEvent(c *fiber.Ctx) (err error) {
	fmt.Printf("\nHandleCreateDeviceEvent( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to create Events.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleCreateDeviceEvent( ): -> c.BodyParser(&device) -> device.EVT", device.EVT)

	/* SEND CREATE EVENT REQUEST */
	if err = device.CreateEventRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleCreateDeviceEvent( ): -> device.CreateEventRequest(...) -> device.EVT", device.EVT)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 CREATE EVENT REQUEST sent to device.",
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

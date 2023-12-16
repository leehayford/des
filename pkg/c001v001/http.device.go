package c001v001

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	"github.com/leehayford/des/pkg"
)

func InitializeDeviceRoutes(app, api *fiber.App) {
	api.Route("/001/001/device", func(router fiber.Router) {

		/* DEVICE-ADMIN-LEVEL OPERATIONS */
		router.Post("/register", pkg.DesAuth, HandleRegisterDevice)
		router.Post("/check_des_conn", pkg.DesAuth, HandleCheckDESDeviceClient)
		router.Post("/disconnect", pkg.DesAuth, HandleDisconnectDevice)

		/* DEVICE-OPERATOR-LEVEL OPERATIONS */
		router.Post("/start", pkg.DesAuth, HandleStartJob)
		router.Post("/end", pkg.DesAuth, HandleEndJob)
		router.Post("/admin", pkg.DesAuth, HandleSetAdmin)
		router.Post("/state", pkg.DesAuth, HandleSetState)
		router.Post("/header", pkg.DesAuth, HandleSetHeader)
		router.Post("/config", pkg.DesAuth, HandleSetConfig)
		router.Post("/event", pkg.DesAuth, HandleCreateDeviceEvent)

		router.Post("/debug", pkg.DesAuth, HandleSetDebug)
		router.Post("/msg_limit", pkg.DesAuth, HandleTestMessageLimit)
		router.Post("/sim_offline_start", pkg.DesAuth, HandleSimOfflineStart)

		/* DEVICE-VIEWER-LEVEL OPERATIONS */
		router.Post("/job_events", pkg.DesAuth, HandleGetActiveJobEvents)
		router.Post("/search", pkg.DesAuth, HandleSearchDevices)
		router.Get("/list", pkg.DesAuth, HandleGetDeviceList)

		/* TODO: ROLES HANDLED PER MQTT TOPIC / WS */
		app.Use("/ws", pkg.HandleWSUpgrade)
		router.Get("/ws", pkg.DesAuth, websocket.New(HandleDeviceUserClient_Connect))

	})
}

/*
	NOT TESTED

# RETURNS THE LIST OF DEVICES REGISTERED TO THIS DES

ALONG WITH THE ACTIVE JOB FOR EACH DEVICE
IN THE FORM OF A DESRegistration
*/
func HandleGetDeviceList(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleGetDeviceList( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an operator to view device list",
		})
	}

	regs, err := GetDeviceList()
	if err != nil {
		pkg.LogErr(err)
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
	// fmt.Printf("\nHandleSearchDevices( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an operator to search devices",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	params := pkg.DESSearchParam{}
	if err = c.BodyParser(&params); err != nil {
		pkg.LogErr(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleSearchDevices( )", params)

	/* SEARCH ACTIVE DEVICES BASED ON params */
	regs, err := pkg.SearchDESDevices(params)
	if err != nil {
		pkg.LogErr(err)
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
	fmt.Printf("\nHandleStartJobX( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an operator to start a job",
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

	/* TODO : MOVE TO DES, CREATE CUSTOM Status ?
	CHECK DEVICE AVAILABILITY */
	if ok := DevicePingsMapRead(device.DESDevSerial).OK; !ok {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "Device not connected to broker",
		})
	} // pkg.Json("HandleStartJob(): -> device.CheckPing( ) -> device", device)

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
		"message": "C001V001 Job Start Request sent to device.",
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
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an operator to end a job",
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
USED WHEN THE DES NEEDS TO AQUIRE THE LATES MODELS
- EX: WHERE A DEVICE HAS STARTED A JOB AND THERE IS NO DATABASE REGISTERED
*/
func HandleReportRequest(c *fiber.Ctx) (err error) {

	return
}

/*
	USED TO ALTER THE ADMIN SETTINGS FOR A GIVEN DEVICE

BOTH DURING A JOB OR WHEN SENT TO CMDARCHIVE, TO ALTER THE DEVICE DEFAULTS
*/
func HandleSetAdmin(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleSetAdmin( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an operator to alter device administration data.",
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

/*
USED TO SET THE STATE VALUES FOR A GIVEN DEVICE
***NOTE***

THE STATE IS A READ ONLY STRUCTURE AT THIS TIME
FUTURE VERSIONS WILL ALLOW DEVICE ADMINISTRATORS TO ALTER SOME STATE VALUES REMOTELY
CURRENTLY THIS HANDLER IS USED ONLY TO REQUEST THE CURRENT DEVICE STATE
*/
func HandleSetState(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleSetState( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an operator to see device hardware ID data.",
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
	pkg.Json("HandleSetState(): -> c.BodyParser(&device) -> device.STA", device.STA)

	/* SEND GET STATE REQUEST */
	if err = device.SetStateRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleSetState(): -> device.SetStateRequest(...) -> device", device)

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
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an operator to alter job header data.",
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
	} // pkg.Json("HandleSetHeader(): -> device.SetHeaderRequest(...) -> device.HDR", device.HDR)

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
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an operator to alter job configuration data.",
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
	USED TO CREATE AN EVENT FOR A GIVEN DEVICE

BOTH DURING A JOB AND TO MAKE NOTE OF NON-JOB SPECIFIC ... STUFF ( MAINTENANCE ETC. )
*/
func HandleCreateDeviceEvent(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleCreateDeviceEvent( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an operator to create Events.",
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

func HandleGetActiveJobEvents(c *fiber.Ctx) (err error) {
	fmt.Printf("\nHandleGetActiveJobEvents( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Viewer(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be a registered user to view job evens.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleGetActiveJobEvents(): -> c.BodyParser(&device) -> device", device)

	evts, err := device.GetActiveJobEvents()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleGetActiveJobEvents(): -> device.GetActiveJobEvents() -> evts", evts)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"events": &evts},
	})
}

/* USED TO OPEN A WEB SOCKET CONNECTION BETWEEN A USER AND A GIVEN DEVICE */
func HandleDeviceUserClient_Connect(c *websocket.Conn) {
	fmt.Printf("\nWSDeviceUserClient_Connect( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != pkg.ROLE_ADMIN && role != pkg.ROLE_OPERATOR && role != pkg.ROLE_USER {
		/* CREATE JSON WSMessage STRUCT */
		res := AuthResponse{Status: "fail", Message: "You need permission to watch a live feed."}
		js, err := json.Marshal(&WSMessage{Type: "auth", Data: res})
		if err != nil {
			pkg.LogErr(err)
			return
		}
		c.Conn.WriteJSON(string(js))
		return
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	url_str, _ := url.QueryUnescape(c.Query("device"))
	if err := json.Unmarshal([]byte(url_str), &device); err != nil {
		pkg.LogErr(err)
	}

	/* CONNECTED DEVICE USER CLIENT *** DO NOT RUN IN GO ROUTINE *** */
	duc := DeviceUserClient{Device: device}
	duc.DeviceUserClient_Connect(c)
}


/**************************************************************************************************************/
/* DES ADMINISTRATION ************************************************************************************/
/**************************************************************************************************************/

/*
	 TODO: TEST *** DO NOT USE ***
		USED WHEN DATACAN ADMIN WEB CLIENTS REGISTER NEW C001V001 DEVICES ON THIS DES

PERFORMS DES DEVICE REGISTRATION
PERFORMS CLASS/VERSION SPECIFIC REGISTRATION ACTIONS
*/
func HandleRegisterDevice(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleRegisterDevice( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Admin(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to register devices",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	reg := pkg.DESRegistration{}
	if err = c.BodyParser(&reg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleRegisterDevice( ) -> c.BodyParser( reg ) -> reg", reg)
	
	/* TODO: VALIDATE SERIAL# ( TO UPPER ): 
			!= ""
			DOESN'T ALREADY EXIST
			LENGTH < 10
	*/

	/* REGISTER A C001V001 DEVICE ON THIS DES */
	device := &Device{}
	if err := device.RegisterDevice(c.IP(), reg); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": fmt.Sprintf("%s Registered.", device.DESDevSerial),
		"data":    fiber.Map{"device": &device},
	})
}

func HandleDisconnectDevice(c *fiber.Ctx) (err error) {
	fmt.Printf("\nHandleDisconnectDevice( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Admin(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to view disconnect devices.",
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
	pkg.Json("HandleDisconnectDevice(): -> c.BodyParser(&device) -> device", device)

	d := DevicesMapRead(device.DESDevSerial)

	/* CLOSE DEVICE CLIENT CONNECTIONS */
	if err = d.DeviceClient_Disconnect(); err != nil {
		msg := fmt.Sprintf(
			"Failed to close existing device connectsions for %s\n%s\n",
			device.DESDevSerial,
			err.Error(),
		)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": msg,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": fmt.Sprintf("%s DES client disconnected.", device.DESDevSerial),
		"data":    fiber.Map{"device": &device},
	})
}

func HandleCheckDESDeviceClient(c *fiber.Ctx) (err error) {
	fmt.Printf("\nHandleCheckDESDeviceClient( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Admin(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to connect devices.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} //pkg.Json("HandleCheckDESDeviceClient(): -> c.BodyParser(&device) -> device", device)

	/* GET / VALIDATE DESRegistration */
	ser := device.DESDevSerial
	if err = device.GetDeviceDESRegistration(ser); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("DES Registration for %s was not found.\n%s\nDB ERROR", ser, err.Error()),
		})
	} // pkg.Json("HandleCheckDESDeviceClient(): -> device.GetDeviceDESRegistration -> device", device)

	d := DevicesMapRead(device.DESDevSerial)

	/* CLOSE ANY EXISTING CONNECTIONS AND RECONNECT THE DES DEVICE CLIENTS */
	if err = d.DeviceClient_RefreshConnections(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("Connections for %s could not be refreshed; ERROR:\n%s\n", ser, err.Error()),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": fmt.Sprintf("%s DES device client connected.", d.DESDevSerial),
		"data":    fiber.Map{"device": &d},
	})
}


/**************************************************************************************************************/
/* DEBUGGING STUFF :  REMOVE FOR PRODUCTION *******************************************************/
/**************************************************************************************************************/

/*
	USED TO ALTER THE DEBUG SETTINGS FOR A GIVEN DEVICE

THIS INFORMATION IS NOT LOGGED TO THE DATABASE OR SENT TO THE PHYSICAL DEVICE
*/
func HandleSetDebug(c *fiber.Ctx) (err error) {
	fmt.Printf("\nHandleSetDebug( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Admin(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to change debug settings.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleSetDebug(): -> c.BodyParser(&device) -> device", device)

	/* UPDATE THE MAPPED DES DEVICE DBG */
	if err := device.SetDebug(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	pkg.Json("HandleSetDebug(): ->device.SetDebugRequest() -> device.DBG", device.DBG)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Debug settings updated.",
		"data":    fiber.Map{"device": &device},
	})
}

type MsgLimit struct {
	Kafka string `json:"kafka"`
}

func HandleTestMessageLimit(c *fiber.Ctx) (err error) {
	fmt.Printf("\nHandleTestMessageLimit( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Admin(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to change debug settings.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := &Device{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleTestMessageLimit(): -> c.BodyParser(&device) -> device", device)

	length, err := device.TestMsgLimit()

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": fmt.Sprintf("%d byte message sent.", length),
		"data":    fiber.Map{"device": &device},
	})
}

func HandleSimOfflineStart(c *fiber.Ctx) (err error) {
	fmt.Printf("\nHandleSimOfflineStart( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Admin(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to change debug settings.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := &Device{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // pkg.Json("HandleTestMessageLimit(): -> c.BodyParser(&device) -> device", device)

	device.GetMappedClients()
	device.MQTTPublication_DeviceClient_CMDTestOLS()
	// device.GetDeviceDESU()

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Offline job start simmulation running.",
		"data":    fiber.Map{"device.DESU": &device.DESU},
	})
}

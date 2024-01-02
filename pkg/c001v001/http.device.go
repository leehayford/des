package c001v001

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	"github.com/leehayford/des/pkg"
)

func InitializeDeviceRoutes(app, api *fiber.App) {
	api.Route("/001/001/device", func(router fiber.Router) {

		/* DEVICE-ADMIN-LEVEL OPERATIONS */
		router.Post("/register", pkg.DesAuth, HandleRegisterDevice)
		router.Post("/des_client_refresh", pkg.DesAuth, HandleDESDeviceClientRefresh)
		router.Post("/des_client_disconnect", pkg.DesAuth, HandleDESDeviceClientDisconnect)

		/* DEVICE-OPERATOR-LEVEL OPERATIONS */
		router.Post("/start", pkg.DesAuth, HandleStartJobRequest)
		router.Post("/end", pkg.DesAuth, HandleEndJobRequest)
		router.Post("/admin", pkg.DesAuth, HandleSetAdminRequest)
		router.Post("/state", pkg.DesAuth, HandleSetStateRequest)
		router.Post("/header", pkg.DesAuth, HandleSetHeaderRequest)
		router.Post("/config", pkg.DesAuth, HandleSetConfigRequest)
		router.Post("/event", pkg.DesAuth, HandleSetEventRequest)

		/* DEVICE-VIEWER-LEVEL OPERATIONS */
		router.Post("/job_events", pkg.DesAuth, HandleQryActiveJobEvents)
		router.Post("/job_samples", pkg.DesAuth, HandleQryActiveJobSamples)
		router.Post("/search", pkg.DesAuth, HandleSearchDevices)
		router.Get("/list", pkg.DesAuth, HandleGetDeviceList)

		/* TODO: ROLES HANDLED PER MQTT TOPIC / WS */
		app.Use("/ws", pkg.HandleWSUpgrade)
		router.Get("/ws", pkg.DesAuth, websocket.New(HandleDeviceUserClient_Connect))

		/* DEVELOPMENT *** NOT FOR PRODUCTION *** */
		router.Post("/debug", pkg.DesAuth, HandleSetDebug)
		router.Post("/msg_limit", pkg.DesAuth, HandleTestMessageLimit)
		router.Post("/sim_offline_start", pkg.DesAuth, HandleSimOfflineStart)
	})
}

func ValidatePostRequestBody_Device(c *fiber.Ctx, device *Device) (err error) {

	if err = pkg.ParseRequestBody(c, &device); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	/*  TODO: ADDITIONAL DEVICE VALIDATION */
	return
}
func ValidateDeviceMsgSourcePOST(c *fiber.Ctx, src *pkg.DESMessageSource) (err error) {

	return
}
func ValidateDeviceMsgSourceSIG(c *fiber.Ctx, src *pkg.DESMessageSource) (err error) {

	return
}

/*
	RETURNS THE LIST OF DEVICES REGISTERED TO THIS DES

ALONG WITH THE ACTIVE JOB FOR EACH DEVICE
IN THE FORM OF A DESRegistration
*/
func HandleGetDeviceList(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleGetDeviceList( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_OPERATOR + ": View device list")
	}

	regs, err := GetDeviceList()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	devices := GetDevices(regs)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"devices": devices})
}

/* NOT TESTED --> CURRENTLY HANDLED ON FRONT END...*/
func HandleSearchDevices(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleSearchDevices( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_OPERATOR + ": Search devices")
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	params := pkg.DESSearchParam{}
	if err = pkg.ParseRequestBody(c, &params); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	/* SEARCH ACTIVE DEVICES BASED ON params */
	regs, err := pkg.SearchDESDevices(params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	devices := GetDevices(regs)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"devices": devices})
}

/*
	USED WHEN DEVICE OPERATOR WEB CLIENTS WANT TO START A NEW JOB ON THIS DEVICE

SEND AN MQTT JOB ADMIN, HEADER, CONFIG, & EVENT TO THE DEVICE
UPON MQTT MESSAGE AT '.../CMD/EVENT, DEVICE CLIENT PERFORMS

	DES JOB REGISTRATION
	CLASS/VERSION SPECIFIC JOB START ACTIONS
*/
func HandleStartJobRequest(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleStartJobRequest( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_OPERATOR + ": Start a job")
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = ValidatePostRequestBody_Device(c, &device); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	} // pkg.Json("HandleStartJobRequest(): -> c.BodyParser(&device) -> device", device)

	/* CHECK DEVICE AVAILABILITY */
	if ok := DevicePingsMapRead(device.DESDevSerial).OK; !ok {
		return c.Status(fiber.StatusBadRequest).SendString(pkg.ERR_MQTT_DEVICE_CONN)
	} 

	/* SEND START JOB REQUEST */
	uid := (c.Locals("sub").(string))
	if err = device.StartJobRequest(c.IP(), uid); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	// pkg.Json("HandleStartJobRequest(): -> device.StartJobRequest(...) -> device", device)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"device": &device})
}

/*
	USED WHEN DEVICE OPERATOR WEB CLIENTS WANT TO END A JOB ON THIS DEVICE

SEND AN MQTT END JOB EVENT TO THE DEVICE
UPON MQTT MESSAGE AT '.../CMD/EVENT, DEVICE CLIENT PERFORMS

	DES JOB REGISTRATION ( UPDATE CMDARCHIVE START DATE )
	CLASS/VERSION SPECIFIC JOB END ACTIONS
*/
func HandleEndJobRequest(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleEndJobRequest( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_OPERATOR + ": End a job")
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = ValidatePostRequestBody_Device(c, &device); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	} // pkg.Json("HandleEndJobRequest(): -> c.BodyParser(&device) -> dev", device)

	/* CHECK DEVICE AVAILABILITY */
	if ok := DevicePingsMapRead(device.DESDevSerial).OK; !ok {
		return c.Status(fiber.StatusBadRequest).SendString(pkg.ERR_MQTT_DEVICE_CONN)
	} 

	/* SEND END JOB REQUEST */
	uid := (c.Locals("sub").(string))
	if err = device.EndJobRequest(c.IP(), uid); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	} // pkg.Json("HandleEndJobRequest(): -> device.EndJobRequest(...) -> device", device)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"device": &device})
}

/*
	USED TO ALTER THE ADMIN SETTINGS FOR A GIVEN DEVICE

BOTH DURING A JOB OR WHEN SENT TO CMDARCHIVE, TO ALTER THE DEVICE DEFAULTS
*/
func HandleSetAdminRequest(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleSetAdminRequest( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_OPERATOR + ": Edit device alarms")
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = ValidatePostRequestBody_Device(c, &device); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	} // pkg.Json("HandleSetAdminRequest(): -> c.BodyParser(&device) -> device.ADM", device.ADM)

	/* CHECK DEVICE AVAILABILITY */
	if ok := DevicePingsMapRead(device.DESDevSerial).OK; !ok {
		return c.Status(fiber.StatusBadRequest).SendString(pkg.ERR_MQTT_DEVICE_CONN)
	} 

	/* SEND SET ADMIN REQUEST */
	if err = device.SetAdminRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	} // pkg.Json("HandleSetAdminRequest(): -> device.SetAdminRequest(...) -> device.ADM", device.ADM)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"device": &device})
}

/*
USED TO SET THE STATE VALUES FOR A GIVEN DEVICE
***NOTE***

THE STATE IS A READ ONLY STRUCTURE AT THIS TIME
FUTURE VERSIONS WILL ALLOW DEVICE ADMINISTRATORS TO ALTER SOME STATE VALUES REMOTELY
CURRENTLY THIS HANDLER IS USED ONLY TO REQUEST THE CURRENT DEVICE STATE
*/
func HandleSetStateRequest(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleSetStateRequest( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_OPERATOR + ": Request device state")
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = ValidatePostRequestBody_Device(c, &device); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	} // pkg.Json("HandleSetStateRequest(): -> c.BodyParser(&device) -> device.STA", device.STA)

	/* CHECK DEVICE AVAILABILITY */
	if ok := DevicePingsMapRead(device.DESDevSerial).OK; !ok {
		return c.Status(fiber.StatusBadRequest).SendString(pkg.ERR_MQTT_DEVICE_CONN)
	} 

	/* SEND GET STATE REQUEST */
	if err = device.SetStateRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	} // pkg.Json("HandleSetStateRequest(): -> device.SetStateRequest(...) -> device", device)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"device": &device})
}

/*
	USED TO ALTER THE HEADER SETTINGS FOR A GIVEN DEVICE

BOTH DURING A JOB OR WHEN SENT TO CMDARCHIVE, TO ALTER THE DEVICE DEFAULTS
*/
func HandleSetHeaderRequest(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleSetHeaderRequest( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_OPERATOR + ": Edit job headers")
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = ValidatePostRequestBody_Device(c, &device); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	} // pkg.Json("HandleSetHeaderRequest(): -> c.BodyParser(&device) -> device.HDR", device.HDR)

	/* CHECK DEVICE AVAILABILITY */
	if ok := DevicePingsMapRead(device.DESDevSerial).OK; !ok {
		return c.Status(fiber.StatusBadRequest).SendString(pkg.ERR_MQTT_DEVICE_CONN)
	} 

	/* SEND SET HEADER REQUEST */
	if err = device.SetHeaderRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	} // pkg.Json("HandleSetHeaderRequest(): -> device.SetHeaderRequest(...) -> device.HDR", device.HDR)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"device": &device})
}

/*
	USED TO ALTER THE CONFIG SETTINGS FOR A GIVEN DEVICE

BOTH DURING A JOB OR WHEN SENT TO CMDARCHIVE, TO ALTER THE DEVICE DEFAULTS
*/
func HandleSetConfigRequest(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleSetConfigRequest( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_OPERATOR + ": Edit device configuration")
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = ValidatePostRequestBody_Device(c, &device); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	} // pkg.Json("HandleSetConfigRequest(): -> c.BodyParser(&device) -> device.CFG", device.CFG)

	/* CHECK DEVICE AVAILABILITY */
	if ok := DevicePingsMapRead(device.DESDevSerial).OK; !ok {
		return c.Status(fiber.StatusBadRequest).SendString(pkg.ERR_MQTT_DEVICE_CONN)
	} 

	/* SEND SET CONFIG REQUEST */
	if err = device.SetConfigRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	} // pkg.Json("HandleSetConfigRequest(): -> device.SetConfigRequest(...) -> device.CFG", device.CFG)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"device": &device})
}

/*
	USED TO CREATE AN EVENT FOR A GIVEN DEVICE

BOTH DURING A JOB AND TO MAKE NOTE OF NON-JOB SPECIFIC ... STUFF ( MAINTENANCE ETC. )
*/
func HandleSetEventRequest(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleSetEventRequest( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Operator(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_OPERATOR + ": Post job events")
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = ValidatePostRequestBody_Device(c, &device); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	} // pkg.Json("HandleSetEventRequest( ): -> c.BodyParser(&device) -> device.EVT", device.EVT)

	/* CHECK DEVICE AVAILABILITY */
	if ok := DevicePingsMapRead(device.DESDevSerial).OK; !ok {
		return c.Status(fiber.StatusBadRequest).SendString(pkg.ERR_MQTT_DEVICE_CONN)
	} 

	/* SEND CREATE EVENT REQUEST */
	// uid := (c.Locals("sub").(string))
	if err = device.SetEventRequest(c.IP()); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	} // pkg.Json("HandleSetEventRequest( ): -> device.CreateEventRequest(...) -> device.EVT", device.EVT)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"device": &device})
}

func HandleQryActiveJobEvents(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleQryActiveJobEvents( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Viewer(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_VIEWER + ": View job events")
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = ValidatePostRequestBody_Device(c, &device); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	} // pkg.Json("HandleQryActiveJobEvents(): -> c.BodyParser(&device) -> device", device)

	evts, err := device.QryActiveJobEvents()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	} // pkg.Json("HandleQryActiveJobEvents(): -> device.QryActiveJobEvents() -> evts", evts)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"events": &evts})
}

func HandleQryActiveJobSamples(c *fiber.Ctx) (err error) {
	fmt.Printf("\nHandleQryActiveJobSamples( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Viewer(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_VIEWER + ": View sample data")
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = ValidatePostRequestBody_Device(c, &device); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	} // pkg.Json("HandleQryActiveJobSamples(): -> c.BodyParser(&device) -> device", device)

	/* PARSE AND VALIDATE REQUEST DATA - QUERY PARAMS */
	strQty, err := url.QueryUnescape(c.Query("qty"))
	if err != nil {
		txt := fmt.Sprintf("Invalid query parameter: %s", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(txt)
	}

	qty, err := strconv.Atoi(strQty)
	if err != nil {
		txt := fmt.Sprintf("Invalid query parameter: %s", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(txt)
	}

	xys, err := device.QryActiveJobXYSamples(qty)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	} // pkg.Json("HandleQryActiveJobSamples(): -> device.QryActiveJobSamples() -> len(smps)", len(smps))

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"xy_points": &xys})
}

/* USED TO OPEN A WEB SOCKET CONNECTION BETWEEN A USER AND A GIVEN DEVICE */
func HandleDeviceUserClient_Connect(ws *websocket.Conn) {
	// fmt.Printf("\nWSDeviceUserClient_Connect( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Viewer(ws.Locals("role")) {
		pkg.SendWSConnectionError(ws, pkg.ERR_AUTH_OPERATOR + ": Connect to devices.")
		return
	}

	/* PARSE AND VALIDATE REQUEST DATA - SESSION ID */
	sid, err := url.QueryUnescape(ws.Query("sid"))
	if err != nil {
		pkg.SendWSConnectionError(ws, err.Error())
		return
	}

	if !pkg.ValidateUUIDString(sid) {
		pkg.SendWSConnectionError(ws, pkg.ERR_AUTH_INVALID_SESSION)
		return
	}

	/* PARSE AND VALIDATE REQUEST DATA - DEVICE */
	device := Device{}
	url_str, _ := url.QueryUnescape(ws.Query("device"))
	if err := json.Unmarshal([]byte(url_str), &device); err != nil {
		pkg.LogErr(err)
	}

	/* CONNECTED DEVICE USER CLIENT *** DO NOT RUN IN GO ROUTINE *** */
	duc := DeviceUserClient{Device: device}
	duc.DeviceUserClient_Connect(ws, sid)
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
	if !pkg.UserRole_Super(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_SUPER + ": Register devices")
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = ValidatePostRequestBody_Device(c, &device); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	// if err := c.BodyParser(&device); err != nil {
	// 	txt := fmt.Sprintf("Invalid request body: %s", err.Error())
	// 	return c.Status(fiber.StatusBadRequest).SendString(txt)
	// } // pkg.Json("HandleRegisterDevice( ) -> c.BodyParser( device ) -> device.DESDev", device.DESDev)

	/* REGISTER A C001V001 DEVICE ON THIS DES */
	if err := device.RegisterDevice(c.IP()); err != nil {

		if strings.Contains(err.Error(), "Serial") {
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())

		} else {
			txt := fmt.Sprintf("Failed to register device: %s", err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString(txt)
		}

	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"device": &device})
}

func HandleDESDeviceClientDisconnect(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleDisconnectDevice( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Super(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_SUPER + ": Disconnect devices")
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = ValidatePostRequestBody_Device(c, &device); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	} // pkg.Json("HandleDisconnectDevice(): -> c.BodyParser(&device) -> device", device)

	d := DevicesMapRead(device.DESDevSerial)

	/* CLOSE DEVICE CLIENT CONNECTIONS */
	if err = d.DeviceClient_Disconnect(); err != nil {
		txt := fmt.Sprintf("Failed to close existing device connectsions for %s:\n%s", device.DESDevSerial, err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(txt)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"device": &device})
}

func HandleDESDeviceClientRefresh(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleCheckDESDeviceClient( )\n")

	/* CHECK USER PERMISSION */
	if !pkg.UserRole_Admin(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).
		SendString(pkg.ERR_AUTH_SUPER + ": Connect devices")
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := Device{}
	if err = ValidatePostRequestBody_Device(c, &device); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	} //pkg.Json("HandleCheckDESDeviceClient(): -> c.BodyParser(&device) -> device", device)

	/* GET / VALIDATE DESRegistration */
	ser := device.DESDevSerial
	if err = device.GetDeviceDESRegistration(ser); err != nil {
		txt := fmt.Sprintf("DES Registration for %s was not found.\n%s\nDB ERROR", ser, err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(txt)
	} // pkg.Json("HandleCheckDESDeviceClient(): -> device.GetDeviceDESRegistration -> device", device)

	d := DevicesMapRead(device.DESDevSerial)

	/* CLOSE ANY EXISTING CONNECTIONS AND RECONNECT THE DES DEVICE CLIENTS */
	if err = d.DeviceClient_RefreshConnections(); err != nil {
		txt := fmt.Sprintf("Connections for %s could not be refreshed; ERROR:\n%s\n", ser, err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString(txt)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"device": &d})
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

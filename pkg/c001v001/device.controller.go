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
	regs, err := GetDeviceList()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("GetDesDevList(...) -> query failed:\n%s\n", err),
			"data":    fiber.Map{"regs": regs},
		})
	} // pkg.Json("GetDeviceList(): DESRegistrations", regs)

	var wg sync.WaitGroup
	wg.Add(len(regs)) // fmt.Printf("\nWait Group: %d\n", len(regs))

	devices := []Device{}
	for _, reg := range regs {

		// pkg.Json("HandleGetDeviceList( ) -> reg", reg)
		go func(r pkg.DESRegistration, wg *sync.WaitGroup) {

			defer wg.Done()

			device := Device{
				DESRegistration: r,
				Job: Job{DESRegistration: r}, 
			}
			
			// pkg.TraceFunc("Call -> device.GetDeviceStatus( )")
			device.GetDeviceStatus()
			devices = append(devices, device)
		}(reg, &wg)
	}
	wg.Wait() // pkg.Json("HandleGetDeviceList( ) -> []Device{}:\n", devices)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"devices": devices},
	})
}

/*
USED WHEN DATACAN ADMIN WEB CLIENTS REGISTER NEW C001V001 DEVICES ON THIS DES
PERFORMS DES DEVICE REGISTRATION
PERFORMS CLASS/VERSION SPECIFIC REGISTRATION ACTIONS
*/
func (dev *Device) HandleRegisterDevice(c *fiber.Ctx) (err error) {

	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to register devices",
		})
	}

	reg := pkg.DESRegistration{}
	if err = c.BodyParser(&reg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	if errors := pkg.ValidateStruct(reg); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail",
			"errors": errors,
		})
	}

	/*
		CREATE A DEVICE RECORD IN THE DES DB FOR THIS DEVICE
		 - Creates a new DESevice in the DES database
		 - Gets the C001V001Device's DeviceID from the DES Database
	*/
	reg.DESDevRegTime = time.Now().UTC().UnixMilli()
	reg.DESDevRegAddr = c.IP()
	if device_res := pkg.DES.DB.Create(&reg.DESDev); device_res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": device_res.Error.Error(),
		})
	}

	/*
		CREATE THE DEFAULT JOB FOR THIS DEVICE
	*/
	job := Job{
		DESRegistration: pkg.DESRegistration{
			DESDev: reg.DESDev,
			DESJob: pkg.DESJob{
				DESJobRegTime:   reg.DESDevRegTime,
				DESJobRegAddr:   reg.DESDevRegAddr,
				DESJobRegUserID: reg.DESDevRegUserID,
				DESJobRegApp:    reg.DESDevRegApp,

				DESJobName:  fmt.Sprintf("%s_0000000000000", reg.DESDevSerial),
				DESJobStart: 0,
				DESJobEnd:   0,
				DESJobLng:   reg.DESJobLng,
				DESJobLat:   reg.DESJobLat,
				DESJobDevID: reg.DESDevID,
			},
		},
		Admins:  []Admin{(&Job{}).RegisterJob_Default_JobAdmin()},
		Headers: []Header{(&Job{}).RegisterJob_Default_JobHeader()},
		Configs: []Config{(&Job{}).RegisterJob_Default_JobConfig()},
		Events:  []Event{(&Job{}).RegisterJob_Default_JobEvent()},
	}
	if err = job.RegisterJob(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}

	reg = pkg.DESRegistration{
		DESDev: reg.DESDev,
		DESJob: job.DESJob,
	}

	device := Device{
		DESRegistration: reg,
		Job:             Job{DESRegistration: reg},
		DESMQTTClient:   pkg.DESMQTTClient{},
	}
	if err = device.MQTTDeviceClient_Connect(); err != nil {
		return pkg.TraceErr(err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &reg},
		"message": "C001V001 Device Registered.",
	})
}

/*
USED WHEN DEVICE OPERATOR WEB CLIENTS WANT TO START A NEW JOB ON THIS DEVICE
SEND AN MQTT JOB ADMIN, HEADER, CONFIG, & EVENT TO THE DEVICE
UPON MQTT MESSAGE AT '.../CMD/EVENT, DEVICE CLIENT PERFORMS

	DES JOB REGISTRATION
	CLASS/VERSION SPECIFIC JOB START ACTIONS
*/
func (device *Device) HandleStartJob(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleStartJob( )\n")

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to start a job",
		})
	}
	/* PARSE AND VALIDATE REQUEST DATA */
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	if errors := pkg.ValidateStruct(device); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail",
			"errors": errors,
		})
	}
	// pkg.Json("(dev *Device) HandleStartJob(): -> c.BodyParser(&device) -> dev", device)

	/*
		START NEW JOB
			MAKE ADM, HDR, CFG, EVT ( START JOB )
			ENSURE ADM, HDR, CFG, & EVT HAVE THE SAME TIME STAMP / SIGNATURE
	*/
	time := time.Now().UTC().UnixMilli()

	device.DESRegistration.DESJobRegTime = time
	device.Job.DESRegistration = device.DESRegistration
	// device.DESMQTTClient = pkg.MQTTDevClients[device.DESDevSerial]
	// device.DESMQTTClient = d.DESMQTTClient

	device.ADM.AdmID = 0
	device.ADM.AdmTime = time
	device.ADM.AdmAddr = c.IP()
	device.ADM.AdmUserID = device.DESJobRegUserID
	device.ADM.AdmDefHost = pkg.MQTT_HOST
	device.ADM.AdmDefPort = pkg.MQTT_PORT
	device.ADM.AdmOpHost = pkg.MQTT_HOST
	device.ADM.AdmOpPort = pkg.MQTT_PORT
	device.ADM.AdmSerial = device.Job.DESDevSerial
	// pkg.Json("(device *Device) HandleStartJob(): -> device.ADM", device.ADM)

	device.HDR.HdrID = 0
	device.HDR.HdrTime = time
	device.HDR.HdrAddr = c.IP()
	device.HDR.HdrUserID = device.DESJobRegUserID
	device.HDR.HdrJobStart = time // This is displays the time/date of the request while pending
	device.HDR.HdrJobEnd = -1     // This means there is a pending request for the device to start a new job
	device.HDR.HdrJobName = fmt.Sprintf("%s_0000000000000", device.DESDevSerial)
	device.HDR.HdrGeoLng = -180
	device.HDR.HdrGeoLat = 90
	// pkg.Json("(device *Device) HandleStartJob(): -> device.HDR", device.HDR)

	device.CFG.CfgID = 0
	device.CFG.CfgTime = time
	device.CFG.CfgAddr = c.IP()
	device.CFG.CfgUserID = device.DESJobRegUserID
	// pkg.Json("(device *Device) HandleStartJob(): -> device.CFG", device.CFG)

	device.SMP.SmpID = 0
	device.SMP.SmpTime = time
	device.SMP.SmpJobName = device.HDR.HdrJobName

	device.EVT = Event{
		EvtTime:   time,
		EvtAddr:   c.IP(),
		EvtUserID: device.DESJobRegUserID,
		EvtApp:    device.DESJobRegApp,
		EvtTitle:  "Job Start Request",
		EvtMsg:    "Job start sequence initiated.",
		EvtCode:   2,
	}
	// pkg.Json("(device *Device) HandleStartJob(): -> device.EVT", device.EVT)
	// pkg.Json("(device *Device) HandleStartJob(): -> device", device)

	// /* LOG TO JOB_0: ADM, HDR, CFG, EVT */
	zero := device.ZeroJob()
	zero.Write(&device.ADM)
	zero.Write(&device.HDR)
	zero.Write(&device.CFG)
	zero.Write(&device.EVT)
	// zero.Write(&device.SMP)
	// // fmt.Printf("\nHandleStartJob( ) -> DB Write to %s complete.\n", zero.DESJobName)
	// // pkg.Json("(device *Device) HandleStartJob(): -> device", device)

	d := Devices[device.DESDevSerial]
	device.DESMQTTClient = d.DESMQTTClient
	// fmt.Printf("\nHandleStartJob( ) -> Check %s MQTT device: %v\n", device.DESDevSerial, device.MQTTClientID)
	Devices[device.DESDevSerial] = *device

	// /* MQTT PUB CMD: ADM, HDR, CFG, EVT */
	fmt.Printf("\nHandleStartJob( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDAdmin(device.ADM)
	device.MQTTPublication_DeviceClient_CMDHeader(device.HDR)
	device.MQTTPublication_DeviceClient_CMDConfig(device.CFG)
	device.MQTTPublication_DeviceClient_CMDEvent(device.EVT)

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

	DES JOB REGISTRATION ( UPDATE JOB 0 START DATE )
	CLASS/VERSION SPECIFIC JOB END ACTIONS
*/
func (device *Device) HandleEndJob(c *fiber.Ctx) (err error) {
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
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	if errors := pkg.ValidateStruct(device); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail",
			"errors": errors,
		})
	}
	// pkg.Json("(dev *Device) HandleEndJob(): -> c.BodyParser(&device) -> dev", device)

	time := time.Now().UTC().UnixMilli()

	device.EVT = Event{
		EvtTime:   time,
		EvtAddr:   c.IP(),
		EvtUserID: device.DESJobRegUserID,
		EvtApp:    device.DESJobRegApp,
		EvtTitle:  "Job End Request",
		EvtMsg:    "Job end sequence initiated.",
		EvtCode:   1,
	}

	/* LOG TO JOB_0: EVT */
	zero := device.ZeroJob()
	zero.Write(&device.EVT)
	device.EVT.EvtID = 0
	// fmt.Printf("\nHandleEndJob( ) -> DB Write to %s complete.\n", zero.DESJobName)

	d := Devices[device.DESDevSerial]
	if d.DESMQTTClient.Client == nil  { 
		d.MQTTDeviceClient_Connect()
	 }
	// device.DESMQTTClient = d.DESMQTTClient
	// device.Job = d.Job
	// pkg.Json("(device *Device) HandleEndJob(): -> Devices[device.DESDevSerial]", d)

	d.EVT = device.EVT
	d.Job.Write(&d.EVT)
	d.EVT.EvtID = 0
	// fmt.Printf("\nHandleEndJob( ) -> DB Write to %s complete.\n", device.Job.DESJobName)
	Devices[device.DESDevSerial] = d

	/* MQTT PUB CMD: EVT */
	fmt.Printf("\nHandleEndJob( ) -> Publishing to %s with MQTT device client: %s\n\n", d.DESDevSerial, d.MQTTClientID)
	d.MQTTPublication_DeviceClient_CMDEvent(d.EVT)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 Job End Reqest sent to device.",
	})
}

/*
USED TO ALTER THE ADMIN SETTINGS FOR A GIVEN DEVICE 
BOTH DURING A JOB OR WHEN SENT TO JOB 0, TO ALTER THE DEVICE DEFAULTS
*/
func (device *Device) HandleSetAdmin(c *fiber.Ctx) (err error) { 
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
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	if errors := pkg.ValidateStruct(device); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail",
			"errors": errors,
		})
	}
	pkg.Json("(devive *Device) HandleSetAdmin(): -> c.BodyParser(&device) -> device.ADM", device.ADM)

	device.ADM.AdmID = 0
	device.ADM.AdmTime = time.Now().UTC().UnixMilli()	
	device.ADM.AdmAddr = c.IP()
	
	/* LOG TO JOB_0: ADM */
	zero := device.ZeroJob()
	zero.Write(&device.ADM)
	device.ADM.AdmID = 0
	// fmt.Printf("\nHandleSetAdmin( ) -> DB Write to %s complete.\n", zero.DESJobName)

	d := Devices[device.DESDevSerial]
	if d.DESMQTTClient.Client == nil  { 
		d.MQTTDeviceClient_Connect()
	}
	d.ADM = device.ADM
	Devices[device.DESDevSerial] = d
	
	/* MQTT PUB CMD: ADM */
	fmt.Printf("\nHandleSetAdmin( ) -> Publishing to %s with MQTT device client: %s\n\n", d.DESDevSerial, d.MQTTClientID)
	d.MQTTPublication_DeviceClient_CMDAdmin(d.ADM)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 SET ADMIN Reqest sent to device.",
	})
}

/*
USED TO ALTER THE HEADER SETTINGS FOR A GIVEN DEVICE 
BOTH DURING A JOB OR WHEN SENT TO JOB 0, TO ALTER THE DEVICE DEFAULTS
*/
func (device *Device) HandleSetHeader(c *fiber.Ctx) (err error) { 
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
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	if errors := pkg.ValidateStruct(device); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail",
			"errors": errors,
		})
	}
	pkg.Json("(devive *Device) HandleSetHeader(): -> c.BodyParser(&device) -> device.HDR", device.HDR)

	device.HDR.HdrID = 0
	device.HDR.HdrTime = time.Now().UTC().UnixMilli()	
	device.HDR.HdrAddr = c.IP()
	
	/* LOG TO JOB_0: HDR */
	zero := device.ZeroJob()
	zero.Write(&device.HDR)
	device.HDR.HdrID = 0
	// fmt.Printf("\nHandleSetHeader( ) -> DB Write to %s complete.\n", zero.DESJobName)

	d := Devices[device.DESDevSerial]
	if d.DESMQTTClient.Client == nil  { 
		d.MQTTDeviceClient_Connect()
	}
	d.HDR = device.HDR
	Devices[device.DESDevSerial] = d
	
	/* MQTT PUB CMD: HDR */
	fmt.Printf("\nHandleSetHeader( ) -> Publishing to %s with MQTT device client: %s\n\n", d.DESDevSerial, d.MQTTClientID)
	d.MQTTPublication_DeviceClient_CMDHeader(d.HDR)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 SET HEADER Reqest sent to device.",
	})
}

/*
USED TO ALTER THE CONFIG SETTINGS FOR A GIVEN DEVICE 
BOTH DURING A JOB OR WHEN SENT TO JOB 0, TO ALTER THE DEVICE DEFAULTS
*/
func (device *Device) HandleSetConfig(c *fiber.Ctx) (err error) {
	fmt.Printf("\nHandleSetConfig( )\n")
	
	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to alter job configuration data.",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	if errors := pkg.ValidateStruct(device); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail",
			"errors": errors,
		})
	}
	pkg.Json("(devive *Device) HandleSetConfig(): -> c.BodyParser(&device) -> device.CFG", device.CFG)

	device.CFG.CfgID = 0
	device.CFG.CfgTime = time.Now().UTC().UnixMilli()	
	device.CFG.CfgAddr = c.IP()
	
	/* LOG TO JOB_0: CFG */
	zero := device.ZeroJob()
	zero.Write(&device.CFG)
	device.CFG.CfgID = 0
	// fmt.Printf("\nHandleSetConfig( ) -> DB Write to %s complete.\n", zero.DESJobName)

	d := Devices[device.DESDevSerial]
	if d.DESMQTTClient.Client == nil  { 
		d.MQTTDeviceClient_Connect()
	}
	d.CFG = device.CFG
	Devices[device.DESDevSerial] = d
	
	/* MQTT PUB CMD: CFG */
	fmt.Printf("\nHandleSetConfig( ) -> Publishing to %s with MQTT device client: %s\n\n", d.DESDevSerial, d.MQTTClientID)
	d.MQTTPublication_DeviceClient_CMDConfig(d.CFG)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 SET CONFIG Reqest sent to device.",
	})
}

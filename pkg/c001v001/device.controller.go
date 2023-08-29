package c001v001

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/leehayford/des/pkg"
)

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

		go func(r pkg.DESRegistration, wg *sync.WaitGroup) {

			defer wg.Done()
			job := Job{DESRegistration: r}
			job.GetJobData(-1) // pkg.Json("HandleGetDeviceList(): job", job)

			device := Device{Job: job, DESRegistration: r}
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
		return pkg.Trace(err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &reg},
		"message": "C001V001 Device Registered.",
	})
}

/*
USED WHEN DEVICE OPERATOR WEB CLIENTS WANT TO START A NEW JOB ON THIS DEVICE
SEND AN MQTT JOB HEADER TO THE DEVICE
UPON RESPONSE AT '.../CMD/HEADER
PERFORMS DES JOB REGISTRATION
PERFORMS CLASS/VERSION SPECIFIC JOB REGISTRATION ACTIONS
*/
func (device *Device) HandleStartNewJob(c *fiber.Ctx) (err error) {

	/* CHECK USER PERMISSION */
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to register devices",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	req := Job{}
	if err = c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	if errors := pkg.ValidateStruct(req); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail",
			"errors": errors,
		})
	}
	// pkg.Json("(dev *Device) HandleStartNewJob(): -> c.BodyParser(&req) -> req", req)

	device.DESRegistration = GetDeviceZeroJob(req.DESDevSerial)
	device.Job.DESRegistration = device.DESRegistration
	device.Job.DESJobLng = 0
	device.Job.DESJobLat = 0
	// pkg.Json("(dev *Device) HandleStartNewJob(): -> GetDeviceZeroJob( ) -> device.", device)

	/* CHECK DEVICE LAST KNOWN DEVICE STATE */
	job_0_evt := (&device.Job).GetLastEvent()
	// pkg.Json("(dev *Device) HandleStartNewJob(): -> zero_job.GetLastEvent() -> job_0_evt", job_0_evt)
	if job_0_evt.EvtCode > 1 {
		/*
			END CURRENT JOB (JOB_X)
				MAKE EVENT(END JOB)
				MQTT SEND EVENT
				LOG EVENT IN JOB_0
				LOG EVENT IN JOB_X
		*/
	}

	/*
		START NEW JOB
			MAKE ADM, HDR, CFG, EVT ( START JOB )
			ENSURE ADM, HDR, CFG, & EVT HAVE THE SAME TIME STAMP / SIGNATURE
	*/
	time := time.Now().UTC().UnixMilli()

	evt := Event{
		EvtTime:   time,
		EvtAddr:   c.IP(),
		EvtUserID: device.Job.DESJobRegUserID,
		EvtApp:    device.Job.DESJobRegApp,
		EvtTitle:  "Start Job Request",
		EvtMsg:    "Job start sequence initiated.",
		EvtCode:   1,
	}
	// pkg.Json("(dev *Device) HandleStartNewJob(): -> evt", evt)

	adm := req.Admins[0]
	adm.AdmTime = time
	adm.AdmAddr = c.IP()
	adm.AdmUserID = device.Job.DESJobRegUserID
	adm.AdmDefHost = pkg.MQTT_BROKER
	adm.AdmDefPort = pkg.MQTT_PORT
	adm.AdmOpHost = pkg.MQTT_BROKER
	adm.AdmOpPort = pkg.MQTT_PORT
	adm.AdmSerial = device.Job.DESDevSerial
	// pkg.Json("(dev *Device) HandleStartNewJob(): -> adm", adm)

	hdr := req.Headers[0]
	hdr.HdrTime = time
	hdr.HdrAddr = c.IP()
	hdr.HdrUserID = device.Job.DESJobRegUserID
	hdr.HdrJobStart = time // This is displays the time/date of the request while pending
	hdr.HdrJobEnd = -1 // This means there is a pending request for the device to start a new job
	hdr.HdrJobName = fmt.Sprintf("%s_0000000000000", device.Job.DESDevSerial)
	hdr.HdrJobEnd = 0
	hdr.HdrGeoLng = 0
	hdr.HdrGeoLat = 0
	// pkg.Json("(dev *Device) HandleStartNewJob(): -> hdr", hdr)

	cfg := req.Configs[0]
	cfg.CfgTime = time
	cfg.CfgAddr = c.IP()
	cfg.CfgUserID = device.Job.DESJobRegUserID
	// pkg.Json("(dev *Device) HandleStartNewJob(): -> cfg", cfg)

	/* LOG TO JOB_0: ADM, HDR, CFG, EVT */
	device.Job.Write(&adm)
	device.Job.Write(&hdr)
	device.Job.Write(&cfg)
	device.Job.Write(&evt)

	/* MQTT PUB CMD: ADM, HDR, CFG, EVT */
	device.DESMQTTClient = pkg.MQTTDevClients[device.DESDevSerial]
	fmt.Printf("\nPublishing to %s with MQTT client: %s\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDAdmin(adm)
	device.MQTTPublication_DeviceClient_CMDHeader(hdr)
	device.MQTTPublication_DeviceClient_CMDConfig(cfg)
	device.MQTTPublication_DeviceClient_CMDEvent(evt)

	return
}

func GetDeviceZeroJob(serial string) (reg pkg.DESRegistration) {

	zero_job := fmt.Sprintf("%s_0000000000000", serial)

	qry := pkg.DES.DB.
		Table("des_devs AS d").
		Select("d.*, j.*").
		Joins("JOIN des_jobs AS j ON d.des_dev_id = j.des_job_dev_id").
		Where("d.des_dev_serial = ? AND j.des_job_name LIKE ?", serial, zero_job).
		Order("d.des_dev_id DESC")

	res := qry.Scan(&reg)
	if res.Error != nil {
		pkg.Trace(res.Error)
	} // pkg.Json("GetDeviceZeroJob( ): ", reg)

	return
}

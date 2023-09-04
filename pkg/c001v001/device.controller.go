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

		pkg.Json("HandleGetDeviceList( ) -> reg", reg)
		go func(r pkg.DESRegistration, wg *sync.WaitGroup) {

			defer wg.Done()
			job := Job{DESRegistration: r}
			// job.GetJobData(-1) // pkg.Json("HandleGetDeviceList(): job", job)

			device := Device{Job: job, DESRegistration: r}
			// device := Device{ DESRegistration: r }
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
SEND AN MQTT JOB ADMIN, HEADER, CONFIG, & EVENT TO THE DEVICE
UPON MQTT MESSAGE AT '.../CMD/EVENT, DEVICE CLIENT PERFORMS 
	DES JOB REGISTRATION
	CLASS/VERSION SPECIFIC JOB START ACTIONS
*/
func (device *Device) HandleStartJob(c *fiber.Ctx) (err error) {
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
	device.DESMQTTClient = pkg.MQTTDevClients[device.DESDevSerial]

	device.ADM.AdmID = 0
	device.ADM.AdmTime = time
	device.ADM.AdmAddr = c.IP()
	device.ADM.AdmUserID = device.DESJobRegUserID
	device.ADM.AdmDefHost = pkg.MQTT_BROKER
	device.ADM.AdmDefPort = pkg.MQTT_PORT
	device.ADM.AdmOpHost = pkg.MQTT_BROKER
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
		EvtTitle:  "Start Job Request",
		EvtMsg:    "Job start sequence initiated.",
		EvtCode:   2,
	}
	// pkg.Json("(device *Device) HandleStartJob(): -> device.EVT", device.EVT)

	// pkg.Json("(device *Device) HandleStartJob(): -> device", device)

	/* LOG TO JOB_0: ADM, HDR, CFG, EVT */
	zero := device.GetZeroJob()
	zero.Write(&device.ADM)
	zero.Write(&device.HDR)
	zero.Write(&device.CFG)
	zero.Write(&device.EVT)
	zero.Write(&device.SMP)
	fmt.Printf("\nHandleStartJob( ) -> DB Write to %s complete.\n", zero.DESJobName)
	// pkg.Json("(device *Device) HandleStartJob(): -> device", device)

	// /* MQTT PUB CMD: ADM, HDR, CFG, EVT */
	fmt.Printf("\nHandleStartJob( ) -> Publishing to %s with MQTT device client: %s\n\n", device.DESDevSerial, device.MQTTClientID)
	device.MQTTPublication_DeviceClient_CMDAdmin(device.ADM)
	device.MQTTPublication_DeviceClient_CMDHeader(device.HDR)
	device.MQTTPublication_DeviceClient_CMDConfig(device.CFG)
	device.MQTTPublication_DeviceClient_CMDEvent(device.EVT)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
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
	fmt.Printf("\nHandleEndtJob( )\n")
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
	evt := Event{
		EvtTime:   time,
		EvtAddr:   c.IP(),
		EvtUserID: device.DESJobRegUserID,
		EvtApp:    device.DESJobRegApp,
		EvtTitle:  "Job End Request",
		EvtMsg:    "Job end sequence initiated.",
		EvtCode:   1,
	}

	d := Devices[device.DESDevSerial]
	// pkg.Json("(device *Device) HandleEndJob(): -> Devices[device.DESDevSerial]", d)

	/* LOG TO JOB_0: EVT */
	zero := d.GetZeroJob()
	zero.Write(&evt)
	fmt.Printf("\nHandleEndJob( ) -> DB Write to %s complete.\n", zero.DESJobName)

	evt.EvtID = 0
	d.EVT = evt
	d.Job.Write(&d.EVT)
	fmt.Printf("\nHandleEndJob( ) -> DB Write to %s complete.\n", d.DESJobName)
	
	/* MQTT PUB CMD: EVT */
	fmt.Printf("\nHandleEndJob( ) -> Publishing to %s with MQTT device client: %s\n\n", d.DESDevSerial, d.MQTTClientID)
	d.MQTTPublication_DeviceClient_CMDEvent(evt)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":    fiber.Map{"device": &device},
		"message": "C001V001 Job End Reqest sent to device.",
	})
}

func (device *Device) GetZeroJob() (zero Job) {
	// fmt.Printf("\n(device) GetZeroJob() for: %s\n", device.DESDevSerial)
	qry := pkg.DES.DB.
		Table("des_devs AS d").
		Select("d.*, j.*").
		Joins("JOIN des_jobs AS j ON d.des_dev_id = j.des_job_dev_id").
		Where("d.des_dev_serial = ? AND j.des_job_name LIKE ?",
			device.DESDevSerial, fmt.Sprintf("%s_0000000000000", device.DESDevSerial))

	res := qry.Scan(&zero.DESRegistration)
	if res.Error != nil {
		pkg.Trace(res.Error)
	}
	// pkg.Json("(device *Device) GetZeroJob( )", zero)
	return
}

func (device *Device) GetCurrentJob() {
	// fmt.Printf("\n(device) GetCurrentJob() for: %s\n", device.DESDevSerial)
	
	subQryLatestJob := pkg.DES.DB.
	Table("des_jobs").
	Select("des_job_dev_id, MAX(des_job_reg_time) AS max_time").
	Where("des_job_end = 0").
	Group("des_job_dev_id")

	qry := pkg.DES.DB.
	Table("des_jobs").
	Select("des_devs.*, des_jobs.*").
	Joins(`JOIN ( ? ) j ON des_jobs.des_job_dev_id = j.des_job_dev_id AND des_job_reg_time = j.max_time`, subQryLatestJob).
	Joins("JOIN des_devs ON des_devs.des_dev_id = j.des_job_dev_id").
	Where("des_devs.des_dev_serial = ? ", device.DESDevSerial)

	res := qry.Scan(&device.Job.DESRegistration)
	if res.Error != nil {
		pkg.Trace(res.Error)
	}
	// pkg.Json("(device *Device) GetCurrentJob( )", device.Job)
	return
}

func (device *Device) GetDeviceStatus() (err error) {
	device.GetCurrentJob()
	db := device.Job.JDB()
	db.Connect()
	defer db.Close()

	db.Last(&device.ADM)
	db.Last(&device.HDR)
	db.Last(&device.CFG)
	db.Last(&device.EVT)
	db.Last(&device.SMP)

	db.Close()
	return
}

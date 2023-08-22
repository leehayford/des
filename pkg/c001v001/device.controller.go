package c001v001

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/leehayford/des/pkg"
)

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
				DESJobStart: reg.DESDevRegTime,
				DESJobEnd:   0,
				DESJobLng: reg.DESJobLng,
				DESJobLat: reg.DESJobLat,
				DESJobDevID: reg.DESDevID,
			},
		},
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

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"data":    fiber.Map{"device": &reg},
		"message": "C001V001 Device Registered.",
	})
}


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
			job.GetJobData(50) // pkg.Json("HandleGetDeviceList(): job", job)
	
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

func HandleGetDeviceBySerial( c *fiber.Ctx) (err error) {
	return
}
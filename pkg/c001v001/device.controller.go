
package c001v001

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/leehayford/des/pkg"
)

/*
USED WHEN DATACAN ADMIN WEB CLIENTS REGISTER NEW C001V001 DEVICES ON THIS DES
PERFORMS DES DEVICE REGISTRATION
PERFORMS CLASS/VERSION SPECIFIC REGISTRATION ACTIONS
*/
func (dev *Device) RegisterDevice(c *fiber.Ctx) (err error) {
	
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status": "fail", 
			"message": "You must be an administrator to register devices",
		})
	}

	device := pkg.DESDev{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail", 
			"message": err.Error(),
		})
	}
	if errors := pkg.ValidateStruct(device); errors != nil {
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
	device.DESDevRegTime = time.Now().UTC().UnixMicro()
	device.DESDevRegAddr = c.IP()
	if device_res := pkg.DES.DB.Create(&device); device_res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message": device_res.Error.Error(),
		})
	}

	/*
		CREATE THE DEFAULT JOB FOR THIS DEVICE
	*/
	job := Job{
		DESRegistration: pkg.DESRegistration{
			DESDev: device,
			DESJob: pkg.DESJob{
				DESJobRegTime: device.DESDevRegTime,
				DESJobRegAddr: device.DESDevRegAddr,
				DESJobRegUserID: device.DESDevRegUserID,
				DESJobRegApp: device.DESDevRegApp,
		
				DESJobName: fmt.Sprintf("%s_0000000000000000", device.DESDevSerial),
				DESJobStart: device.DESDevRegTime,
				DESJobEnd: 0,
		
				DESJobDevID: device.DESDevID,
			},
		},
	} 
	if err = job.RegisterJob(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "fail", 
			"message": err.Error(),
		})
	}


	reg := pkg.DESRegistration{
		DESDev: device,
		DESJob: job.DESJob,
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success", 
		"data": fiber.Map{"device": &reg},
		"message": "C001V001 Device Registered.",
	})

}
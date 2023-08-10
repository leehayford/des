package c001v001

import (
	"github.com/leehayford/des/pkg"
)

const DEVICE_CLASS = "001"
const DEVICE_VERSION = "001"

type Device struct {
	Job `json:"job"` // The active job for this device ( last job if it has ended )
	pkg.DESDev
	pkg.DESMQTTClient `json:"-"`
}


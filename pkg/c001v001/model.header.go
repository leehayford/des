package c001v001

import (
	"fmt"
	"sync"
	"github.com/leehayford/des/pkg"
)

/*
HEADER AS WRITTEN TO JOB DATABASE
*/
type Header struct {
	// HdrID   int64  `gorm:"unique; primaryKey" json:"-"` // POSTGRES
	HdrID int64 `gorm:"autoIncrement" json:"-"`

	HdrTime   int64  `gorm:"not null" json:"hdr_time"`
	HdrAddr   string `gorm:"varchar(36)" json:"hdr_addr"`
	HdrUserID string `gorm:"not null; varchar(36)"  json:"hdr_user_id"`
	HdrApp    string `gorm:"varchar(36)" json:"hdr_app"`

	HdrJobStart int64 `json:"hdr_job_start"`
	HdrJobEnd   int64 `json:"hdr_job_end"`

	/*WELL INFORMATION*/
	HdrWellCo    string `gorm:"varchar(32)" json:"hdr_well_co"`
	HdrWellName  string `gorm:"varchar(32)" json:"hdr_well_name"`
	HdrWellSFLoc string `gorm:"varchar(32)" json:"hdr_well_sf_loc"`
	HdrWellBHLoc string `gorm:"varchar(32)" json:"hdr_well_bh_loc"`
	HdrWellLic   string `gorm:"varchar(32)" json:"hdr_well_lic"`

	/* TODO: CHANGE HDR LNG / LAT TO FLOAT32*/
	/*GEO LOCATION - USED TO POPULATE A GeoJSON OBJECT */
	HdrGeoLng float64 `json:"hdr_geo_lng"`
	HdrGeoLat float64 `json:"hdr_geo_lat"`
	// HdrGeoLng float32 `json:"hdr_geo_lng"`
	// HdrGeoLat float32 `json:"hdr_geo_lat"`
}

func WriteHDR(hdr Header, jdbc *pkg.JobDBClient) (err error) {

	/* WHEN Write IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING
	WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	if jdbc.RWM == nil {
		jdbc.RWM = &sync.RWMutex{}
	}
	jdbc.RWM.Lock()
	res := jdbc.Create(&hdr)
	jdbc.RWM.Unlock()

	return res.Error
}
func ReadLastHDR(hdr *Header, jdbc *pkg.JobDBClient) (err error) {
	
	/* WHEN Read IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING
	WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	if jdbc.RWM == nil {
		jdbc.RWM = &sync.RWMutex{}
	}
	jdbc.RWM.Lock()
	res := jdbc.Last(&hdr)
	jdbc.RWM.Unlock()

	return res.Error
}

/*
HEADER - DEFAULT VALUES
*/
func (hdr *Header) DefaultSettings_Header(reg pkg.DESRegistration) {
	hdr.HdrTime = reg.DESJobRegTime
	hdr.HdrAddr = reg.DESJobRegAddr
	hdr.HdrUserID = reg.DESJobRegUserID
	hdr.HdrApp = reg.DESJobRegApp

	hdr.HdrJobStart = 0
	hdr.HdrJobEnd = 0

	hdr.HdrWellCo = ""
	hdr.HdrWellName = ""
	hdr.HdrWellSFLoc = ""
	hdr.HdrWellBHLoc = ""
	hdr.HdrWellLic = ""

	hdr.HdrGeoLng = DEFAULT_GEO_LNG
	hdr.HdrGeoLat = DEFAULT_GEO_LAT
}

/*
HARDWARE IDs - VALIDATE FIELDS
*/
func (hdr *Header) Validate() {
	/* TODO: SET ACCEPTABLE LIMITS FOR THE REST OF THE CONFIG SETTINGS */

	hdr.HdrAddr = pkg.ValidateStringLength(hdr.HdrAddr, 36)
	hdr.HdrUserID = pkg.ValidateStringLength(hdr.HdrUserID, 36)
	hdr.HdrApp = pkg.ValidateStringLength(hdr.HdrApp, 36)

	hdr.HdrWellCo = pkg.ValidateStringLength(hdr.HdrWellCo, 32)
	hdr.HdrWellName = pkg.ValidateStringLength(hdr.HdrWellName, 32)
	hdr.HdrWellSFLoc = pkg.ValidateStringLength(hdr.HdrWellSFLoc, 32)
	hdr.HdrWellBHLoc = pkg.ValidateStringLength(hdr.HdrWellBHLoc, 32)
	hdr.HdrWellLic = pkg.ValidateStringLength(hdr.HdrWellLic, 32)
}

func (hdr *Header) GetMessageSource() (src pkg.DESMessageSource) {
	src.Time = hdr.HdrTime
	src.Addr = hdr.HdrAddr
	src.UserID = hdr.HdrUserID
	src.App = hdr.HdrApp
	return
}

/*
HEADER - VALIDATE CMD REQUEST FROM USER
*/
func (hdr *Header) CMDValidate(device *Device, uid string) (err error) {

	src := hdr.GetMessageSource()
	dev_src := device.ReferenceSRC()
	if err = src.ValidateSRC_CMD(dev_src, uid, hdr); err != nil {
		return
	}

	return
}

/*
HEADER - VALIDATE MQTT SIG FROM DEVICE
*/
func (hdr *Header) SIGValidate(device *Device) (err error) {

	src := hdr.GetMessageSource()
	dev_src := device.ReferenceSRC()
	if err = src.ValidateSRC_SIG(dev_src, hdr); err != nil {
		return
	}

	hdr.Validate()

	return
}

/*
HEADER - CREATE DESJobSearch
*/
func (hdr *Header) SearchToken() (token string) {

	/* TODO: EVALUATE MORE CLEVER TOKENIZATION */
	return fmt.Sprintf("%s %s %s %s %s",
		hdr.HdrWellCo,
		hdr.HdrWellName,
		hdr.HdrWellSFLoc,
		hdr.HdrWellBHLoc,
		hdr.HdrWellLic,
	)
}

/*
HEADER - AS STORED IN DEVICE FLASH
*/
func (hdr Header) HeaderToBytes() (out []byte) {

	out = append(out, pkg.Int64ToBytes(hdr.HdrTime)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrAddr, 36)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrUserID, 36)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrApp, 36)...)

	out = append(out, pkg.Int64ToBytes(hdr.HdrJobStart)...)
	out = append(out, pkg.Int64ToBytes(hdr.HdrJobEnd)...)

	out = append(out, pkg.StringToNBytes(hdr.HdrWellCo, 32)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrWellName, 32)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrWellSFLoc, 32)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrWellBHLoc, 32)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrWellLic, 32)...)

	/* TODO: CHANGE HDR LNG / LAT TO FLOAT32*/
	out = append(out, pkg.Float64ToBytes(hdr.HdrGeoLng)...)
	out = append(out, pkg.Float64ToBytes(hdr.HdrGeoLat)...)
	// out = append(out, pkg.Float32ToBytes(hdr.HdrGeoLng)...)
	// out = append(out, pkg.Float32ToBytes(hdr.HdrGeoLat)...)

	return
}
func (hdr *Header) HeaderFromBytes(b []byte) {

	hdr = &Header{

		HdrTime:   pkg.BytesToInt64_L(b[0:8]),
		HdrAddr:   pkg.StrBytesToString(b[8:44]),
		HdrUserID: pkg.StrBytesToString(b[44:80]),
		HdrApp:    pkg.StrBytesToString(b[80:116]),

		HdrJobStart: pkg.BytesToInt64_L(b[116:124]),
		HdrJobEnd:   pkg.BytesToInt64_L(b[124:132]),

		HdrWellCo:    pkg.StrBytesToString(b[132:164]),
		HdrWellName:  pkg.StrBytesToString(b[164:196]),
		HdrWellSFLoc: pkg.StrBytesToString(b[196:228]),
		HdrWellBHLoc: pkg.StrBytesToString(b[228:260]),
		HdrWellLic:   pkg.StrBytesToString(b[260:292]),

		/* TODO: CHANGE HDR LNG / LAT TO FLOAT32*/
		HdrGeoLng: pkg.BytesToFloat64_L(b[292:300]),
		HdrGeoLat: pkg.BytesToFloat64_L(b[300:308]),
		// HdrGeoLng: pkg.BytesToFloat64_L(b[292:296]),
		// HdrGeoLat: pkg.BytesToFloat64_L(b[296:300]),
	}
	//  pkg.Json("(demo *DemoDeviceClient)HeaderFromBytes() -> hdr", hdr)
	return
}

package c001v001

import (
	"fmt"

	"github.com/leehayford/des/pkg"
)

/*
HEADER AS WRITTEN TO JOB DATABASE
*/
type Header struct {
	HdrID int64 `gorm:"unique; primaryKey" json:"-"`	

	HdrTime   int64  `gorm:"not null" json:"hdr_time"`
	HdrAddr   string `gorm:"varchar(36)" json:"hdr_addr"`
	HdrUserID string `gorm:"not null; varchar(36)" json:"hdr_user_id"`
	HdrApp    string `gorm:"varchar(36)" json:"hdr_app"`

	HdrJobName  string `gorm:"not null; varchar(24)" json:"hdr_job_name"`
	HdrJobStart int64  `json:"hdr_job_start"`
	HdrJobEnd   int64  `json:"hdr_job_end"`

	/*WELL INFORMATION*/
	HdrWellCo    string `gorm:"varchar(32)" json:"hdr_well_co"`
	HdrWellName  string `gorm:"varchar(32)" json:"hdr_well_name"`
	HdrWellSFLoc string `gorm:"varchar(32)" json:"hdr_well_sf_loc"`
	HdrWellBHLoc string `gorm:"varchar(32)" json:"hdr_well_bh_loc"`
	HdrWellLic   string `gorm:"varchar(32)" json:"hdr_well_lic"`

	/*GEO LOCATION - USED TO POPULATE A GeoJSON OBJECT */
	HdrGeoLng float32 `json:"hdr_geo_lng"`
	HdrGeoLat float32 `json:"hdr_geo_lat"`

}
func WriteHDR(hdr Header, dbc *pkg.DBClient) (err error) {

	/* WHEN Write IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING 
		WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	dbc.WG.Add(1)
	hdr.HdrID = 0
	res := dbc.Create(&hdr) 
	dbc.WG.Done()

	return res.Error
}


/*
HEADER - AS STORED IN DEVICE FLASH
*/
func (hdr Header) HeaderToBytes() (out []byte) {

	out = append(out, pkg.Int64ToBytes(hdr.HdrTime)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrAddr, 36)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrUserID, 36)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrApp, 36)...)

	out = append(out, pkg.StringToNBytes(hdr.HdrJobName, 24)...)
	out = append(out, pkg.Int64ToBytes(hdr.HdrJobStart)...)
	out = append(out, pkg.Int64ToBytes(hdr.HdrJobEnd)...)

	out = append(out, pkg.StringToNBytes(hdr.HdrWellCo, 32)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrWellName, 32)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrWellSFLoc, 32)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrWellBHLoc, 32)...)
	out = append(out, pkg.StringToNBytes(hdr.HdrWellLic, 32)...)

	out = append(out, pkg.Float32ToBytes(hdr.HdrGeoLng)...)
	out = append(out, pkg.Float32ToBytes(hdr.HdrGeoLat)...)

	return
}
func (hdr *Header) HeaderFromBytes(b []byte) {

	hdr = &Header{

		HdrTime:   pkg.BytesToInt64_L(b[0:8]),
		HdrAddr:   pkg.StrBytesToString(b[8:44]),
		HdrUserID: pkg.StrBytesToString(b[44:80]),
		HdrApp:    pkg.StrBytesToString(b[80:116]),

		HdrJobName:  pkg.StrBytesToString(b[116:140]),
		HdrJobStart: pkg.BytesToInt64_L(b[140:148]),
		HdrJobEnd:   pkg.BytesToInt64_L(b[148:156]),

		HdrWellCo:    pkg.StrBytesToString(b[156:188]),
		HdrWellName:  pkg.StrBytesToString(b[188:220]),
		HdrWellSFLoc: pkg.StrBytesToString(b[220:252]),
		HdrWellBHLoc: pkg.StrBytesToString(b[252:284]),
		HdrWellLic:   pkg.StrBytesToString(b[284:316]),

		HdrGeoLng: pkg.BytesToFloat32_L(b[316:320]),
		HdrGeoLat: pkg.BytesToFloat32_L(b[320:324]),
	}
	//  pkg.Json("(demo *DemoDeviceClient)HeaderFromBytes() -> hdr", hdr)
	return
}

/*
HEADER - DEFAULT VALUES
*/
func (hdr *Header) DefaultSettings_Header(reg pkg.DESRegistration) {
	hdr.HdrTime = reg.DESJobRegTime
	hdr.HdrAddr =  reg.DESJobRegAddr
	hdr.HdrUserID = reg.DESJobRegUserID
	hdr.HdrApp = reg.DESJobRegApp

	hdr.HdrJobName = reg.DESJobName
	hdr.HdrJobStart = 0
	hdr.HdrJobEnd = 0

	hdr.HdrWellCo = ""
	hdr.HdrWellName = ""
	hdr.HdrWellSFLoc = ""
	hdr.HdrWellBHLoc = ""
	hdr.HdrWellLic = ""

	hdr.HdrGeoLng = reg.DESJobLng // HdrGeoLng: -114.75 + rand.Float32() * ( -110.15 - 114.75 ),
	hdr.HdrGeoLat = reg.DESJobLat // HdrGeoLat: 51.85 + rand.Float32() * ( 54.35 - 51.85 ),

}

/*
HARDWARE IDs - VALIDATE FIELDS
*/
func (hdr *Header) Validate() {
	/* TODO: SET ACCEPTABLE LIMITS FOR THE REST OF THE CONFIG SETTINGS */

	hdr.HdrAddr = pkg.ValidateStringLength(hdr.HdrAddr, 36)
	hdr.HdrUserID = pkg.ValidateStringLength(hdr.HdrUserID, 36)
	hdr.HdrApp = pkg.ValidateStringLength(hdr.HdrApp, 36)
	
	hdr.HdrJobName = pkg.ValidateStringLength(hdr.HdrJobName, 10)
	hdr.HdrWellCo = pkg.ValidateStringLength(hdr.HdrWellCo, 32)
	hdr.HdrWellName = pkg.ValidateStringLength(hdr.HdrWellName, 32)
	hdr.HdrWellSFLoc = pkg.ValidateStringLength(hdr.HdrWellSFLoc, 32)
	hdr.HdrWellBHLoc = pkg.ValidateStringLength(hdr.HdrWellBHLoc, 32)
	hdr.HdrWellLic = pkg.ValidateStringLength(hdr.HdrWellLic, 32)

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
HEADER - CREATE DESJobSearch
*/
func (hdr *Header) Create_DESJobSearch(reg pkg.DESRegistration) {

	s := pkg.DESJobSearch{
		DESJobToken: hdr.SearchToken(),
		DESJobKey: reg.DESJobID,
	}

	if res := pkg.DES.DB.Create(&s); res.Error != nil {
		pkg.TraceErr(res.Error)
	}
}

/*
HEADER - UPDATE DESJobSearch
*/
func (hdr *Header) Update_DESJobSearch(reg pkg.DESRegistration) {

	s := pkg.DESJobSearch{}
	if res := pkg.DES.DB.Where("des_job_key = ?", reg.DESJobID).First(&s); res.Error != nil {
		pkg.TraceErr(res.Error)
	}
	s.DESJobToken = hdr.SearchToken()

	if res := pkg.DES.DB.Save(&s); res.Error != nil {
		pkg.TraceErr(res.Error)
	}
}

// /*

// Longitude: -115.000000
// Latitude: 55.000000
// -114.75 > LNG < -110.15
// 51.85 > LAT < 54.35

// */
// /* GeoJSON OBJECTS */
// type GeoJSONFeatureCollection struct {
// 	GeoFtColType     string           `json:"type"`
// 	GeoFtColFeatures []GeoJSONFeature `json:"features"`
// }

// type GeoJSONFeature struct {
// 	GeoFtType       string          `json:"type"`
// 	GeoFtGeometry   GeoJSONGeometry `json:"geometry"`
// 	GeoFtProperties []interface{}   `json:"properties"`
// }

// type GeoJSONGeometry struct {
// 	GeomType   string    `json:"type"`
// 	GeomCoords []float32 `json:"coordinates"`
// }

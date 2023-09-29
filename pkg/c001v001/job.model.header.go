package c001v001

import (
	"github.com/leehayford/des/pkg"
)

/*
HEADER AS WRITTEN TO JOB DATABASE
*/
type Header struct {

	HdrTime   int64  `gorm:"not null" json:"hdr_time"`
	HdrAddr   string `json:"hdr_addr"`
	HdrUserID string `gorm:"not null; varchar(36)" json:"hdr_user_id"`
	HdrApp    string `gorm:"not null; varchar(36)" json:"hdr_app"`

	HdrJobName  string `gorm:"not null; varchar(24)" json:"hdr_job_name"`
	HdrJobStart int64  `json:"hdr_job_start"`
	HdrJobEnd   int64  `json:"hdr_job_end"`

	/*WELL INFORMATION*/
	HdrWellCo    string `json:"hdr_well_co"`
	HdrWellName  string `json:"hdr_well_name"`
	HdrWellSFLoc string `json:"hdr_well_sf_loc"`
	HdrWellBHLoc string `json:"hdr_well_bh_loc"`
	HdrWellLic   string `json:"hdr_well_lic"`

	/*GEO LOCATION - USED TO POPULATE A GeoJSON OBJECT */
	HdrGeoLng float32 `json:"hdr_geo_lng"`
	HdrGeoLat float32 `json:"hdr_geo_lat"`
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
	//  pkg.Json("(demo *DemoDeviceClient)MakeHdrFromBytes() -> hdr", hdr)
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

package c001v001

import (
	"github.com/leehayford/des/pkg"
)
/*
HEADER AS WRITTEN TO JOB DATABASE
*/
type Header struct {
	Hdr_ID int64 `gorm:"unique; primaryKey" json:"hdr_id"`

	HdrTime   int64  `gorm:"not null" json:"hdr_time"`
	HdrAddr   string `json:"hdr_addr"`
	HdrUserID string `gorm:"not null; varchar(36)" json:"hdr_user_id"`
	HdrApp    string `gorm:"not null; varchar(36)" json:"hdr_app"`

	HdrJobName  string  `gorm:"not null; varchar(24)" json:"hdr_job_name"`
	HdrJobStart int64   `json:"hdr_job_start"`
	HdrJobEnd   int64   `json:"hdr_job_end"`

	/*WELL INFORMATION*/
	HdrWellCo string `json:"hdr_well_co"`
	HdrWellName string `json:"hdr_well_name"`
	HdrWellSFLoc string `json:"hdr_well_sf_loc"`
	HdrWellBHLoc string `json:"hdr_well_bh_loc"`
	HdrWellLic string `json:"hdr_well_lic"`

	/*GEO LOCATION - USED TO POPULATE A GeoJSON OBJECT */
	HdrGeoLng float32 `json:"hdr_geo_lng"`
	HdrGeoLat float32 `json:"hdr_geo_lat"`

}

/*
HEADER - MQTT MESSAGE STRUCTURE
*/
type MQTT_JobHeader struct {
	HdrTime   int64  `json:"hdr_time"`
	HdrAddr   string `json:"hdr_addr"`
	HdrUserID string `json:"hdr_user_id"`
	HdrApp    string `json:"hdr_app"`

	HdrJobName  string  `json:"hdr_job_name"`
	HdrJobStart int64   `json:"hdr_job_start"`
	HdrJobEnd   int64   `json:"hdr_job_end"`

	/*WELL INFORMATION*/
	HdrWellCo string `json:"hdr_well_co"`
	HdrWellName string `json:"hdr_well_name"`
	HdrWellSFLoc string `json:"hdr_well_sf_loc"`
	HdrWellBHLoc string `json:"hdr_well_bh_loc"`
	HdrWellLic string `json:"hdr_well_lic"`

	/*GEO LOCATION - USED TO POPULATE A GeoJSON OBJECT */
	HdrGeoLng float32 `json:"hdr_geo_lng"`
	HdrGeoLat float32 `json:"hdr_geo_lat"`
}

func (hdr *Header) FilterHdrRecord() MQTT_JobHeader {
	return MQTT_JobHeader {
		HdrTime: hdr.HdrTime,
		HdrAddr: hdr.HdrAddr,
		HdrUserID: hdr.HdrUserID,
		HdrApp: hdr.HdrApp,

		HdrJobName: hdr.HdrJobName,
		HdrJobStart: hdr.HdrJobStart,
		HdrJobEnd:  hdr.HdrJobEnd,

		HdrWellCo: hdr.HdrWellCo,
		HdrWellName: hdr.HdrWellName,
		HdrWellSFLoc: hdr.HdrWellSFLoc,
		HdrWellBHLoc: hdr.HdrWellBHLoc,
		HdrWellLic: hdr.HdrWellLic,

		HdrGeoLng: hdr.HdrGeoLng,
		HdrGeoLat: hdr.HdrGeoLat,
	}
}

/*
HEADER - AS STORED IN DEVICE FLASH
*/
func (hdr *Header) FilterHdrBytes() (out []byte) {

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
func (hdr *Header) MakeHdrFromBytes(b []byte) {

	hdr = &Header{

		HdrTime:   pkg.BytesToInt64_L(b[0:8]),
		HdrAddr:   pkg.FFStrBytesToString(b[8:44]),
		HdrUserID: pkg.FFStrBytesToString(b[44:80]),
		HdrApp:    pkg.FFStrBytesToString(b[80:116]),

		HdrJobName: pkg.FFStrBytesToString(b[116:140]),
		HdrJobStart: pkg.BytesToInt64_L(b[140:148]),
		HdrJobEnd: pkg.BytesToInt64_L(b[148:156]),

		HdrWellCo: pkg.FFStrBytesToString(b[156:188]),
		HdrWellName: pkg.FFStrBytesToString(b[188:220]),
		HdrWellSFLoc: pkg.FFStrBytesToString(b[220:252]),
		HdrWellBHLoc: pkg.FFStrBytesToString(b[252:284]),
		HdrWellLic: pkg.FFStrBytesToString(b[284:316]),

		HdrGeoLng: pkg.BytesToFloat32_L(b[316:320]),
		HdrGeoLat: pkg.BytesToFloat32_L(b[320:324]),
	}
	//  pkg.Json("(demo *DemoDeviceClient)MakeHdrFromBytes() -> hdr", hdr)
	return
}

/*

Longitude: -115.000000
Latitude: 55.000000
-114.75 > LNG < -110.15
51.85 > LAT < 54.35

*/
/* GeoJSON OBJECTS */
type GeoJSONFeatureCollection struct {
	GeoFtColType string `json:"type"`
	GeoFtColFeatures []GeoJSONFeature `json:"features"`
}

type GeoJSONFeature struct {
	GeoFtType string `json:"type"`
	GeoFtGeometry GeoJSONGeometry `json:"geometry"`
	GeoFtProperties []interface{} `json:"properties"`
}

type GeoJSONGeometry struct {
	GeomType string `json:"type"`
	GeomCoords []float32 `json:"coordinates"`
}
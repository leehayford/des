package c001v001

/*
HEADER AS WRITTEN TO JOB DATABASE
*/
type Header struct {
	Hdr_ID int64 `gorm:"unique; primaryKey" json:"hdr_id"`

	HdrTime   int64  `gorm:"not null" json:"hdr_time"`
	HdrAddr   string `json:"hdr_addr"`
	HdrUserID string `gorm:"not null; varchar(36)" json:"hdr_user_id"`
	HdrApp    string `gorm:"not null; varchar(36)" json:"hdr_app"`

	/*WELL INFORMATION*/
	HdrWellCo string `json:"hdr_well_co"`
	HdrWellName string `json:"hdr_well_name"`
	HdrWellSFLoc string `json:"hdr_well_sf_loc"`
	HdrWellBHLoc string `json:"hdr_well_bh_loc"`
	HdrWellLic string `json:"hdr_well_lic"`

	HdrJobName  string  `gorm:"not null; varchar(24)" json:"hdr_job_name"`
	HdrJobStart int64   `json:"hdr_job_start"`
	HdrJobEnd   int64   `json:"hdr_job_end"`

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

	/*WELL INFORMATION*/
	HdrWellCo string `json:"hdr_well_co"`
	HdrWellName string `json:"hdr_well_name"`
	HdrWellSFLoc string `json:"hdr_well_sf_loc"`
	HdrWellBHLoc string `json:"hdr_well_bh_loc"`
	HdrWellLic string `json:"hdr_well_lic"`

	HdrJobName  string  `json:"hdr_job_name"`
	HdrJobStart int64   `json:"hdr_job_start"`
	HdrJobEnd   int64   `json:"hdr_job_end"`

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

		HdrWellCo: hdr.HdrWellCo,
		HdrWellName: hdr.HdrWellName,
		HdrWellSFLoc: hdr.HdrWellSFLoc,
		HdrWellBHLoc: hdr.HdrWellBHLoc,
		HdrWellLic: hdr.HdrWellLic,

		HdrJobName: hdr.HdrJobName,
		HdrJobStart: hdr.HdrJobStart,
		HdrJobEnd:  hdr.HdrJobEnd,

		HdrGeoLng: hdr.HdrGeoLng,
		HdrGeoLat: hdr.HdrGeoLat,
	}
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
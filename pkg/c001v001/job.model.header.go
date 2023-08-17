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

	/*GEO LOCATION - USED TO POPULATE A GeoJSON OBJECT */
	HdrGeoLng float32 `json:"hdr_geo_lng"`
	HdrGeoLat float32 `json:"hdr_geo_lat"`

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
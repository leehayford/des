package c001v001

import (
	"sync"
	"github.com/leehayford/des/pkg"
)

/*
CONFIG - AS WRITTEN TO JOB DATABASE
*/
type Config struct {
	// CfgID int64 `gorm:"unique; primaryKey" json:"-"`	// POSTGRESS
	CfgID int64 `gorm:"autoIncrement" json:"-"` // SQLITE

	CfgTime   int64  `gorm:"not null" json:"cfg_time"`
	CfgAddr   string `gorm:"varchar(36)" json:"cfg_addr"`
	CfgUserID string `gorm:"not null; varchar(36)" json:"cfg_user_id"`
	CfgApp    string `gorm:"varchar(36)" json:"cfg_app"`

	/*JOB*/
	CfgSCVD     float32 `json:"cfg_scvd"`
	CfgSCVDMult float32 `json:"cfg_scvd_mult"`
	CfgSSPRate  float32 `json:"cfg_ssp_rate"`
	CfgSSPDur   int32   `json:"cfg_ssp_dur"`
	CfgHiSCVF   float32 `json:"cfg_hi_scvf"`
	CfgFlowTog  float32 `json:"cfg_flow_tog"`
	CfgSSCVFDur int32   `json:"cfg_sscvf_dur"`

	/*VALVE*/
	CfgVlvTgt int32 `json:"cfg_vlv_tgt"`
	CfgVlvPos int32 `json:"cfg_vlv_pos"`

	/*OP PERIODS*/
	CfgOpSample int32 `json:"cfg_op_sample"`
	CfgOpLog    int32 `json:"cfg_op_log"`
	CfgOpTrans  int32 `json:"cfg_op_trans"`

	/*DIAG PERIODS*/
	CfgDiagSample int32 `json:"cfg_diag_sample"`
	CfgDiagLog    int32 `json:"cfg_diag_log"`
	CfgDiagTrans  int32 `json:"cfg_diag_trans"`
}

func WriteCFG(cfg Config, jdbc *pkg.JobDBClient) (err error) {

	/* WHEN Write IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING
	WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	if jdbc.RWM == nil {
		jdbc.RWM = &sync.RWMutex{}
	}
	jdbc.RWM.Lock()
	res := jdbc.Create(&cfg)
	jdbc.RWM.Unlock()

	return res.Error
}
func ReadLastCFG(cfg *Config, jdbc *pkg.JobDBClient) (err error) {
	
	/* WHEN Read IS CALLED IN A GO ROUTINE, SEVERAL TRANSACTIONS MAY BE PENDING
	WE WANT TO PREVENT DISCONNECTION UNTIL THIS TRANSACTION HAS FINISHED
	*/
	if jdbc.RWM == nil {
		jdbc.RWM = &sync.RWMutex{}
	}
	jdbc.RWM.Lock()
	cfg.CfgID = 0
	res := jdbc.Last(&cfg)
	jdbc.RWM.Unlock()

	return res.Error
}

/*
CONFIG - AS STORED IN DEVICE FLASH
*/
func (cfg Config) ConfigToBytes() (out []byte) {

	out = append(out, pkg.Int64ToBytes(cfg.CfgTime)...)
	out = append(out, pkg.StringToNBytes(cfg.CfgAddr, 36)...)
	out = append(out, pkg.StringToNBytes(cfg.CfgUserID, 36)...)
	out = append(out, pkg.StringToNBytes(cfg.CfgApp, 36)...)

	out = append(out, pkg.Float32ToBytes(cfg.CfgSCVD)...)
	out = append(out, pkg.Float32ToBytes(cfg.CfgSCVDMult)...)
	out = append(out, pkg.Float32ToBytes(cfg.CfgSSPRate)...)
	out = append(out, pkg.Int32ToBytes(cfg.CfgSSPDur)...)
	out = append(out, pkg.Float32ToBytes(cfg.CfgHiSCVF)...)
	out = append(out, pkg.Float32ToBytes(cfg.CfgFlowTog)...)
	out = append(out, pkg.Int32ToBytes(cfg.CfgSSCVFDur)...)

	out = append(out, pkg.Int32ToBytes(cfg.CfgVlvTgt)...)
	out = append(out, pkg.Int32ToBytes(cfg.CfgVlvPos)...)

	out = append(out, pkg.Int32ToBytes(cfg.CfgOpSample)...)
	out = append(out, pkg.Int32ToBytes(cfg.CfgOpLog)...)
	out = append(out, pkg.Int32ToBytes(cfg.CfgOpTrans)...)

	out = append(out, pkg.Int32ToBytes(cfg.CfgDiagSample)...)
	out = append(out, pkg.Int32ToBytes(cfg.CfgDiagLog)...)
	out = append(out, pkg.Int32ToBytes(cfg.CfgDiagTrans)...)

	return
}
func (cfg *Config) ConfigFromBytes(b []byte) {

	cfg = &Config{

		CfgTime:   pkg.BytesToInt64_L(b[0:8]),
		CfgAddr:   pkg.StrBytesToString(b[8:44]),
		CfgUserID: pkg.StrBytesToString(b[44:80]),
		CfgApp:    pkg.StrBytesToString(b[80:116]),

		CfgSCVD:     pkg.BytesToFloat32_L(b[116:120]),
		CfgSCVDMult: pkg.BytesToFloat32_L(b[120:124]),
		CfgSSPRate:  pkg.BytesToFloat32_L(b[124:128]),
		CfgSSPDur:   pkg.BytesToInt32_L(b[128:132]),
		CfgHiSCVF:   pkg.BytesToFloat32_L(b[132:136]),
		CfgFlowTog:  pkg.BytesToFloat32_L(b[136:140]),
		CfgSSCVFDur: pkg.BytesToInt32_L(b[140:144]),

		CfgVlvTgt: pkg.BytesToInt32_L(b[144:148]),
		CfgVlvPos: pkg.BytesToInt32_L(b[148:152]),

		CfgOpSample: pkg.BytesToInt32_L(b[152:156]),
		CfgOpLog:    pkg.BytesToInt32_L(b[156:160]),
		CfgOpTrans:  pkg.BytesToInt32_L(b[160:164]),

		CfgDiagSample: pkg.BytesToInt32_L(b[164:168]),
		CfgDiagLog:    pkg.BytesToInt32_L(b[168:172]),
		CfgDiagTrans:  pkg.BytesToInt32_L(b[172:176]),
	}
	//  pkg.Json("(demo *DemoDeviceClient)MakeCfgFromBytes() -> cfg", cfg)
	return
}

/*
CONFIG - DEFAULT VALUES
*/
func (cfg *Config) DefaultSettings_Config(reg pkg.DESRegistration) {
	cfg.CfgTime = reg.DESJobRegTime
	cfg.CfgAddr = reg.DESJobRegAddr
	cfg.CfgUserID = reg.DESJobRegUserID
	cfg.CfgApp = reg.DESJobRegApp

	/* JOB */
	cfg.CfgSCVD = 596.8       // m
	cfg.CfgSCVDMult = 10.5    // kPa / m
	cfg.CfgSSPRate = 1.95     // kPa / hour
	cfg.CfgSSPDur = 21600000  // 6 hours
	cfg.CfgHiSCVF = 201.4     //  L/min
	cfg.CfgFlowTog = 1.85     // L/min
	cfg.CfgSSCVFDur = 7200000 // 2 hours

	/* VALVE */
	cfg.CfgVlvTgt = 2 // vent
	cfg.CfgVlvPos = 2 // vent

	/* OP PERIODS*/
	cfg.CfgOpSample = 1000 // millisecond
	cfg.CfgOpLog = 1000    // millisecond
	cfg.CfgOpTrans = 1000  // millisecond

	/* DIAG PERIODS */
	cfg.CfgDiagSample = 10000 // millisecond
	cfg.CfgDiagLog = 100000   // millisecond
	cfg.CfgDiagTrans = 100000 // millisecond
}

/*
CONFIG - VALIDATE FIELDS
*/
func (cfg *Config) Validate() {
	/* TODO: SET ACCEPTABLE LIMITS FOR THE REST OF THE CONFIG SETTINGS */

	cfg.CfgAddr = pkg.ValidateStringLength(cfg.CfgAddr, 36)
	cfg.CfgUserID = pkg.ValidateStringLength(cfg.CfgUserID, 36)
	cfg.CfgApp = pkg.ValidateStringLength(cfg.CfgApp, 36)

	/* ENSURE THE SAMPLE / LOG / TRANS RATES HAVE BEEN SET WITHIN ACCEPTABLE LIMITS */
	if cfg.CfgOpSample < MIN_SAMPLE_PERIOD {
		cfg.CfgOpSample = MIN_SAMPLE_PERIOD
	}
	if cfg.CfgOpLog < cfg.CfgOpSample {
		cfg.CfgOpLog = cfg.CfgOpSample
	}
	if cfg.CfgOpTrans < cfg.CfgOpSample {
		cfg.CfgOpTrans = cfg.CfgOpSample
	}
	if cfg.CfgDiagSample < MIN_SAMPLE_PERIOD {
		cfg.CfgDiagSample = MIN_SAMPLE_PERIOD
	}
	if cfg.CfgDiagLog < cfg.CfgDiagSample {
		cfg.CfgDiagLog = cfg.CfgDiagSample
	}
	if cfg.CfgDiagTrans < cfg.CfgDiagSample {
		cfg.CfgDiagTrans = cfg.CfgDiagSample
	}

	/* ENSURE THE SSP DURATION HAS BEEN SET WITHIN ACCEPTABLE LIMITS */
	if cfg.CfgSSPDur < cfg.CfgOpLog {
		cfg.CfgSSPDur = cfg.CfgOpLog
	}
}

func (cfg *Config) GetMessageSource() (src pkg.DESMessageSource) {
	src.Time = cfg.CfgTime
	src.Addr = cfg.CfgAddr
	src.UserID = cfg.CfgUserID
	src.App = cfg.CfgApp
	return
}

/* CONFIG - VALIDATE CMD REQUEST FROM USER */
func (cfg *Config) CMDValidate(device *Device, uid string) (err error) {

	src := cfg.GetMessageSource()
	dev_src := device.ReferenceSRC()
	if err = src.ValidateSRC_CMD(dev_src, uid, cfg); err != nil {
		return
	}

	return
}

/* CONFIG - VALIDATE MQTT SIG FROM DEVICE */
func (cfg *Config) SIGValidate(device *Device) (err error) {

	src := cfg.GetMessageSource()
	dev_src := device.ReferenceSRC()
	if err = src.ValidateSRC_SIG(dev_src, cfg); err != nil {
		return
	}

	cfg.Validate()

	return
}

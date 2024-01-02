package c001v001

import (
	"encoding/json"
	
	"github.com/leehayford/des/pkg"
)


/* ADM DEMO MEMORY -> JSON*/
func (device Device) WriteAdmToJSONFile(jobName string, adm Admin) (err error) {
	return pkg.WriteModelToJSONFile(jobName, "adm", adm)
}
func (device *Device) ReadLastADMFromJSONFile(jobName string) (adm Admin, err error) {

	buf, err := pkg.ReadModelBytesFromJSONFile(jobName, "adm")
	if err != nil {
		return
	}

	adms := []Admin{}
	if err = json.Unmarshal(buf, &adms); err != nil {
		pkg.LogErr(err)
		return
	} 
	
	for i := len(adms) -1; i <= 0; i-- {
		chk := adms[i]
		if chk.AdmAddr == device.DESDevSerial {
			adm = chk
			break
		}
	} // pkg.Json("(*Device) ReadLastADMFromJSONFile: -> adm: ", adm)

	device.ADM = adm
	return
}

/* STA DEMO MEMORY -> JSON */
func (device Device) WriteStateToJSONFile(jobName string, sta State) (err error) {
	return pkg.WriteModelToJSONFile(jobName, "sta", sta)
}
func (device *Device) ReadLastSTAFromJSONFile(jobName string) (sta State, err error) {

	buf, err := pkg.ReadModelBytesFromJSONFile(jobName, "sta")
	if err != nil {
		return
	}

	stas := []State{}
	if err = json.Unmarshal(buf, &stas); err != nil {
		pkg.LogErr(err)
		return
	} 
	
	for i := len(stas) -1; i <= 0; i-- {
		chk := stas[i]
		if chk.StaAddr == device.DESDevSerial {
			sta = chk
			break
		}
	} 
	pkg.Json("(*Device) ReadLastSTAFromJSONFile: -> sta: ", sta)

	device.STA = sta
	return
}


/* HDR DEMO MEMORY -> JSON */
func (device Device) WriteHdrToJSONFile(jobName string, hdr Header) (err error) {
	return pkg.WriteModelToJSONFile(jobName, "hdr", hdr)
}
func (device *Device) ReadLastHDRFromJSONFile(jobName string) (hdr Header, err error) {

	buf, err := pkg.ReadModelBytesFromJSONFile(jobName, "hdr")
	if err != nil {
		return
	}

	hdrs := []Header{}
	if err = json.Unmarshal(buf, &hdrs); err != nil {
		pkg.LogErr(err)
		return
	} 
	
	for i := len(hdrs) -1; i <= 0; i-- {
		chk := hdrs[i]
		if chk.HdrAddr == device.DESDevSerial {
			hdr = chk
			break
		}
	} // pkg.Json("(*Device) ReadLastHDRFromJSONFile: -> hdr: ", hdr)

	device.HDR = hdr
	return
}


/* CFG DEMO MEMORY -> JSON */
func (device Device) WriteCfgToJSONFile(jobName string, cfg Config) (err error) {
	return pkg.WriteModelToJSONFile(jobName, "cfg", cfg)
}
func (device *Device) ReadLastCFGFromJSONFile(jobName string) (cfg Config, err error) {

	buf, err := pkg.ReadModelBytesFromJSONFile(jobName, "cfg")
	if err != nil {
		return
	}

	cfgs := []Config{}
	if err = json.Unmarshal(buf, &cfgs); err != nil {
		pkg.LogErr(err)
		return
	} 
	
	for i := len(cfgs) -1; i <= 0; i-- {
		chk := cfgs[i]
		if chk.CfgAddr == device.DESDevSerial {
			cfg = chk
			break
		}
	} // pkg.Json("(*Device) ReadLastCFGFromJSONFile: -> cfg: ", cfg)

	device.CFG = cfg
	return
}


/* EVT DEMO MEMORY -> JSON */
func (device Device) WriteEvtToJSONFile(jobName string, evt Event) (err error) {
	return pkg.WriteModelToJSONFile(jobName, "evt", evt)
}
func (device *Device) ReadLastEVTFromJSONFile(jobName string) (evt Event, err error) {

	buf, err := pkg.ReadModelBytesFromJSONFile(jobName, "evt")
	if err != nil {
		return
	}

	evts := []Event{}
	if err = json.Unmarshal(buf, &evts); err != nil {
		pkg.LogErr(err)
		return
	} 
	
	for i := len(evts) -1; i <= 0; i-- {
		chk := evts[i]
		if chk.EvtAddr == device.DESDevSerial {
			evt = chk
			break
		}
	} // pkg.Json("(*Device) ReadLastEVTFromJSONFile: -> evt: ", evt)

	device.EVT = evt
	return
}


/* HEX FILES *************************************************************************************/

/* SMP DEMO MEMORY -> 40 BYTES -> HxD 40 x 1 */
func (device Device) WriteSMPToHEXFile(jobName string, smp Sample) (err error) {

	buf := smp.SampleToBytes() // fmt.Printf("\nsmpBytes ( %d ) : %x\n", len(buf), buf)
	return pkg.WriteModelBytesToHEXFile(jobName, "smp", buf)
}

/* ADM DEMO MEMORY -> 284 BYTES -> HxD 71 x 4 */
func (device Device) WriteADMToHEXFile(jobName string, adm Admin) (err error) {
	buf := adm.AdminToBytes() // fmt.Printf("\nadmBytes ( %d ) : %x\n", len(buf), buf)
	return pkg.WriteModelBytesToHEXFile(jobName, "adm", buf)
}
func (device *Device) ReadLastADMFromHEXFile(jobName string, adm Admin) (err error) {

	buf, err := pkg.ReadModelBytesFromHEXFile(jobName, "adm")
	if err != nil {
		return
	}
	b := buf[len(buf)-284:]
	// fmt.Printf("\nadmBytes ( %d ) : %v\n", len(b), b)
	adm.AdminFromBytes(b)
	return
}

/* STA DEMO MEMORY -> 180 BYTES -> HxD 45 x 4 */
func (device Device) WriteSTAToHEXFile(jobName string, sta State) (err error) {
	buf := sta.StateToBytes() // fmt.Printf("\nstaBytes ( %d ) : %x\n", len(buf), buf)
	return pkg.WriteModelBytesToHEXFile(jobName, "sta", buf)
}
func (device *Device) ReadLastSTAFromHEXFile(jobName string, sta State) (err error) {

	buf, err := pkg.ReadModelBytesFromHEXFile(jobName, "sta")
	if err != nil {
		return
	}
	b := buf[len(buf)-180:]
	// fmt.Printf("\nstaBytes ( %d ) : %v\n", len(b), b)
	sta.StateFromBytes(b)
	return
}

/* HDR DEMO MEMORY -> 308 BYTES -> HxD 44 x 7 */
func (device Device) WriteHDRToHEXFile(jobName string, hdr Header) (err error) {
	buf := hdr.HeaderToBytes() // fmt.Printf("\nhdrBytes ( %d ) : %x\n", len(buf), buf)
	return pkg.WriteModelBytesToHEXFile(jobName, "hdr", buf)
}
func (device *Device) ReadLastHDRFromHEXFile(jobName string, hdr Header) (err error) {

	buf, err := pkg.ReadModelBytesFromHEXFile(jobName, "hdr")
	if err != nil {
		return
	}
	b := buf[len(buf)-308:]
	// fmt.Printf("\nhdrBytes ( %d ) : %v\n", len(b), b)
	hdr.HeaderFromBytes(b)
	return
}

/* CFG DEMO MEMORY -> 176 BYTES -> HxD 44 x 4 */
func (device Device) WriteCFGToHEXFile(jobName string, cfg Config) (err error) {
	buf := cfg.ConfigToBytes() // fmt.Printf("\ncfgBytes ( %d ) : %x\n", len(buf), buf)
	return pkg.WriteModelBytesToHEXFile(jobName, "cfg", buf)
}
func (device *Device) ReadLastCFGFromHEXFile(jobName string, cfg Config) (err error) {

	buf, err := pkg.ReadModelBytesFromHEXFile(jobName, "cfg")
	if err != nil {
		return
	}
	b := buf[len(buf)-176:]
	// fmt.Printf("\ncfgBytes ( %d ) : %v\n", len(b), b)
	cfg.ConfigFromBytes(b)
	return
}

/* EVT DEMO MEMORY -> 668 BYTES -> HxD 167 x 4  */
func (device Device) WriteEVTToHEXFile(jobName string, evt Event) (err error) {
	buf := evt.EventToBytes() // fmt.Printf("\nevtBytes ( %d ) : %x\n", len(buf), buf)
	return pkg.WriteModelBytesToHEXFile(jobName, "evt", buf)
}
func (device *Device) ReadLastEVTFromHEXFile(jobName string, evt *Event) (err error) {

	buf, err := pkg.ReadModelBytesFromHEXFile(jobName, "evt")
	if err != nil {
		return
	}
	b := buf[len(buf)-668:]
	// fmt.Printf("\nevtBytes ( %d ) : %v\n", len(b), b)
	evt.EventFromBytes(b)
	return
}

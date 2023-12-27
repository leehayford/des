package c001v001

import (
	
	"github.com/leehayford/des/pkg"
)


/* ADM DEMO MEMORY -> JSON*/
func (device Device) WriteAdmToJSONFile(jobName string, adm Admin) (err error) {
	return pkg.WriteModelToJSONFile(jobName, "adm", adm)
}

/* STA DEMO MEMORY -> JSON */
func (device Device) WriteStateToJSONFile(jobName string, sta State) (err error) {
	return pkg.WriteModelToJSONFile(jobName, "sta", sta)
}

/* HDR DEMO MEMORY -> JSON */
func (device Device) WriteHdrToJSONFile(jobName string, hdr Header) (err error) {
	return pkg.WriteModelToJSONFile(jobName, "hdr", hdr)
}

/* CFG DEMO MEMORY -> JSON */
func (device Device) WriteCfgToJSONFile(jobName string, cfg Config) (err error) {
	return pkg.WriteModelToJSONFile(jobName, "cfg", cfg)
}

/* EVT DEMO MEMORY -> JSON */
func (device Device) WriteEvtToJSONFile(jobName string, evt Event) (err error) {
	return pkg.WriteModelToJSONFile(jobName, "evt", evt)
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
func (device Device) ReadLastADMFromHEXFile(jobName string, adm *Admin) (err error) {

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
func (device Device) ReadLastSTAFromHEXFile(jobName string, sta *State) (err error) {

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
func (device Device) ReadLastHDRFromHEXFile(jobName string, hdr *Header) (err error) {

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
func (device Device) ReadLastCFGFromHEXFile(jobName string, cfg *Config) (err error) {

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
func (device Device) ReadLastEVTFromHEXFile(jobName string, evt *Event) (err error) {

	buf, err := pkg.ReadModelBytesFromHEXFile(jobName, "evt")
	if err != nil {
		return
	}
	b := buf[len(buf)-668:]
	// fmt.Printf("\nevtBytes ( %d ) : %v\n", len(b), b)
	evt.EventFromBytes(b)
	return
}

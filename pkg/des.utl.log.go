/* Data Exchange Server (DES) is a component of the Datacan Data2Desk (D2D) Platform.
License:

	[PROPER LEGALESE HERE...]

	INTERIM LICENSE DESCRIPTION:
	In spirit, this license:
	1. Allows <Third Party> to use, modify, and / or distributre this software in perpetuity so long as <Third Party> understands:
		a. The software is porvided as is without guarantee of additional support from DataCan in any form.
		b. The software is porvided as is without guarantee of exclusivity.

	2. Prohibits <Third Party> from taking any action which might interfere with DataCan's right to use, modify and / or distributre this software in perpetuity.
*/

package pkg

import (
	"encoding/json"
	"fmt"
	"runtime"
)

func LogErr(err error) error {
	pc, file, line, _ := runtime.Caller(1)
	name := runtime.FuncForPC(pc).Name()

	fmt.Printf("\n***ERROR***\n\tFile :\t%s\n\tFunc :\t%s\n\tLine :\t%d\n\n\t%s\n\n***********\n\n", file, name, line, err.Error())
	/* TODO: LOG err TO FILE */

	return err
}

func LogChk(msg string) {
	pc, file, line, _ := runtime.Caller(1)
	name := runtime.FuncForPC(pc).Name()

	fmt.Printf("\n***OK***\n\tFile :\t%s\n\tFunc :\t%s\n\tLine :\t%d\n\n\t%s\n\n***********\n\n", file, name, line, msg)
	/* TODO: LOG msg TO FILE */
}

func Json(name string, v any) {
	js, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		LogErr(err)
	}
	fmt.Printf("\nJSON: %s:\n%s\n", name, string(js))
}

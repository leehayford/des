
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

func TraceErr(err error) error {
	pc, file, line, _ := runtime.Caller(1)
	name := runtime.FuncForPC(pc).Name()
	/* TODO: LOG THIS SOME PLACE */
	fmt.Printf("\n***ERROR***\n\tFile :\t%s\n\tFunc  :\t%s\n\tLine  :\t%d\ntError :\n\t%s\n\n", file, name, line, err.Error())
	return err
}

func TraceFunc(msg string) {
	pc, file, line, _ := runtime.Caller(1)
	name := runtime.FuncForPC(pc).Name()
	fmt.Printf("\n**************************************************\n%s from:\n\tFile: %s\n\tFunc: %s\n\tLine: %d\n", msg, file, name, line)
}

func Json(name string, v any) {
	js, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		TraceErr(err)
	}
	fmt.Printf("\nJSON: %s:\n%s\n", name, string(js))
}

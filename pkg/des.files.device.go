
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
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

/* JSON FILES ***********************************************************************************/

/*
	CONVERTS MODEL TO JSON STRING AND WRITES TO ~/device_files/dirName/fileName.json

	MODELS APPENDED TO A SINGLE JSON ARRAY [ { 1 }, { 2 }, { 3 } ]
*/
func WriteModelToJSONFile(dirName, fileName string, mod interface{}) (err error) {

	js, err := ModelToJSONString(mod)
	if err != nil {
		LogErr(err)
	}

	dir := fmt.Sprintf("device_files/%s", dirName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		LogErr(err)
	}

	path := fmt.Sprintf("%s/%s.json", dir, fileName)
	
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return LogErr(err)
	} // defer f.Close()
	fi, _ := f.Stat()
	f.Close()

	if fi.Size() == 0 {
		/* SURROUND IN A '[ ]' IF THIS IS THE FIRST RECORD */
		js = fmt.Sprintf("[%s]", js)
	} else {
		/* REMOVE '] AND PREPEND A COMMA IF THIS IS NOT THE FIRST RECORD */
		str, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s.json", dir, fileName))
		trunc := strings.Split(string(str), "]")[0]
		js = fmt.Sprintf("%s,%s]", trunc, js)
	}

	if err = ioutil.WriteFile(path, []byte(js), 644); err != nil {
		return LogErr(err)
	}

	return
}


/* HEX FILES *************************************************************************************/

/* APPENDS MODEL HEX VALUES TO ~/device_files/dirName/fileName.bin */
func WriteModelBytesToHEXFile(dirName, fileName string, buf []byte) (err error) {

	dir := fmt.Sprintf("device_files/%s", dirName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		LogErr(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/%s.bin", dir, fileName), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return LogErr(err)
	}
	defer f.Close()

	_, err = f.Write(buf)
	if err != nil {
		return LogErr(err)
	}

	f.Close()
	return
}

/* RETURNS ALL BYTES FROM ~/device_hex_files/dirName/fileName.bin */
func ReadModelBytesFromHEXFile(jobName, fileName string) (buf []byte, arr error) {

	dir := fmt.Sprintf("device_files/%s", jobName)
	f, err := os.OpenFile(fmt.Sprintf("%s/%s.bin", dir, fileName), os.O_RDONLY, 0600)
	if err != nil {
		return nil, LogErr(err)
	}

	buf, err = ioutil.ReadAll(f)
	f.Close()
	return
}

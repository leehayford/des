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
CONVERTS MODEL TO JSON STRING AND WRITES TO ~/DES_DEVICE_FILES/dirName/fileName.json

MODELS APPENDED TO A SINGLE JSON ARRAY [ { 1 }, { 2 }, { 3 } ]
*/
func WriteModelToJSONFile(dirName, fileName string, mod interface{}) (err error) {
	if fileName == "" {
		return LogErr(fmt.Errorf(ERR_FILE_NAME_EMPTY))
	}
	js, err := ModelToJSONString(mod)
	if err != nil {
		LogErr(err)
	}

	dir := fmt.Sprintf("%s/%s/%s", DATA_DIR, DEVICE_FILE_DIR, dirName)
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

func ReadModelBytesFromJSONFile(dirName, fileName string) (buf []byte, err error) {
	if fileName == "" {
		return nil, LogErr(fmt.Errorf(ERR_FILE_NAME_EMPTY))
	}
	dir := fmt.Sprintf("%s/%s/%s", DATA_DIR, DEVICE_FILE_DIR, dirName)
	f, err := os.OpenFile(fmt.Sprintf("%s/%s.json", dir, fileName), os.O_RDONLY, 0600)
	if err != nil {
		return nil, LogErr(err)
	}

	buf, err = ioutil.ReadAll(f)
	f.Close()
	return
}

/* HEX FILES *************************************************************************************/

/* APPENDS MODEL HEX VALUES TO ~/DES_DEVICE_FILES/dirName/fileName.bin */
func WriteModelBytesToHEXFile(dirName, fileName string, buf []byte) (err error) {
	if fileName == "" {
		return LogErr(fmt.Errorf(ERR_FILE_NAME_EMPTY))
	}
	dir := fmt.Sprintf("%s/%s/%s", DATA_DIR, DEVICE_FILE_DIR, dirName)
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

/* RETURNS ALL BYTES FROM ~/DES_DEVICE_FILES/dirName/fileName.bin */
func ReadModelBytesFromHEXFile(dirName, fileName string) (buf []byte, arr error) {
	if fileName == "" {
		return nil, LogErr(fmt.Errorf(ERR_FILE_NAME_EMPTY))
	}
	dir := fmt.Sprintf("%s/%s/%s", DATA_DIR, DEVICE_FILE_DIR, dirName)
	f, err := os.OpenFile(fmt.Sprintf("%s/%s.bin", dir, fileName), os.O_RDONLY, 0600)
	if err != nil {
		return nil, LogErr(err)
	}

	buf, err = ioutil.ReadAll(f)
	f.Close()
	return
}

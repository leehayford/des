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

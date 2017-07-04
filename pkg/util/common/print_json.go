package common

import (
	"bytes"
	"fmt"
	"encoding/json"
	"runtime"
)

// pretty print a Json structure (for debugging)
func Print_json(item interface{}) error {
	body, err := json.Marshal(item)
	if err != nil { return err}

	var pbody bytes.Buffer
	err = json.Indent(&pbody, body, "", "\t")
	if err != nil { return err}

	fmt.Println(string(pbody.Bytes()))
	return nil
}



const IsDebug = true

func LogDebug(format string, vars ...interface{}) {
	if IsDebug {
		//f := fmt.Sprintf("DEBUG %s %v", time.Now(), format)
		f := fmt.Sprintf("DEBUG %v", format)
		fpcs, _, no, ok := runtime.Caller(1)
		if ok && false {
			fun := runtime.FuncForPC(fpcs)
			f += fmt.Sprintf(" FUNC %s#%d ", fun.Name(), no)
		}
		fmt.Printf("DEBUG "+format+"\n",vars...)
	}
}



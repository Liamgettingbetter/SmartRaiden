package cases

import (
	"fmt"
	"reflect"
	"strings"
)

// CaseManager include env and cases
type CaseManager struct {
	Cases map[string]reflect.Value
}

// NewCaseManager constructor
func NewCaseManager() (caseManager *CaseManager) {
	caseManager = new(CaseManager)
	caseManager.Cases = make(map[string]reflect.Value)
	// use reflect to load all cases
	fmt.Println("load cases...")
	vf := reflect.ValueOf(caseManager)
	vft := vf.Type()
	for i := 0; i < vf.NumMethod(); i++ {
		mName := vft.Method(i).Name
		if strings.Contains(mName, "Case") {
			fmt.Println("CaseName:", mName)
			caseManager.Cases[mName] = vf.Method(i)
		}
	}
	fmt.Printf("Total %d cases load success\n", len(caseManager.Cases))
	fmt.Println("Start Crash Test...")
	return
}

// RunAll run all
func (c *CaseManager) RunAll() {
	fmt.Println("Run all cases...")
	for k, v := range c.Cases {
		rs := v.Call(nil)
		if rs[0].Interface() == nil {
			fmt.Printf("%s SUCCESS\n", k)
		} else {
			err := rs[0].Interface().(error)
			if err == nil {
				fmt.Printf("%s SUCCESS\n", k)
			} else {
				fmt.Printf("%s FAILED!!!\n", k)
				//panic(err)
			}
		}
	}
}

// RunOne run one
func (c *CaseManager) RunOne(caseName string) {
	if v, ok := c.Cases[caseName]; ok {
		rs := v.Call(nil)
		if rs[0].Interface() == nil {
			fmt.Printf("%s SUCCESS\n", caseName)
		} else {
			err := rs[0].Interface().(error)
			if err == nil {
				fmt.Printf("%s SUCCESS\n", caseName)
			} else {
				fmt.Printf("%s FAILED!!!\n", caseName)
				panic(err)
			}
		}
	} else {
		fmt.Printf("%s doesn't exist !!! \n", caseName)
	}
}

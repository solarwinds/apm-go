// © 2023 SolarWinds Worldwide, LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"github.com/solarwinds/apm-go/internal/log"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func toBool(s string) (bool, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "yes" || s == "true" || s == "enabled" {
		return true, nil
	} else if s == "no" || s == "false" || s == "disabled" {
		return false, nil
	}
	return false, fmt.Errorf("cannot convert %s to bool", s)
}

// c must be a pointer to a struct object
func loadEnvsInternal(c interface{}) {
	cv := reflect.Indirect(reflect.ValueOf(c))
	ct := cv.Type()

	if !cv.CanSet() {
		return
	}

	for i := 0; i < ct.NumField(); i++ {
		fieldV := reflect.Indirect(cv.Field(i))
		if !fieldV.CanSet() || ct.Field(i).Anonymous {
			continue
		}

		field := ct.Field(i)
		fieldK := fieldV.Kind()
		if fieldK == reflect.Struct {
			// Need to use its pointer, otherwise it won't be addressable after
			// passed into the nested method
			loadEnvsInternal(getValPtr(cv.Field(i)).Interface())
		} else {
			tagV := field.Tag.Get("env")
			if tagV == "" {
				continue
			}

			envVal := os.Getenv(tagV)
			if envVal == "" {
				continue
			}

			val, err := stringToValue(envVal, fieldV.Type())
			if err == nil {
				setField(c, "Set", field, val)
			}
		}
	}
}

// stringToValue converts a string to a value of the type specified by typ.
func stringToValue(s string, typ reflect.Type) (reflect.Value, error) {
	s = strings.TrimSpace(s)

	var val interface{}
	var err error

	kind := typ.Kind()
	switch kind {
	case reflect.Int:
		if s == "" {
			s = "0"
		}
		val, err = strconv.Atoi(s)
		if err != nil {
			log.Warningf("Ignore invalid int value: %s", s)
		}
	case reflect.Int64:
		if s == "" {
			s = "0"
		}
		val, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			log.Warningf("Ignore invalid int64 value: %s", s)
		}
	case reflect.Float64:
		if s == "" {
			s = "0"
		}
		val, err = strconv.ParseFloat(s, 64)
		if err != nil {
			log.Warningf("Ignore invalid float64 value: %s", s)
		}
	case reflect.String:
		val = s
	case reflect.Bool:
		if s == "" {
			s = "false"
		}
		val, err = toBool(s)
		if err != nil {
			log.Warningf("Ignore invalid bool value: %s", fmt.Errorf("%w: %s", err, s))
		}
	case reflect.Slice:
		if s == "" {
			return reflect.Zero(typ), nil
		} else {
			panic("Slice with non-empty value is not supported")
		}

	default:
		panic(fmt.Sprintf("Unsupported kind: %v, val: %s", kind, s))
	}
	// convert to the target type as `typ` may be a user defined type
	return reflect.ValueOf(val).Convert(typ), err
}

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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToBool(t *testing.T) {
	b, err := toBool("enabled")
	assert.True(t, b)
	assert.Nil(t, err)

	b, err = toBool("disabled")
	assert.False(t, b)
	assert.Nil(t, err)

	b, err = toBool("yes")
	assert.True(t, b)
	assert.Nil(t, err)

	b, err = toBool("true")
	assert.True(t, b)
	assert.Nil(t, err)

	b, err = toBool("YES")
	assert.True(t, b)
	assert.Nil(t, err)

	b, err = toBool("no")
	assert.False(t, b)
	assert.Nil(t, err)

	b, err = toBool("false")
	assert.False(t, b)
	assert.Nil(t, err)

	b, err = toBool("NO")
	assert.False(t, b)
	assert.Nil(t, err)

	b, err = toBool("invalid")
	assert.False(t, b)
	assert.NotNil(t, err)
}

func TestStringToValue(t *testing.T) {
	typInt := reflect.TypeOf(1)
	typInt64 := reflect.TypeOf(int64(1))
	typString := reflect.TypeOf("a")
	typBool := reflect.TypeOf(false)

	type NewStr string
	typNewStr := reflect.TypeOf(NewStr("newStr"))

	stringToValueWrapper := func(s string, typ reflect.Type) reflect.Value {
		val, _ := stringToValue(s, typ)
		return val
	}

	_, err := stringToValue("hi", typInt)
	assert.NotNil(t, err)

	assert.Equal(t, stringToValueWrapper("1", typInt).Interface(), 1)
	assert.Equal(t, stringToValueWrapper("1", typInt).Kind(), reflect.Int)

	assert.Equal(t, stringToValueWrapper("", typInt).Interface(), 0)
	assert.Equal(t, stringToValueWrapper("a", typInt).Interface(), 0)

	assert.Equal(t, stringToValueWrapper("1", typInt64).Interface(), int64(1))
	assert.Equal(t, stringToValueWrapper("1", typInt64).Kind(), reflect.Int64)
	assert.Equal(t, stringToValueWrapper("", typInt64).Interface(), int64(0))
	assert.Equal(t, stringToValueWrapper("a", typInt64).Interface(), int64(0))

	assert.Equal(t, stringToValueWrapper("a", typString).Interface(), "a")
	assert.Equal(t, stringToValueWrapper("a", typString).Kind(), reflect.String)
	assert.Equal(t, stringToValueWrapper("1", typString).Interface(), "1")
	assert.Equal(t, stringToValueWrapper("A", typString).Interface(), "A")

	assert.Equal(t, stringToValueWrapper("yes", typBool).Interface(), true)
	assert.Equal(t, stringToValueWrapper("yes", typBool).Kind(), reflect.Bool)
	assert.Equal(t, stringToValueWrapper("false", typBool).Interface(), false)

	assert.Equal(t, stringToValueWrapper("hello", typNewStr).Interface(), NewStr("hello"))
	assert.Equal(t, stringToValueWrapper("hello", typNewStr).Type(), reflect.TypeOf(NewStr("hello")))
}

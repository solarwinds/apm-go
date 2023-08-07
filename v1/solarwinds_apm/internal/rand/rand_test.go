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

package rand

import (
	"github.com/stretchr/testify/require"
	"math"
	"sync"
	"testing"
)

func TestRandom(t *testing.T) {
	var mut sync.Mutex
	m := make(map[string]bool)
	var wg sync.WaitGroup
	wg.Add(10)
	var startWg sync.WaitGroup
	startWg.Add(1)
	for i := 0; i < 10; i++ {
		go func() {
			startWg.Wait()
			for i := 0; i < 10000; i++ {
				a := [16]byte{}
				Random(a[:])
				key := string(a[:])
				mut.Lock()
				_, ok := m[key]
				require.False(t, ok)
				m[key] = true
				mut.Unlock()
			}
			wg.Done()
		}()
	}
	startWg.Done()
	wg.Wait()
}

func TestRandIntn(t *testing.T) {
	var mut sync.Mutex
	m := make(map[int]bool)
	var wg sync.WaitGroup
	wg.Add(10)
	var startWg sync.WaitGroup
	startWg.Add(1)
	for i := 0; i < 10; i++ {
		go func() {
			startWg.Wait()
			for n := 1; n < 1001; n++ {
				rand := RandIntn(math.MaxInt)
				mut.Lock()
				m[rand] = true
				mut.Unlock()
			}
			wg.Done()
		}()
	}
	startWg.Done()
	wg.Wait()
	require.Len(t, m, 10000)
}

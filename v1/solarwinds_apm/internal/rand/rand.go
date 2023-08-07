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
	cryptorand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"
)

type entropySource struct {
	sync.Mutex
	rng *rand.Rand
}

var entropy = func() *entropySource {
	// To create a random seed, we use crypto/rand instead of
	// time.Now().UnixNano() because the latter is deterministic.
	// Once the seed is generated, we use the faster math/rand to generate
	// random data.
	var seed int64
	_ = binary.Read(cryptorand.Reader, binary.LittleEndian, &seed)
	return &entropySource{
		rng: rand.New(rand.NewSource(seed)),
	}
}()

func Random(bytes []byte) {
	entropy.Lock()
	defer entropy.Unlock()
	_, _ = entropy.rng.Read(bytes)
}

func RandIntn(n int) int {
	entropy.Lock()
	defer entropy.Unlock()
	return entropy.rng.Intn(n)
}

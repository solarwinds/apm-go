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

package reporter

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	xtrCurrentVersion = 2
	oboeMaxTaskIDLen  = 20
	oboeMaxOpIDLen    = 8
)

var (
	errInvalidTaskID = errors.New("invalid task id")
)

// All-zero slice to validate the task ID, do not modify it
var allZeroTaskID = make([]byte, oboeMaxTaskIDLen)

// orchestras tune to the oboe.
type oboeIDs struct{ taskID, opID []byte }

func (ids oboeIDs) validate() error {
	if !bytes.Equal(allZeroTaskID, ids.taskID) {
		return nil
	} else {
		return errInvalidTaskID
	}
}

type oboeMetadata struct {
	version uint8
	ids     oboeIDs
	taskLen int
	opLen   int
	flags   uint8
}

func (md *oboeMetadata) Init() {
	if md == nil {
		return
	}
	md.version = xtrCurrentVersion
	md.taskLen = oboeMaxTaskIDLen
	md.opLen = oboeMaxOpIDLen
	md.ids.taskID = make([]byte, oboeMaxTaskIDLen)
	md.ids.opID = make([]byte, oboeMaxOpIDLen)
}

// randReader provides random IDs, and can be overridden for testing.
// set by default to read from the crypto/rand Reader.
var randReader = rand.Reader

func (md *oboeMetadata) SetRandom() error {
	if md == nil {
		return errors.New("md.SetRandom: nil md")
	}

	if err := md.SetRandomTaskID(randReader); err != nil {
		return err
	}
	return md.SetRandomOpID()
}

// SetRandomTaskID randomize the task ID. It will retry if the random reader returns
// an error or produced task ID is all-zero, which rarely happens though.
func (md *oboeMetadata) SetRandomTaskID(rand io.Reader) (err error) {
	retried := 0
	for retried < 2 {
		if _, err = rand.Read(md.ids.taskID); err != nil {
			break
		}

		if err = md.ids.validate(); err != nil {
			retried++
			continue
		}
		break
	}

	return err
}

func (md *oboeMetadata) SetRandomOpID() error {
	_, err := randReader.Read(md.ids.opID)
	return err
}

/*
 * Pack a metadata struct into a buffer.
 *
 * md       - pointer to the metadata struct
 * task_len - the task_id length to take
 * op_len   - the op_id length to take
 * buf      - the buffer to pack the metadata into
 * buf_len  - the space available in the buffer
 *
 * returns the length of the packed metadata, in terms of uint8_ts.
 */
func (md *oboeMetadata) Pack(buf []byte) (int, error) {
	if md == nil {
		return 0, errors.New("md.Pack: nil md")
	}
	if md.taskLen == 0 || md.opLen == 0 {
		return 0, errors.New("md.Pack: invalid md (0 len)")
	}

	reqLen := md.taskLen + md.opLen + 2

	if len(buf) < reqLen {
		return 0, errors.New("md.Pack: buf too short to pack")
	}

	/*
	 * Flag field layout:
	 *     7    6     5     4     3     2     1     0
	 * +-----+-----+-----+-----+-----+-----+-----+-----+
	 * |                       |     |     |           |
	 * |        version        | oid | opt |    tid    |
	 * |                       |     |     |           |
	 * +-----+-----+-----+-----+-----+-----+-----+-----+
	 *
	 * tid - task id length
	 *          0 <~> 4, 1 <~> 8, 2 <~> 12, 3 <~> 20
	 * oid - op id length
	 *          (oid + 1) * 4
	 * opt - are options present
	 *
	 * version - the version of X-Trace
	 */
	taskBits := (uint8(md.taskLen) >> 2) - 1

	buf[0] = md.version << 4
	if taskBits == 4 {
		buf[0] |= 3
	} else {
		buf[0] |= taskBits
	}
	buf[0] |= ((uint8(md.opLen) >> 2) - 1) << 3

	copy(buf[1:1+md.taskLen], md.ids.taskID)
	copy(buf[1+md.taskLen:1+md.taskLen+md.opLen], md.ids.opID)
	buf[1+md.taskLen+md.opLen] = md.flags

	return reqLen, nil
}

func (md *oboeMetadata) ToString() (string, error) {
	if md == nil {
		return "", errors.New("md.ToString: nil md")
	}
	buf := make([]byte, 64)
	result, err := md.Pack(buf)
	if err != nil {
		return "", err
	}
	// encode as hex
	enc := make([]byte, 2*result)
	l := hex.Encode(enc, buf[:result])
	return strings.ToUpper(string(enc[:l])), nil
}

// Trigger trace signature authentication errors
const (
	ttAuthBadTimestamp   = "bad-timestamp"
	ttAuthNoSignatureKey = "no-signature-key"
	ttAuthBadSignature   = "bad-signature"
)

// TODO: This could live in the `xtrace` package, except it requires
// TODO: the ability to extract the TT Token from oboe settings.
// TODO: Determine a clean/elegant way to clean this up.
func ValidateXTraceOptionsSignature(signature, ts, data string) error {
	var err error
	_, err = tsInScope(ts)
	if err != nil {
		return errors.New(ttAuthBadTimestamp)
	}

	token, err := getTriggerTraceToken()
	if err != nil {
		return errors.New(ttAuthNoSignatureKey)
	}

	if HmacHash(token, []byte(data)) != signature {
		return errors.New(ttAuthBadSignature)
	}
	return nil
}

func HmacHashTT(data []byte) (string, error) {
	token, err := getTriggerTraceToken()
	if err != nil {
		return "", err
	}
	return HmacHash(token, data), nil
}

func HmacHash(token, data []byte) string {
	h := hmac.New(sha1.New, token)
	h.Write(data)
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}

func getTriggerTraceToken() ([]byte, error) {
	setting, ok := getSetting("")
	if !ok {
		return nil, errors.New("failed to get settings")
	}
	if len(setting.triggerToken) == 0 {
		return nil, errors.New("no valid signature key found")
	}
	return setting.triggerToken, nil
}

func tsInScope(tsStr string) (string, error) {
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return "", errors.Wrap(err, "tsInScope")
	}

	t := time.Unix(ts, 0)
	if t.Before(time.Now().Add(time.Minute*-5)) ||
		t.After(time.Now().Add(time.Minute*5)) {
		return "", fmt.Errorf("timestamp out of scope: %s", tsStr)
	}
	return strconv.FormatInt(ts, 10), nil
}

// String returns a hex string representation
func (md *oboeMetadata) String() string {
	mdStr, _ := md.ToString()
	return mdStr
}

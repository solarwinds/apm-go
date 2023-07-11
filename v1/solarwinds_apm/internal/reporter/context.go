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
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	oboeMetadataStringLen = 60
	maskTaskIDLen         = 0x03
	maskOpIDLen           = 0x08
	maskHasOptions        = 0x04
	maskVersion           = 0xF0

	xtrCurrentVersion      = 2
	oboeMaxTaskIDLen       = 20
	oboeMaxOpIDLen         = 8
	oboeMaxMetadataPackLen = 512
)

// x-trace flags
const (
	XTR_FLAGS_NONE    = 0x0
	XTR_FLAGS_SAMPLED = 0x1
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

type oboeContext struct {
	metadata oboeMetadata
	txCtx    *transactionContext
}

type transactionContext struct {
	name string
	// if the trace/transaction is enabled (defined by per-URL transaction filtering)
	enabled bool
	sync.RWMutex
}

type KVMap map[string]interface{}

type Overrides struct {
	ExplicitTS    time.Time
	ExplicitMdStr string
}

// ContextOptions defines the options of creating a context.
type ContextOptions struct {
	// MdStr is the string representation of the X-Trace ID.
	MdStr string
	// URL is used to do the URL-based transaction filtering.
	URL string
	// XTraceOptions represents the X-Trace-Options header.
	XTraceOptions string
	// CB is the callback function to produce the KVs.
	// XTraceOptionsSignature represents the X-Trace-Options-Signature header.
	XTraceOptionsSignature string
	Overrides              Overrides
	CB                     func() KVMap
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

func (ids *oboeIDs) setOpID(opID []byte) {
	copy(ids.opID, opID)
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

func (md *oboeMetadata) Unpack(data []byte) error {
	if md == nil {
		return errors.New("md.Unpack: nil md")
	}

	if len(data) == 0 { // no header to read
		return errors.New("md.Unpack: empty buf")
	}

	flag := data[0]
	var taskLen, opLen int
	var version uint8

	/* don't recognize this? */
	if (flag&maskVersion)>>4 != xtrCurrentVersion {
		return errors.New("md.Unpack: unrecognized X-Trace version")
	}
	version = (flag & maskVersion) >> 4

	taskLen = (int(flag&maskTaskIDLen) + 1) << 2
	if taskLen == 16 {
		taskLen = 20
	}
	opLen = ((int(flag&maskOpIDLen) >> 3) + 1) << 2

	/* do header lengths describe reality? */
	if (taskLen + opLen + 2) > len(data) {
		return errors.New("md.Unpack: wrong header length")
	}

	md.version = version
	md.taskLen = taskLen
	md.opLen = opLen

	md.ids.taskID = data[1 : 1+taskLen]
	md.ids.opID = data[1+taskLen : 1+taskLen+opLen]
	md.flags = data[1+taskLen+opLen]

	return nil
}

func (md *oboeMetadata) FromString(buf string) error {
	if md == nil {
		return errors.New("md.FromString: nil md")
	}

	ubuf := make([]byte, oboeMaxMetadataPackLen)

	// a hex string's length would be an even number
	if len(buf)%2 == 1 {
		return errors.New("md.FromString: hex not even")
	}

	// check if there are more hex bytes than we want
	if len(buf)/2 > oboeMaxMetadataPackLen {
		return errors.New("md.FromString: too many hex bytes")
	}

	// invalid hex?
	ret, err := hex.Decode(ubuf, []byte(buf))
	if ret != len(buf)/2 || err != nil {
		return errors.New("md.FromString: hex not valid")
	}
	ubuf = ubuf[:ret] // truncate buffer to fit decoded bytes
	err = md.Unpack(ubuf)
	if err != nil {
		return err
	}
	return md.ids.validate()
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

func (md *oboeMetadata) opString() string {
	enc := make([]byte, 2*md.opLen)
	l := hex.Encode(enc, md.ids.opID[:md.opLen])
	return strings.ToUpper(string(enc[:l]))
}

func (md *oboeMetadata) isSampled() bool {
	return md.flags&XTR_FLAGS_SAMPLED != 0
}

// A Context is an oboe context that may or not be tracing.
type Context interface {
}

// A Event is an event that may or may not be tracing, created by a Context.
type Event interface {
	ReportContext(c Context, addCtxEdge bool, args ...interface{}) error
	MetadataString() string
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

func (ctx *oboeContext) newEventWithExplicitID(label Label, layer string, xTraceID string) (*event, error) {
	return newEvent(&ctx.metadata, label, layer, xTraceID)
}

// Create and report and event using a map of KVs
func (ctx *oboeContext) ReportEventMap(label Label, layer string, keys map[string]interface{}) error {
	return ctx.reportEventMap(label, layer, true, keys)
}

func (ctx *oboeContext) reportEventMapWithOverrides(label Label, layer string, addCtxEdge bool, overrides Overrides, keys map[string]interface{}) error {
	var args []interface{}
	for k, v := range keys {
		args = append(args, k)
		args = append(args, v)
	}
	return ctx.reportEventWithOverrides(label, layer, addCtxEdge, overrides, args...)
}

func (ctx *oboeContext) reportEventMap(label Label, layer string, addCtxEdge bool, keys map[string]interface{}) error {
	return ctx.reportEventMapWithOverrides(label, layer, addCtxEdge, Overrides{}, keys)
}

// Create and report an event using KVs from variadic args
func (ctx *oboeContext) ReportEvent(label Label, layer string, args ...interface{}) error {
	return ctx.reportEventWithOverrides(label, layer, true, Overrides{}, args...)
}

func (ctx *oboeContext) ReportEventWithOverrides(label Label, layer string, overrides Overrides, args ...interface{}) error {
	return ctx.reportEventWithOverrides(label, layer, true, overrides, args...)
}

func (ctx *oboeContext) reportEvent(label Label, layer string, addCtxEdge bool, args ...interface{}) error {
	return ctx.reportEventWithOverrides(label, layer, addCtxEdge, Overrides{}, args)
}

// Create and report an event using KVs from variadic args
func (ctx *oboeContext) reportEventWithOverrides(label Label, layer string, addCtxEdge bool, overrides Overrides, args ...interface{}) error {
	// create new event from context
	e, err := ctx.newEventWithExplicitID(label, layer, overrides.ExplicitMdStr)
	if err != nil { // error creating event (e.g. couldn't init random IDs)
		return err
	}

	return ctx.report(e, addCtxEdge, overrides, args...)
}

// report an event using KVs from variadic args
func (ctx *oboeContext) report(e *event, addCtxEdge bool, overrides Overrides, args ...interface{}) error {
	for i := 0; i+1 < len(args); i += 2 {
		if err := e.AddKV(args[i], args[i+1]); err != nil {
			return err
		}
	}

	if addCtxEdge {
		e.AddEdge(ctx)
	}
	e.overrides = overrides
	// report event
	return e.Report(ctx)
}

func (ctx *oboeContext) MetadataString() string { return ctx.metadata.String() }

// String returns a hex string representation
func (md *oboeMetadata) String() string {
	mdStr, _ := md.ToString()
	return mdStr
}

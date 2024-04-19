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

package hdrhist

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
)

func EncodeCompressed(h *Hist) ([]byte, error) {
	var buf bytes.Buffer
	b64w := base64.NewEncoder(base64.StdEncoding, &buf)
	if err := encodeCompressed(h, b64w, h.Max()); err != nil {
		_ = b64w.Close()
		return nil, errors.Wrap(err, "unable to encode histogram")
	}
	// DO NOT defer this close, otherwise that could prevent bytes from getting flushed
	if err := b64w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeCompressed(h *Hist, w io.Writer, histMax int64) error {
	const compressedEncodingCookie = compressedEncodingV2CookieBase | 0x10
	var buf bytes.Buffer

	var cookie int32 = compressedEncodingCookie
	err := binary.Write(&buf, binary.BigEndian, cookie)
	if err != nil {
		return err
	}
	buf.WriteString("\x00\x00\x00\x00")
	preCompressed := buf.Len()
	zw, _ := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err = encodeInto(h, zw, histMax); err != nil {
		return err
	}
	if err = zw.Close(); err != nil {
		return err
	}
	binary.BigEndian.PutUint32(buf.Bytes()[4:], uint32(buf.Len()-preCompressed))

	_, err = buf.WriteTo(w)
	return errors.Wrap(err, "unable to write compressed hist")
}

func encodeInto(h *Hist, w io.Writer, max int64) error {
	const encodingCookie = encodingV2CookieBase | 0x10

	importantLen := h.b.countsIndex(max) + 1
	var buf bytes.Buffer
	var cookie int32 = encodingCookie
	if err := binary.Write(&buf, binary.BigEndian, cookie); err != nil {
		return err
	}
	buf.WriteString("\x00\x00\x00\x00") // length will go here
	buf.WriteString("\x00\x00\x00\x00") // normalizing index offset
	cfg := h.Config()
	if err := binary.Write(&buf, binary.BigEndian, cfg.SigFigs); err != nil {
		return err
	}
	if err := binary.Write(&buf, binary.BigEndian, cfg.LowestDiscernible); err != nil {
		return err
	}
	if err := binary.Write(&buf, binary.BigEndian, cfg.HighestTrackable); err != nil {
		return err
	}
	// int to double conversion ratio
	buf.WriteString("\x3f\xf0\x00\x00\x00\x00\x00\x00")
	payloadStart := buf.Len()
	fillBuffer(&buf, h, importantLen)
	binary.BigEndian.PutUint32(buf.Bytes()[4:], uint32(buf.Len()-payloadStart))
	_, err := buf.WriteTo(w)
	return errors.Wrap(err, "unable to write uncompressed hist")
}

func fillBuffer(buf *bytes.Buffer, h *Hist, n int) {
	srci := 0
	for srci < n {
		// V2 format uses a ZigZag LEB128-64b9B encoded int64.
		// Positive values are counts, negative values indicate
		// a run zero counts of that length.
		c := h.b.counts[srci]
		srci++
		if c < 0 {
			panic(errors.Errorf(
				"can't encode hist with negative counts (count: %d, idx: %d, value range: [%d, %d])",
				c,
				srci,
				h.b.lowestEquiv(h.b.valueFor(srci)),
				h.b.highestEquiv(h.b.valueFor(srci)),
			))
		}

		// count zeros run length
		zerosCount := int64(0)
		if c == 0 {
			zerosCount++
			for srci < n && h.b.counts[srci] == 0 {
				zerosCount++
				srci++
			}
		}
		if zerosCount > 1 {
			buf.Write(encodeZigZag(-zerosCount))
		} else {
			buf.Write(encodeZigZag(c))
		}
	}
}

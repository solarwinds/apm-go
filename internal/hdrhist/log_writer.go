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
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
)

type LogWriter struct {
	w        io.Writer
	buf      bytes.Buffer
	baseTime *time.Time
}

func (l *LogWriter) WriteStartTime(start time.Time) error {
	const JavaDate = "Mon Jan 02 15:04:05 MST 2006"

	sec := start.Unix()
	millis := float64(start.Nanosecond()) / 1e6 // Java version only stores ms

	_, err := fmt.Fprintf(l.w, "#[StartTime: %.3f (seconds since epoch), %s]\n",
		float64(sec)+millis, start.Format(JavaDate))
	return errors.Wrap(err, "unable to write start time")
}

func (l *LogWriter) WriteBaseTime(base time.Time) error {
	sec := base.Unix()
	millis := float64(base.Nanosecond()) / 1e6 // Java version only stores ms
	_, err := fmt.Fprintf(l.w, "#[BaseTime: %.3f (seconds since epoch)]\n", float64(sec)+millis)
	return errors.Wrap(err, "unable to write base time")
}

func (l *LogWriter) WriteComment(text string) error {
	_, err := l.w.Write([]byte("#" + text + "\n"))
	return errors.Wrapf(err, "unable to write comment")
}

var logWriterLegend = []byte("\"StartTimestamp\",\"Interval_Length\",\"Interval_Max\",\"Interval_Compressed_Histogram\"\n")

func (l *LogWriter) WriteLegend() error {
	_, err := l.w.Write(logWriterLegend)
	return err
}

func (l *LogWriter) SetBaseTime(base time.Time) {
	l.baseTime = &base
}

func (l *LogWriter) GetBaseTime() (time.Time, bool) {
	if l.baseTime == nil {
		return time.Time{}, false
	}
	return *l.baseTime, true
}

func (l *LogWriter) WriteIntervalHist(h *Hist) error {
	t, ok := h.StartTime()
	e, okEnd := h.EndTime()
	if ok && okEnd {
		if b, ok := l.GetBaseTime(); ok {
			d := e.Sub(b)
			t = time.Unix(int64(d/time.Second), int64(d%time.Second))
		}
	}
	return l.writeHist(h, t, e)
}

func (l *LogWriter) writeHist(h *Hist, start time.Time, end time.Time) error {
	const MaxValueUnitRatio = 1000000.0
	l.buf.Reset()
	max := h.Max()
	fmt.Fprintf(&l.buf, "%.3f,%.3f,%.3f,",
		float64(start.Unix())+(float64(start.Nanosecond()/1e6)/1e3),
		float64(end.Sub(start)/1e6)/1e3,
		float64(max)/MaxValueUnitRatio)
	b64w := base64.NewEncoder(base64.StdEncoding, &l.buf)
	if err := encodeCompressed(h, b64w, max); err != nil {
		return err
	}
	if err := b64w.Close(); err != nil {
		return err
	}
	l.buf.WriteString("\n")
	_, err := l.buf.WriteTo(l.w)
	return errors.Wrap(err, "unable to write hist")
}

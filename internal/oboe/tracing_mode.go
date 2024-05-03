package oboe

import "github.com/solarwinds/apm-go/internal/config"

type TracingMode int

const (
	TraceDisabled TracingMode = iota // disable tracing, will neither start nor continue traces
	TraceEnabled                     // perform sampling every inbound request for tracing
	TraceUnknown                     // for cache purpose only
)

// NewTracingMode creates a tracing mode object from a string
func NewTracingMode(mode config.TracingMode) TracingMode {
	switch mode {
	case config.DisabledTracingMode:
		return TraceDisabled
	case config.EnabledTracingMode:
		return TraceEnabled
	default:
	}
	return TraceUnknown
}

func (tm TracingMode) isUnknown() bool {
	return tm == TraceUnknown
}

func (tm TracingMode) toFlags() settingFlag {
	switch tm {
	case TraceEnabled:
		return FlagSampleStart | FlagSampleThroughAlways | FlagTriggerTrace
	case TraceDisabled:
	default:
	}
	return FlagOk
}

func (tm TracingMode) ToString() string {
	switch tm {
	case TraceEnabled:
		return string(config.EnabledTracingMode)
	case TraceDisabled:
		return string(config.DisabledTracingMode)
	default:
		return string(config.UnknownTracingMode)
	}
}

package reporter

type settingType int
type settingFlag uint16

// setting types
const (
	TYPE_DEFAULT settingType = iota // default setting which serves as a fallback if no other settings found
	TYPE_LAYER                      // layer specific settings
)

// setting flags offset
const (
	FlagInvalidOffset = iota
	FlagOverrideOffset
	FlagSampleStartOffset
	FlagSampleThroughOffset
	FlagSampleThroughAlwaysOffset
	FlagTriggerTraceOffset
)

// setting flags
const (
	FLAG_OK                    settingFlag = 0x0
	FLAG_INVALID               settingFlag = 1 << FlagInvalidOffset
	FLAG_OVERRIDE              settingFlag = 1 << FlagOverrideOffset
	FLAG_SAMPLE_START          settingFlag = 1 << FlagSampleStartOffset
	FLAG_SAMPLE_THROUGH        settingFlag = 1 << FlagSampleThroughOffset
	FLAG_SAMPLE_THROUGH_ALWAYS settingFlag = 1 << FlagSampleThroughAlwaysOffset
	FLAG_TRIGGER_TRACE         settingFlag = 1 << FlagTriggerTraceOffset
)

// Enabled returns if the trace is enabled or not.
func (f settingFlag) Enabled() bool {
	return f&(FLAG_SAMPLE_START|FLAG_SAMPLE_THROUGH_ALWAYS) != 0
}

// TriggerTraceEnabled returns if the trigger trace is enabled
func (f settingFlag) TriggerTraceEnabled() bool {
	return f&FLAG_TRIGGER_TRACE != 0
}

func (st settingType) toSampleSource() sampleSource {
	var source sampleSource
	switch st {
	case TYPE_DEFAULT:
		source = SAMPLE_SOURCE_DEFAULT
	case TYPE_LAYER:
		source = SAMPLE_SOURCE_LAYER
	default:
		source = SAMPLE_SOURCE_NONE
	}
	return source
}

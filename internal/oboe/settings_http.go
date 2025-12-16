package oboe

type httpSettings struct {
	Arguments *httpSettingArguments `json:"arguments"`
	Flags     string                `json:"flags"`
	Timestamp int64                 `json:"timestamp"`
	Ttl       int64                 `json:"ttl"`
	Value     int64                 `json:"value"`
}

type httpSettingArguments struct {
	BucketCapacity               float64 `json:"BucketCapacity"`
	BucketRate                   float64 `json:"BucketRate"`
	MetricsFlushInterval         int     `json:"MetricsFlushInterval"`
	TriggerRelaxedBucketCapacity float64 `json:"TriggerRelaxedBucketCapacity"`
	TriggerRelaxedBucketRate     float64 `json:"TriggerRelaxedBucketRate"`
	TriggerStrictBucketCapacity  float64 `json:"TriggerStrictBucketCapacity"`
	TriggerStrictBucketRate      float64 `json:"TriggerStrictBucketRate"`
}

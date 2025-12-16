package oboe

type nullSettingsPoller struct{}

func newNullSettingsPoller() SettingsPoller {
	return &nullSettingsPoller{}
}

func (nsp *nullSettingsPoller) Start() {
	// no-op
}

func (nsp *nullSettingsPoller) Shutdown() {
	// no-op
}

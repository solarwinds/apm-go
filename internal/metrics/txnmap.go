package metrics

import "sync"

// txnMap records the received transaction names in a metrics report cycle. It will refuse
// new transaction names if reaching the capacity.
type txnMap struct {
	// The map to store transaction names
	transactionNames map[string]struct{}
	// The maximum capacity of the transaction map. The value is got from server settings which
	// is updated periodically.
	// The default value metricsTransactionsMaxDefault is used when a new txnMap
	// is initialized.
	currCap int32
	// The maximum capacity which is set by the server settings. This update usually happens in
	// between two metrics reporting cycles. To avoid affecting the map capacity of the current reporting
	// cycle, the new capacity got from the server is stored in nextCap and will only be flushed to currCap
	// when the reset() is called.
	nextCap int32
	// Whether there is an overflow. isOverflowed means the user tried to store more transaction names
	// than the capacity defined by settings.
	// This flag is cleared in every metrics cycle.
	overflow bool
	// The mutex to protect this whole struct. If the performance is a concern we should use separate
	// mutexes for each of the fields. But for now it seems not necessary.
	mutex sync.Mutex
}

// newTxnMap initializes a new txnMap struct
func newTxnMap(cap int32) *txnMap {
	return &txnMap{
		transactionNames: make(map[string]struct{}),
		currCap:          cap,
		nextCap:          cap,
		overflow:         false,
	}
}

// SetCap sets the capacity of the transaction map
func (t *txnMap) SetCap(cap int32) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.nextCap = cap
}

// cap returns the current capacity
func (t *txnMap) cap() int32 {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.currCap
}

// reset resets the transaction map to a initialized state. The new capacity got from the
// server will be used in next metrics reporting cycle after reset.
func (t *txnMap) reset() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.transactionNames = make(map[string]struct{})
	t.currCap = t.nextCap
	t.overflow = false
}

// clone returns a shallow copy
func (t *txnMap) clone() *txnMap {
	return &txnMap{
		transactionNames: t.transactionNames,
		currCap:          t.currCap,
		nextCap:          t.nextCap,
		overflow:         t.overflow,
	}
}

// isWithinLimit checks if the transaction name is stored in the txnMap. It will store this new
// transaction name and return true if not stored before and the map isn't full, or return false
// otherwise.
func (t *txnMap) isWithinLimit(name string) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if _, ok := t.transactionNames[name]; !ok {
		// only record if we haven't reached the limits yet
		if int32(len(t.transactionNames)) < t.currCap {
			t.transactionNames[name] = struct{}{}
			return true
		}
		t.overflow = true
		return false
	}

	return true
}

// isOverflowed returns true is the transaction map is overflow (reached its limit)
// or false if otherwise.
func (t *txnMap) isOverflowed() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.overflow
}

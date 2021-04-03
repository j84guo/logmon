package http

import "container/list"

type HttpLogAlerter struct {
	maxPerSecond uint64
	numSecondsInPeriod uint64
	periodThreshold uint64
	period *list.List
	numHitsInPeriod uint64
	numHitsInCurrentSecond uint64
	alertActive bool
	lastTimestamp uint64
}

func NewHttpLogAlerter(maxPerSecond uint64, numSecondsInPeriod uint64,) *HttpLogAlerter {
	return &HttpLogAlerter {
		maxPerSecond: maxPerSecond,
		numSecondsInPeriod: numSecondsInPeriod,
		periodThreshold: maxPerSecond * numSecondsInPeriod,
		period: list.New(),
		numHitsInPeriod: 0,
		numHitsInCurrentSecond: 0,
		alertActive: false,
		lastTimestamp: 0,
	}
}

func (hla *HttpLogAlerter) AddLog(ts uint64) {
	// Increment hits in current second, remember timestamp of most recent log
	hla.numHitsInCurrentSecond += 1
	hla.lastTimestamp = ts
}

func (hla *HttpLogAlerter) StartNextSecond() {
	// Remove oldest second in period if we need to make room
	if uint64(hla.period.Len()) == hla.numSecondsInPeriod {
		hla.numHitsInPeriod -= hla.period.Front().Value.(uint64)
		hla.period.Remove(hla.period.Front())
	}
	// Add the current second to the period
	hla.numHitsInPeriod += hla.numHitsInCurrentSecond
	hla.period.PushBack(hla.numHitsInCurrentSecond)
	hla.numHitsInCurrentSecond = 0
}

func (hla *HttpLogAlerter) IsAlertTriggered() bool {
	if uint64(hla.period.Len()) == hla.numSecondsInPeriod {
		if hla.numHitsInPeriod > hla.periodThreshold {
			if !hla.alertActive {
				hla.alertActive = true
				return true
			}
		}
	}
	return false
}

func (hla *HttpLogAlerter) IsAlertRecovered() bool {
	if uint64(hla.period.Len()) == hla.numSecondsInPeriod {
		if hla.numHitsInPeriod < hla.periodThreshold {
			if hla.alertActive {
				hla.alertActive = false
				return true
			}
		}
	}
	return false
}

func (hla *HttpLogAlerter) GetNumHitsInPeriod() uint64 {
	return hla.numHitsInPeriod
}

func (hla *HttpLogAlerter) GetLastTimestamp() uint64 {
	return hla.lastTimestamp
}

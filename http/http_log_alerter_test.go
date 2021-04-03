package http

import "testing"

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}

func TestAlertNotTriggeredBySmallWindow(t *testing.T) {
	const maxPerSecond = uint64(10)
	const numSecondsInPeriod = uint64(120)

	hla := NewHttpLogAlerter(10, numSecondsInPeriod)

	for i := uint64(0); i < numSecondsInPeriod - 1; i += 1 {
		for j := uint64(0); j <= maxPerSecond; j += 1 {
			hla.AddLog(0)
		}
		hla.StartNextSecond()
	}

	// No alert since window not 120 seconds yet
	assertEqual(t, hla.IsAlertTriggered(), false)
	assertEqual(t, hla.IsAlertRecovered(), false)
}

func TestAlertNotTriggeredByLowTraffic(t *testing.T) {
	const maxPerSecond = uint64(10)
	const numSecondsInPeriod = uint64(120)

	hla := NewHttpLogAlerter(maxPerSecond, numSecondsInPeriod)

	for i := uint64(0); i < numSecondsInPeriod; i += 1 {
		hla.AddLog(0)
		hla.StartNextSecond()
	}

	// No alert since traffic low
	assertEqual(t, hla.IsAlertTriggered(), false)
	assertEqual(t, hla.IsAlertRecovered(), false)
}

func TestAlertTriggered(t *testing.T) {
	const maxPerSecond = uint64(10)
	const numSecondsInPeriod = uint64(120)

	hla := NewHttpLogAlerter(maxPerSecond, numSecondsInPeriod)

	// Each second has high traffic
	for i := uint64(0); i < numSecondsInPeriod; i += 1 {
		for j := uint64(0); j <= maxPerSecond; j += 1 {
			hla.AddLog(0)
		}
		hla.StartNextSecond()
	}

	// Alert triggered
	assertEqual(t, hla.IsAlertTriggered(), true)
	assertEqual(t, hla.IsAlertRecovered(), false)
}

func TestAlertNotDuplicated(t *testing.T) {
	const maxPerSecond = uint64(10)
	const numSecondsInPeriod = uint64(120)

	hla := NewHttpLogAlerter(maxPerSecond, numSecondsInPeriod)

	for i := uint64(0); i < numSecondsInPeriod; i += 1 {
		for j := uint64(0); j <= maxPerSecond; j += 1 {
			hla.AddLog(0)
		}
		hla.StartNextSecond()
	}

	// Alert triggered
	assertEqual(t, hla.IsAlertTriggered(), true)
	assertEqual(t, hla.IsAlertRecovered(), false)

	// Next second also has high traffic
	for j := uint64(0); j <= maxPerSecond; j += 1 {
		hla.AddLog(0)
	}
	hla.StartNextSecond()

	// Alert not duplicated
	assertEqual(t, hla.IsAlertTriggered(), false)
	assertEqual(t, hla.IsAlertRecovered(), false)
}

func TestAlertRecovered(t *testing.T) {
	const maxPerSecond = uint64(10)
	const numSecondsInPeriod = uint64(120)

	hla := NewHttpLogAlerter(maxPerSecond, numSecondsInPeriod)

	for i := uint64(0); i < numSecondsInPeriod; i += 1 {
		for j := uint64(0); j < maxPerSecond; j += 1 {
			hla.AddLog(0)
		}
		if i == numSecondsInPeriod - 1 {
			hla.AddLog(0)
		}
		hla.StartNextSecond()
	}

	// Hits in period go over threshold by 1 - alert triggered
	assertEqual(t, hla.IsAlertTriggered(), true)
	assertEqual(t, hla.IsAlertRecovered(), false)

	// No logs in next second - alert recovered
	hla.StartNextSecond()
	assertEqual(t, hla.IsAlertTriggered(), false)
	assertEqual(t, hla.IsAlertRecovered(), true)
}
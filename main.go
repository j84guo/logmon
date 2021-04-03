package main

import (
	"encoding/csv"
	"fmt"
	logmonHttp "github.com/j84guo/logmon/http"
	"log"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"
)

func printTrafficReport(hlt *logmonHttp.HttpLogTracker) {
	report := "[traffic report]\n"
	for _, sectionStats := range hlt.GetStatsByFrequency() {
		report += fmt.Sprintf("> section: /%v\n  total hits: %v\n", sectionStats.Section, sectionStats.NumLogs)
		// Print hits by method
		first := true
		report += "  hits by method: "
		for method, n := range sectionStats.NumLogsByMethod {
			if first {
				first = false
			} else {
				report += ", "
			}
			report += fmt.Sprintf("%v=%v", method, n)
		}
		report += "\n"
		// Print hits by status
		first = true
		report += "  hits by status: "
		for status, n := range sectionStats.NumLogsByStatus {
			if first {
				first = false
			} else {
				report += ", "
			}
			report += fmt.Sprintf("%v=%v", status, n)
		}
		report += "\n"
	}
	// We make a single print statement per report so that messages from concurrent goroutines don't interleave
	fmt.Print(report)
}

func runTrafficReporting(ticker *time.Ticker, httpLogChan <-chan *logmonHttp.HttpLog, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	hlt := logmonHttp.NewHttpLogTracker()
	for {
		select {
		case <-ticker.C:
			// Print report and reset tracker
			printTrafficReport(hlt)
			hlt = logmonHttp.NewHttpLogTracker()
		case hl := <-httpLogChan:
			// Done
			if hl == nil {
				return
			}
			// Add log to tracker
			hlt.AddLog(hl)
		}
	}
}

func runTrafficAlerting(httpLogChan <-chan *logmonHttp.HttpLog, maxHitsPerSecond uint64, numSecondsInPeriod uint64, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	if maxHitsPerSecond <= 0 || numSecondsInPeriod <= 0 {
		panic("maxHitsPerSecond and numSecondsInPeriod must both be greater than zero")
	}

	// Check for alerts each second
	ticker := time.NewTicker(1 * time.Second)

	// Alerting state and logic
	hla := logmonHttp.NewHttpLogAlerter(maxHitsPerSecond, numSecondsInPeriod)

	for {
		select {
		case <-ticker.C:
			// Update sliding window
			hla.StartNextSecond()
			// Check for alert triggered or resolved
			if hla.IsAlertTriggered() {
				fmt.Println(fmt.Sprintf("[alert] High traffic generated an alert - hits = %v, triggered at %v", hla.GetNumHitsInPeriod(), hla.GetLastTimestamp()))
			} else if hla.IsAlertRecovered() {
				fmt.Println(fmt.Sprintf("[alert] Low traffic resolved an alert - hits = %v, triggered at %v", hla.GetNumHitsInPeriod(), hla.GetLastTimestamp()))
			}
		case hl := <-httpLogChan:
			// Done
			if hl == nil {
				return
			}
			// Add log to current second's count
			hla.AddLog(hl.Timestamp)
		}
	}
}

func checkCsvHeaderOrDie(reader *csv.Reader) {
	actualCsvHeader, _ := reader.Read()
	if !reflect.DeepEqual(logmonHttp.GetExpectedLogFields(), actualCsvHeader) {
		log.Fatalf("Expected header %v, but got %v\n", logmonHttp.GetExpectedLogFields(), actualCsvHeader)
	}
}

func main() {
	// Optional argument
	maxHitsPerSecond := uint64(10)
	if len(os.Args) > 1 {
		n, e := strconv.ParseUint(os.Args[1], 10, 64)
		if e != nil {
			log.Fatalf("usage: %v [maxHitsPersecond]", os.Args[0])
		}
		maxHitsPerSecond = n
	}

	// Read CSV from stdin
	reader := csv.NewReader(os.Stdin)
	checkCsvHeaderOrDie(reader)

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	waitGroup.Add(1)

	// Print a report every 10 seconds
	reportingTicker := time.NewTicker(10 * time.Second)
	reportingChan := make(chan *logmonHttp.HttpLog, 1024)
	go runTrafficReporting(reportingTicker, reportingChan, &waitGroup)

	// Check for alerts every 1 second
	alertingChan := make(chan *logmonHttp.HttpLog, 1024)
	go runTrafficAlerting(alertingChan, maxHitsPerSecond, 120, &waitGroup)

	// Send logs to goroutines
	for {
		values, e := reader.Read()
		if e != nil {
			break
		}
		hl, e := logmonHttp.NewHttpLog(values)
		if e != nil {
			log.Printf("Ignoring invalid log %v due to error: %v\n", values, e)
		}
		reportingChan <- hl
		alertingChan <- hl
	}

	// Signal goroutines to finish and wait for them
	reportingChan <- nil
	alertingChan <- nil
	waitGroup.Wait()
}

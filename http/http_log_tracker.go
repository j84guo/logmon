package http

import "container/list"

type SectionStats struct {
	// Number of requests to that Section
	Section string
	NumLogs uint64
	NumLogsByMethod map[string]uint64
	NumLogsByStatus map[uint16]uint64
	// Bucket in which the Section is currently held
	bucket *list.Element
}

func (sectionStats *SectionStats) addLog(hl *HttpLog) {
	sectionStats.NumLogs += 1
	n, _ := sectionStats.NumLogsByMethod[hl.Method]
	sectionStats.NumLogsByMethod[hl.Method] = n + 1
	n, _ = sectionStats.NumLogsByStatus[hl.Status]
	sectionStats.NumLogsByStatus[hl.Status] = n + 1
}

type sectionBucket struct {
	// Number of requests for sections in this bucket
	numLogs uint64
	// Set of Section names
	sections map[string]bool
}

// Basically an LFU-type structure which allows keys (section names) to be incremented, while maintaining the keys in
// some sorted order (by hit count).
type HttpLogTracker struct {
	// Map Section to Status
	statsBySection map[string]*SectionStats
	// Buckets of Section names sorted by number of logs
	sections      *list.List
}

func NewHttpLogTracker() *HttpLogTracker {
	return &HttpLogTracker{
		statsBySection: make(map[string]*SectionStats),
		sections:       list.New(),
	}
}

func (hlt *HttpLogTracker) ContainsSection(section string) bool {
	_, ok := hlt.statsBySection[section]
	return ok
}

func (hlt *HttpLogTracker) NumSections() int {
	return len(hlt.statsBySection)
}

// Increment the hit count of a section
func (hlt *HttpLogTracker) AddLog(hl *HttpLog) {
	section := hl.GetSection()
	if !hlt.ContainsSection(section) {
		// Create new bucket of count 1, or add to existing
		var bucket *list.Element
		if hlt.NumSections() == 0 || hlt.sections.Front().Value.(*sectionBucket).numLogs != 1 {
			bucket = hlt.sections.PushFront(&sectionBucket {
				numLogs: 1,
				sections: map[string]bool{section: true},
			})
		} else {
			bucket = hlt.sections.Front();
			bucket.Value.(*sectionBucket).sections[section] = true
		}
		// Create new stats
		hlt.statsBySection[section] = &SectionStats{
			Section: section,
			NumLogs: 1,
			NumLogsByMethod: map[string]uint64{hl.Method: 1},
			NumLogsByStatus: map[uint16]uint64{hl.Status: 1},
			bucket:  bucket,
		}
	} else {
		// Update stats
		stats := hlt.statsBySection[section]
		stats.addLog(hl)
		// Get next bucket, or create new next bucket, and add to it
		var nextBucket *list.Element
		if stats.bucket.Next() == nil || stats.bucket.Next().Value.(*sectionBucket).numLogs != stats.NumLogs {
			nextBucket = hlt.sections.InsertAfter(&sectionBucket{
				numLogs: stats.NumLogs,
				sections: map[string]bool{section: true},
			}, stats.bucket)
		} else {
			nextBucket = stats.bucket.Next()
			nextBucket.Value.(*sectionBucket).sections[section] = true
		}
		// Remove from existing bucket, deleting existing bucket if empty
		delete(stats.bucket.Value.(*sectionBucket).sections, section)
		if len(stats.bucket.Value.(*sectionBucket).sections) == 0 {
			hlt.sections.Remove(stats.bucket)
		}
		// Set new bucket on stats struct
		stats.bucket = nextBucket
	}
}

// Retrieve stats for each section, ordered by the section's number of hits
func (hlt *HttpLogTracker) GetStatsByFrequency() []SectionStats {
	statsList := make([]SectionStats, 0)
	for node := hlt.sections.Back(); node != nil; node = node.Prev() {
		bucket := *node.Value.(*sectionBucket)
		for section := range bucket.sections {
			stats := *hlt.statsBySection[section]
			stats.bucket = nil
			statsList = append(statsList, stats)
		}
	}
	return statsList
}

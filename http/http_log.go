package http

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type HttpLog struct {
	RemoteHost    string
	Rfc931        string
	AuthUser      string
	Timestamp     uint64
	Method        string
	Path          string
	Version       string
	Status        uint16
	ResponseBytes uint64
}

func GetExpectedLogFields() []string {
	return []string{"remotehost", "rfc931", "authuser", "date", "request", "status", "bytes"}
}

func NewHttpLog(values []string) (*HttpLog, error) {
	if len(values) != len(GetExpectedLogFields()) {
		return nil, errors.New(fmt.Sprintf("Expected %v values but got %v", len(values), len(GetExpectedLogFields())))
	}
	timestamp, e := strconv.ParseUint(values[3], 10, 64)
	if e != nil {
		return nil, e
	}
	requestLine := strings.Fields(values[4])
	if len(requestLine) != 3 {
		return nil, errors.New(fmt.Sprintf("Invalid request line: %v", requestLine))
	}
	status, e := strconv.ParseUint(values[5], 10, 16)
	if e != nil {
		return nil, e
	}
	responseBytes, e := strconv.ParseUint(values[6], 10, 64)
	if e != nil {
		return nil, e
	}
	httpLog := &HttpLog{
		RemoteHost:    values[0],
		Rfc931:        values[1],
		AuthUser:      values[2],
		Timestamp:     timestamp,
		Method:        requestLine[0],
		Path:          requestLine[1],
		Version:       requestLine[2],
		Status:        uint16(status),
		ResponseBytes: responseBytes,
	}
	if httpLog.Path[0] != '/' {
		return nil, errors.New(fmt.Sprintf("Invalid Path %v", httpLog.Path))
	}
	return httpLog, nil
}

func (hl *HttpLog) GetSection() string {
	// The Path is validated to start with /, which results in at least 2 tokens when splitting
	return strings.SplitN(hl.Path, "/", 3)[1]
}
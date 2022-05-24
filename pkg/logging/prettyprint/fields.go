package prettyprint

import (
	"encoding/json"
	"reflect"
	"time"
)

var logFieldNames []string

func init() {
	// setup logFieldNames to be able to remove them from Extra when unmarshalling
	cliT := reflect.TypeOf(LogFields{})
	numFields := cliT.NumField()
	for i := 0; i < numFields; i++ {
		fldT := cliT.Field(i)
		logFieldNames = append(logFieldNames, fldT.Tag.Get("json"))
	}
}

type LogFields struct {
	Level     string    `json:"level"`
	Module    string    `json:"module"`
	Caller    string    `json:"caller"`
	Timestamp time.Time `json:"time"`
	Message   string    `json:"message"`
}

type LogItem struct {
	LogFields
	Extra map[string]interface{}
}

func (l *LogItem) UnmarshalJSON(bytes []byte) error {
	var logFields LogFields

	if err := json.Unmarshal(bytes, &logFields); err != nil {
		return err
	}
	extra := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &extra); err != nil {
		return err
	}
	for key := range extra {
		for _, logFldName := range logFieldNames {
			if key == logFldName {
				delete(extra, key)
			}
		}
	}

	l.LogFields = logFields
	l.Extra = extra
	return nil
}

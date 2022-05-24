package model

import (
	"github.com/rs/zerolog"
)

type LogRequest struct {
	Body   interface{}
	RespCh chan *LogRequestResponse
}

type GetLogRequestBody struct {
	OffsetFromEnd int
	LineCount     int
	LogLvl        zerolog.Level
}

type OpenLogRequestBody struct {
	FilePath string
}

type LogRequestResponse struct {
	Body     []byte
	NewLines int
	Err      error
}

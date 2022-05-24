package main

import (
	"time"
)

type params struct {
	// TODO this!
	// cli.BuildVersionFlag
	LogCount      uint          `flag:"n|Number of lines that should be generated||required"`
	OutputPath    string        `flag:"o|Path to the output file|./output.log||required"`
	SleepDuration time.Duration `flag:"i|Sleep interval between the individual logs"`
}

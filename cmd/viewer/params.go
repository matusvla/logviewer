package main

type params struct {
	// cli.BuildVersionFlag
	LogLevel string `flag:"loglevel|path to a log file of the viewer - for debugging purposes|"`
	LogPath  string `flag:"logpath|path to log file|./viewer.log"` // todo this is probably not needed at startup
}

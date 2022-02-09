package main

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	// VersionString represents version string.kingpin.
	VersionString = "0.1.0"
)

var (
	argBind  = kingpin.Flag("bind", "Server bind address.").Default("0.0.0.0").String()
	argPort  = kingpin.Flag("port", "Server port.").Default("8080").Int()
	argDebug = kingpin.Flag("debug", "Debug mode.").Default("false").Bool()
)

func argparse() {
	kingpin.Version(VersionString)
	kingpin.Parse()
}

func setLogger(debug bool) {
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	format := new(logrus.TextFormatter)
	format.TimestampFormat = "01-02T15:04:05.000000Z0700"
	format.FullTimestamp = true
	logrus.SetFormatter(format)

	logrus.Printf("Version:   %s", VersionString)
}

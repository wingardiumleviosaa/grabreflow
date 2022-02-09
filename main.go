package main

import (
	app "grabreflow/pkg/app/instance"

	"github.com/sirupsen/logrus"
)

func main() {
	argparse()
	setLogger(*argDebug)

	a := app.NewInstance(*argBind, *argPort)

	if err := a.Init(); err != nil {
		logrus.Fatal("init: %v", err)
	}

	if err := a.Run(); err != nil {
		logrus.Fatal("run: %v", err)
	}
}

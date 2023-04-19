package spec

import "github.com/sirupsen/logrus"

var (
	log = logrus.WithField(
		"module", "spec",
	)
)

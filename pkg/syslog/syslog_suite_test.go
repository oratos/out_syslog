package syslog_test

import (
	"io/ioutil"
	"log"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
)

func TestSyslog(t *testing.T) {
	RegisterFailHandler(Fail)
	log.SetOutput(ioutil.Discard)
	format.TruncatedDiff = false
	RunSpecs(t, "Syslog Suite")
}

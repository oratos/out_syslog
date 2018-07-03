package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cmd Suite")
}

var (
	pluginPath string
	cleanup    func()
)

var _ = BeforeSuite(func() {
	pluginPath, cleanup = buildPlugin()
})

var _ = AfterSuite(func() {
	cleanup()
})

func buildPlugin() (string, func()) {
	path, err := gexec.Build(
		"github.com/pivotal-cf/fluent-bit-out-syslog/cmd",
		"-buildmode", "c-shared",
	)
	Expect(err).ToNot(HaveOccurred())
	return path, gexec.CleanupBuildArtifacts
}

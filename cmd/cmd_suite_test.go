package main_test

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/oratos/out_syslog/pkg/fluentbin"
)

func TestCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cmd Suite")
}

var (
	fbPath     string
	pluginPath string
	cleanups   []func()
)

var _ = BeforeSuite(func() {
	var cleanup func()
	fbPath, cleanup = writeBin(fluentbin.MustAsset(binName))
	cleanups = append(cleanups, cleanup)

	pluginPath, cleanup = buildPlugin()
	cleanups = append(cleanups, cleanup)
})

var _ = AfterSuite(func() {
	for _, c := range cleanups {
		c()
	}
})

func writeBin(bin []byte) (string, func()) {
	f, err := ioutil.TempFile("", "")
	Expect(err).ToNot(HaveOccurred())
	defer f.Close()

	n, err := f.Write(bin)
	Expect(err).ToNot(HaveOccurred())
	if n != len(bin) {
		Fail("unable to write bin to temp file")
	}

	if err := os.Chmod(f.Name(), 0777); err != nil {
        Fail("Fail to change file mode")
    }

	return f.Name(), func() {
		err := os.Remove(f.Name())
		Expect(err).ToNot(HaveOccurred())
	}
}

func buildPlugin() (string, func()) {
	path, err := gexec.Build(
		"github.com/oratos/out_syslog/cmd",
		"-buildmode", "c-shared",
	)
	Expect(err).ToNot(HaveOccurred())
	return path, func() {
		gexec.CleanupBuildArtifacts()
	}
}

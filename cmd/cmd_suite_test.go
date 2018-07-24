package main_test

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	cleanup    = func() {}
)

var _ = BeforeSuite(func() {
	detectDocker()
	pluginPath, cleanup = buildPlugin()
})

var _ = AfterSuite(func() {
	cleanup()
})

func detectDocker() {
	cmd := exec.Command("which", "docker")
	sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	sess.Wait()
	if sess.ExitCode() != 0 {
		Fail("docker is required to be installed and running on your machine" +
			" in order to build the plugin and run it with fluent bit")
	}
}

func buildPlugin() (string, func()) {
	tmpPath, err := ioutil.TempDir("/tmp", "")
	Expect(err).ToNot(HaveOccurred())

	gopath, err := goPath()
	Expect(err).ToNot(HaveOccurred())
	cmd := exec.Command(
		"docker",
		"run",
		"--volume", gopath+":/go",
		"--volume", tmpPath+":/output",
		"golang:latest",
		"go",
		"build",
		"-buildmode", "c-shared",
		"-o", "/output/out_syslog.so",
		"github.com/pivotal-cf/fluent-bit-out-syslog/cmd",
	)
	sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	// The code build takes no more than 1 minute, but docker pull might take time
	sess.Wait(5 * time.Minute)
	Eventually(sess).Should(gexec.Exit(0))

	return path.Join(tmpPath, "out_syslog.so"), func() {
		err := os.RemoveAll(tmpPath)
		Expect(err).ToNot(HaveOccurred())
	}
}

func goPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	absWd, err := filepath.Abs(wd)
	if err != nil {
		return "", err
	}
	for _, path := range strings.Split(os.Getenv("GOPATH"), ":") {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if strings.Contains(absWd, absPath) {
			return absPath, nil
		}
	}
	return "", errors.New("unable to find go path dir")
}

package integration_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("V3 Wrapped CF NodeJS Buildpack", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Describe("nodeJS versions", func() {
		Context("when specifying a range for the nodeJS version in the package.json", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "integration", "testdata", "simple_app"))
			})

			It("resolves to a nodeJS version successfully", func() {
				Expect(app.Push()).To(Succeed())
				Eventually(func() ([]string, error) { return app.InstanceStates() }, 120*time.Second).Should(Equal([]string{"RUNNING"}))

				Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp(`NodeJS \d+\.\d+\.\d+: Contributing`))
				Eventually(app.Stdout.ANSIStrippedString).Should(ContainSubstring("Installing node_modules"))
				Expect(app.GetBody("/")).To(ContainSubstring("Hello World!"))
			})
		})

		Context("Unbuilt buildpack (eg github)", func() {
			var bpName string

			BeforeEach(func() {
				if cutlass.Cached {
					Skip("skipping cached buildpack test")
				}

				tmpDir, err := ioutil.TempDir("", "")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(tmpDir)

				bpName = "unbuilt-v3-node"
				bpZip := filepath.Join(tmpDir, bpName+".zip")

				app = cutlass.New(filepath.Join(bpDir, "integration", "testdata", "simple_app"))
				app.Buildpacks = []string{bpName + "_buildpack"}

				cmd := exec.Command("git", "archive", "-o", bpZip, "HEAD")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Dir = bpDir
				Expect(cmd.Run()).To(Succeed())

				Expect(cutlass.CreateOrUpdateBuildpack(bpName, bpZip, "")).To(Succeed())
			})

			AfterEach(func() {
				Expect(cutlass.DeleteBuildpack(bpName)).To(Succeed())
			})

			It("runs", func() {
				Expect(app.Push()).To(Succeed())
				Eventually(func() ([]string, error) { return app.InstanceStates() }, 120*time.Second).Should(Equal([]string{"RUNNING"}))

				Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp(`NodeJS \d+\.\d+\.\d+: Contributing`))
				Expect(app.GetBody("/")).To(ContainSubstring("Hello World!"))
			})
		})
	})

	Context("multiple buildpacks with v2 and v3 buildpacks", func() {
		BeforeEach(func() {
			if ok, err := cutlass.ApiGreaterThan("2.65.1"); err != nil || !ok {
				Skip("API version does not have multi-buildpack support")
			}

			app = cutlass.New(filepath.Join(bpDir, "integration", "testdata", "v2_supplies_dotnet"))
			app.Disk = "1G"
			app.Memory = "1G"
		})

		It("makes the supplied v2 dependency available at v3 launch and build", func() {
			app.Buildpacks = []string{
				"https://github.com/cloudfoundry/dotnet-core-buildpack#master",
				"nodejs_buildpack",
			}
			Expect(app.Push()).To(Succeed())

			Expect(app.Stdout.String()).To(ContainSubstring("Supplying Dotnet Core"))
			Expect(app.GetBody("/")).To(MatchRegexp(`dotnet: \d+\.\d+\.\d+`))
			Expect(app.GetBody("/text")).To(MatchRegexp(`Text: \d+\.\d+\.\d+`))
		})

		It("throws an error when a v3 buildpack is followed by a v2 buildpack", func() {
			app.StartCommand = "npm start"
			app.Buildpacks = []string{
				"nodejs_buildpack",
				"https://github.com/cloudfoundry/binary-buildpack#develop",
			}
			Expect(app.Push()).NotTo(Succeed())
			Expect(app.Stdout.String()).To(ContainSubstring("ERROR: You are running a V2 buildpack after a V3 buildpack. This is unsupported."))
			Expect(app.Stdout.String()).NotTo(ContainSubstring("ERR bash: npm: command not found"))
		})

		It("throws an error when a v3 buildpack is followed by an older v2 buildpack that does not respect the sentinel file", func() {
			app.StartCommand = "npm start"
			app.Buildpacks = []string{
				"nodejs_buildpack",
				"https://github.com/cloudfoundry/binary-buildpack#v1.0.13",
			}
			Expect(app.Push()).NotTo(Succeed())
			Expect(app.Stdout.String()).NotTo(ContainSubstring("ERR bash: npm: command not found"))
		})

		// This test won't run the latest local code as it runs against a remote branch
		Context("when using github urls", func() {
			It("makes the supplied v2 dependency available at v3 launch and build", func() {
				app.Buildpacks = []string{
					"https://github.com/cloudfoundry/dotnet-core-buildpack#master",
					"https://github.com/cloudfoundry/nodejs-buildpack#v3",
				}
				Expect(app.Push()).To(Succeed())

				Expect(app.Stdout.String()).To(ContainSubstring("Supplying Dotnet Core"))
				Expect(app.GetBody("/")).To(MatchRegexp(`dotnet: \d+\.\d+\.\d+`))
				Expect(app.GetBody("/text")).To(MatchRegexp(`Text: \d+\.\d+\.\d+`))
			})
		})
	})
})

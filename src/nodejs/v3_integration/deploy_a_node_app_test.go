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
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "brats"))
			})

			It("resolves to a nodeJS version successfully", func() {
				Expect(app.Push()).To(Succeed())
				Eventually(func() ([]string, error) { return app.InstanceStates() }, 120*time.Second).Should(Equal([]string{"RUNNING"}))

				Eventually(app.Stdout.String).Should(MatchRegexp(`.*NodeJS.*8\.\d+\.\d+.*:.*Contributing.* to launch`))
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
				os.RemoveAll(tmpDir)

				bpName = "unbuilt-v3-node"
				bpZip := filepath.Join(tmpDir, bpName+".zip")

				app = cutlass.New(filepath.Join(bpDir, "fixtures", "brats"))
				app.Buildpacks = []string{bpName + "_buildpack"}

				cmd := exec.Command("git", "archive", "-o", bpZip, "HEAD")
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

				Eventually(app.Stdout.String).Should(MatchRegexp(`.*NodeJS.*8\.\d+\.\d+.*:.*Contributing.* to launch`))
				Expect(app.GetBody("/")).To(ContainSubstring("Hello World!"))
			})
		})
	})
})

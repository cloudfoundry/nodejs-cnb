package integration_test

import (
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
			var (
				bpName string
				app    *cutlass.App
			)
			BeforeEach(func() {
				bpName = "unbuilt-v3-node"
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "brats"))
				app.Buildpacks = []string{bpName + "_buildpack"}
				cmd := exec.Command("git", "archive", "-o", filepath.Join("/tmp", bpName+".zip"), "HEAD")
				cmd.Dir = bpDir
				Expect(cmd.Run()).To(Succeed())
				Expect(cutlass.CreateOrUpdateBuildpack(bpName, filepath.Join("/tmp", bpName+".zip"), "")).To(Succeed())
				Expect(os.Remove(filepath.Join("/tmp", bpName+".zip"))).To(Succeed())
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

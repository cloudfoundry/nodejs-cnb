package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("pushing an app a second time", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	BeforeEach(func() {
		if cutlass.Cached {
			Skip("running uncached tests")
		}

		app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "simple_app"))
		app.Buildpacks = []string{"nodejs_buildpack"}
	})

	downloadRegexp := `Downloading from .*/node\-[\d\.]+\-linux\-x64\-(cflinuxfs.*-)?[\da-f]+\.tgz`
	copyRegexp := "Node Engine .*: Reusing cached layer"

	It("uses the cache for manifest dependencies", func() {
		PushAppAndConfirm(app)
		Expect(app.Stdout.ANSIStrippedString()).To(MatchRegexp(downloadRegexp))
		Expect(app.Stdout.ANSIStrippedString()).ToNot(MatchRegexp(copyRegexp))

		app.Stdout.Reset()
		PushAppAndConfirm(app)
		Expect(app.Stdout.ANSIStrippedString()).ToNot(MatchRegexp(downloadRegexp))
		Expect(app.Stdout.ANSIStrippedString()).To(MatchRegexp(copyRegexp))
	})
})

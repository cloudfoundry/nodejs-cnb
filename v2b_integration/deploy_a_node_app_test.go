package integration_test

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF NodeJS Buildpack", func() {
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
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "node_version_range"))
			})

			It("resolves to a nodeJS version successfully", func() {
				PushAppAndConfirm(app)

				Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp(`Node Engine \d+\.\d+\.\d+: Contributing to layer`))
				Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))

				if ApiHasTask() {
					By("running a task", func() {
						By("can find node in the container", func() {
							command := exec.Command("cf", "run-task", app.Name, "echo \"RUNNING A TASK: $(node --version)\"")
							_, err := command.Output()
							Expect(err).To(BeNil())

							Eventually(func() string {
								return app.Stdout.ANSIStrippedString()
							}, "30s").Should(MatchRegexp("RUNNING A TASK: v\\d+\\.\\d+\\.\\d+"))
						})
					})
				}
			})
		})

		Context("when not specifying a nodeJS version in the package.json", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "without_node_version"))
			})

			It("resolves to the stable nodeJS version successfully", func() {
				PushAppAndConfirm(app)
				defaultNode := "10"
				Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp(fmt.Sprintf(`Node Engine %s\.\d+\.\d+: Contributing to layer`, defaultNode)))
				Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
			})
		})

		Context("with an unreleased nodejs version", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "unreleased_node_version"))
			})

			It("displays a nice error message and gracefully fails", func() {
				Expect(app.Push()).ToNot(BeNil())

				Eventually(app.Stdout.ANSIStrippedString, 2*time.Second).Should(ContainSubstring("no valid dependencies for node, 9000.0.0"))
				Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())
			})
		})

		Context("with an unsupported, but released, nodejs version", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "unsupported_node_version"))
			})

			It("displays a nice error messages and gracefully fails", func() {
				Expect(app.Push()).ToNot(BeNil())

				Eventually(app.Stdout.ANSIStrippedString, 2*time.Second).Should(ContainSubstring("no valid dependencies for node, 4.1.1"))
				Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())
			})
		})
	})

	Context("with no Procfile and OPTIMIZE_MEMORY=true", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "simple_app"))
			app.SetEnv("OPTIMIZE_MEMORY", "true")
		})

		It("is running with autosized max_old_space_size", func() {
			PushAppAndConfirm(app)

			Expect(app.GetBody("/")).To(ContainSubstring("NodeOptions: --max_old_space_size=96"))
		})
	})

	Context("with no Procfile and OPTIMIZE_MEMORY is unset", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "simple_app"))
		})

		It("is not running with autosized max_old_space_size", func() {
			PushAppAndConfirm(app)

			Expect(app.GetBody("/")).To(ContainSubstring("NodeOptions: undefined"))
		})

		Context("a nvmrc file that takes precedence over package.json", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "simple_app_with_nvmrc"))
			})

			It("deploys", func() {
				PushAppAndConfirm(app)

				Expect(app.GetBody("/")).To(ContainSubstring("NodeOptions: undefined"))
			})
		})
	})

	Describe("Vendored Dependencies", func() {
		Context("with an app that has vendored dependencies", func() {
			It("deploys", func() {
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "vendored_dependencies"))
				PushAppAndConfirm(app)
				Expect(app.Stdout.ANSIStrippedString()).To(ContainSubstring("Rebuilding node_modules"))

				By("does not output protip that recommends user vendors dependencies", func() {
					Expect(app.Stdout.ANSIStrippedString()).ToNot(MatchRegexp("It is recommended to vendor the application's Node.js dependencies"))
				})

				if !cutlass.Cached {
					By("with an uncached buildpack", func() {
						By("successfully deploys and includes the dependencies", func() {
							Expect(app.GetBody("/")).To(ContainSubstring("0000000005"))
							Eventually(app.Stdout.ANSIStrippedString).Should(ContainSubstring("Downloading from https://"))
						})
					})
				}

				if cutlass.Cached {
					By("with a cached buildpack", func() {
						By("deploys without hitting the internet", func() {
							Expect(app.GetBody("/")).To(ContainSubstring("0000000005"))
							Eventually(app.Stdout.ANSIStrippedString).Should(ContainSubstring("Copy [/tmp/buildpacks/"))
						})
					})
				}
			})

			AssertNoInternetTraffic(filepath.Join(bpDir, "v2b_integration", "testdata", "vendored_dependencies"))
		})

		Context("Vendored Depencencies with node module binaries", func() {
			BeforeEach(func() {
				if !ApiSupportsSymlinks() {
					Skip("Requires api symlink support")
				}
			})

			It("deploys", func() {
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "vendored_dependencies_with_binaries"))
				PushAppAndConfirm(app)
			})
		})

		Context("with an app with a yarn.lock and vendored dependencies", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "with_yarn_vendored"))
				if !cutlass.Cached {
					Skip("offline requires vendored dependencies")
				}
			})

			It("deploys without hitting the internet", func() {
				PushAppAndConfirm(app)

				Expect(filepath.Join(app.Path, "node_modules")).To(BeADirectory())
				Eventually(app.Stdout.ANSIStrippedString).Should(ContainSubstring("Running yarn in offline mode"))
				Expect(app.GetBody("/microtime")).To(MatchRegexp("native time: \\d+\\.\\d+"))
			})

			AssertNoInternetTraffic(filepath.Join(bpDir, "v2b_integration", "testdata", "with_yarn_vendored"))
		})
		Context("with an incomplete node_modules directory", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "incomplete_node_modules"))
			})

			It("downloads missing dependencies from package.json", func() {
				PushAppAndConfirm(app)
				Expect(filepath.Join(app.Path, "node_modules")).To(BeADirectory())
				Expect(filepath.Join(app.Path, "node_modules", "hashish")).ToNot(BeADirectory())
			})
		})
		Context("with an incomplete package.json", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "incomplete_package_json"))
			})

			It("does not overwrite the vendored modules not listed in package.json", func() {
				PushAppAndConfirm(app)
				Expect(app.Files(".")).To(ContainElement(ContainSubstring("node_modules/leftpad")))
				Expect(app.Files(".")).NotTo(ContainElement(ContainSubstring("node_modules/hashish")))
				Expect(app.Files(".")).NotTo(ContainElement(ContainSubstring("node_modules/traverse")))
			})
		})
	})

	Describe("No Vendored Dependencies", func() {
		Context("with an app with no vendored dependencies", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "no_vendored_dependencies"))
			})

			It("successfully deploys and vendors the dependencies", func() {
				PushAppAndConfirm(app)

				Expect(filepath.Join(app.Path, "node_modules")).ToNot(BeADirectory())

				Eventually(app.Stdout.ANSIStrippedString).Should(ContainSubstring("Installing node_modules"))
				Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))

				By("outputs protip that recommends user vendors dependencies", func() {
					Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp("It is recommended to vendor the application's Node.js dependencies"))
				})
			})

			AssertUsesProxyDuringStagingIfPresent(filepath.Join(bpDir, "v2b_integration", "testdata", "no_vendored_dependencies"))
		})
		Context("with an app with a yarn.lock file", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "with_yarn"))
			})

			It("successfully deploys and vendors the dependencies via yarn", func() {
				PushAppAndConfirm(app)

				Expect(filepath.Join(app.Path, "node_modules")).ToNot(BeADirectory())

				Eventually(app.Stdout.ANSIStrippedString).Should(ContainSubstring("Running yarn in online mode"))

				Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
			})

			AssertUsesProxyDuringStagingIfPresent(filepath.Join(bpDir, "v2b_integration", "testdata", "with_yarn"))
		})
		Context("with an app with an out of date yarn.lock", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "out_of_date_yarn_lock"))
			})

			It("warns that yarn.lock is out of date", func() {
				PushAppAndConfirm(app)
				Eventually(app.Stdout.ANSIStrippedString).Should(ContainSubstring("yarn.lock is outdated"))
			})
		})
		Context("with an app with pre and post scripts", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "pre_post_commands"))
			})

			It("runs the scripts through npm run", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("Text: Hello Buildpacks Team"))
				Expect(app.GetBody("/")).To(ContainSubstring("Text: Goodbye Buildpacks Team"))
			})

			It("runs the postinstall script in the app directory", func() {
				PushAppAndConfirm(app)
				Eventually(app.Stdout.ANSIStrippedString, 2*time.Second).Should(ContainSubstring("Current dir: /home/vcap/app")) ///home/vcap/app is the v3 app dir
			})
		})
	})

	Describe("NODE_HOME and NODE_ENV", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "logenv"))
		})

		It("sets the NODE_HOME to correct value", func() {
			PushAppAndConfirm(app)
			Eventually(app.Stdout.ANSIStrippedString).Should(ContainSubstring("Writing NODE_HOME"))

			body, err := app.GetBody("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(MatchRegexp(`"NODE_HOME":"[^"]*/node"`))
			Expect(body).To(ContainSubstring(`"NODE_ENV":"production"`))
			Expect(body).To(ContainSubstring(`"MEMORY_AVAILABLE":"128"`))
		})
	})

	Describe(".profile script", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "with_profile_script"))
		})

		It("runs .profile script when staging", func() {
			PushAppAndConfirm(app)
			Eventually(app.Stdout.ANSIStrippedString).Should(ContainSubstring("Writing NODE_HOME"))

			_, err := app.GetBody("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(app.Stdout.String()).To(ContainSubstring("PROFILE_SCRIPT_IS_PRESENT_AND_RAN"))

			_, headers, err := app.Get("/.profile", map[string]string{})
			Expect(err).ToNot(HaveOccurred())
			Expect(headers).To(HaveKeyWithValue("StatusCode", []string{"404"}))

		})
	})

	Describe("when setting env vars on the app", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "simple_app"))
			app.SetEnv("APP_ENV_VAR", "SUPER SECRET SECRET")
			PushAppAndConfirm(app)
		})

		It("Does not save app env vars into the droplet", func() {
			Expect(app.DownloadDroplet(filepath.Join(app.Path, "droplet.tgz"))).To(Succeed())
			dropletPath := filepath.Join(app.Path, "droplet.tgz")
			file, err := os.Open(dropletPath)
			defer os.RemoveAll(dropletPath)
			Expect(err).ToNot(HaveOccurred())
			defer file.Close()
			gz, err := gzip.NewReader(file)
			Expect(err).ToNot(HaveOccurred())
			defer gz.Close()
			tr := tar.NewReader(gz)

			for {
				hdr, err := tr.Next()
				if err == io.EOF {
					break
				}
				b, err := ioutil.ReadAll(tr)
				for _, content := range []string{"MY_SPECIAL_VAR", "SUPER SENSITIVE DATA"} {
					if strings.Contains(string(b), content) {
						Fail(fmt.Sprintf("Found sensitive string %s in %s", content, hdr.Name))
					}
				}
			}
		})
	})

	Describe("Unbuilt buildpack (eg github)", func() {
		// only run if using uncached buildpack
		var bpName string
		BeforeEach(func() {
			if cutlass.Cached {
				Skip("unbuilt requires uncached buildpack")
			}
			app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "simple_app"))
			bpName = fmt.Sprintf("unbuilt-nodejs-%s", cutlass.RandStringRunes(8))
			app.Buildpacks = []string{bpName + "_buildpack"}
			//cmd := exec.Command("git", "archive", "-o", filepath.Join("/tmp", bpName+".zip"), "HEAD")
			stashCmd := exec.Command("git", "stash", "create")
			out, err := stashCmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())

			cmd := exec.Command("git", "archive", "-o", filepath.Join("/tmp", bpName+".zip"), strings.Trim(string(out), "\n"))
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

			Expect(app.Stdout.String()).To(ContainSubstring("Installing node"))
			Expect(app.GetBody("/")).To(ContainSubstring("NodeOptions: "))
		})
	})

	Describe("System CA Store", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "v2b_integration", "testdata", "use-openssl-ca"))
			app.SetEnv("SSL_CERT_FILE", "cert.pem")
		})
		It("uses the system CA store (or env)", func() {
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("Response over self signed https"))
		})
	})
})

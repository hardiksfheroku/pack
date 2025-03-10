package writer_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/platform"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/inspectimage"
	"github.com/buildpacks/pack/internal/inspectimage/writer"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestHumanReadable(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Human Readable Writer", testHumanReadable, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testHumanReadable(t *testing.T, when spec.G, it spec.S) {
	var (
		assert = h.NewAssertionManager(t)
		outBuf bytes.Buffer

		remoteInfo *client.ImageInfo
		localInfo  *client.ImageInfo

		expectedRemoteOutput = `REMOTE:

Stack: test.stack.id.remote

Base Image:
  Reference: some-remote-run-image-reference
  Top Layer: some-remote-top-layer

Run Images:
  user-configured-mirror-for-remote        (user-configured)
  some-remote-run-image
  some-remote-mirror
  other-remote-mirror

Buildpacks:
  ID                          VERSION        HOMEPAGE
  test.bp.one.remote          1.0.0          https://some-homepage-one
  test.bp.two.remote          2.0.0          https://some-homepage-two
  test.bp.three.remote        3.0.0          -

Processes:
  TYPE                              SHELL        COMMAND                      ARGS                     WORK DIR
  some-remote-type (default)        bash         /some/remote command         some remote args         /some-test-work-dir
  other-remote-type                              /other/remote/command        other remote args        /other-test-work-dir`

		expectedLocalOutput = `LOCAL:

Stack: test.stack.id.local

Base Image:
  Reference: some-local-run-image-reference
  Top Layer: some-local-top-layer

Run Images:
  user-configured-mirror-for-local        (user-configured)
  some-local-run-image
  some-local-mirror
  other-local-mirror

Buildpacks:
  ID                         VERSION        HOMEPAGE
  test.bp.one.local          1.0.0          https://some-homepage-one
  test.bp.two.local          2.0.0          https://some-homepage-two
  test.bp.three.local        3.0.0          -

Processes:
  TYPE                             SHELL        COMMAND                     ARGS                    WORK DIR
  some-local-type (default)        bash         /some/local command         some local args         /some-test-work-dir
  other-local-type                              /other/local/command        other local args        /other-test-work-dir`
	)

	when("Print", func() {
		it.Before(func() {
			type someData struct {
				String string
				Bool   bool
				Int    int
				Nested struct {
					String string
				}
			}

			remoteInfo = &client.ImageInfo{
				StackID: "test.stack.id.remote",
				Buildpacks: []buildpack.GroupElement{
					{ID: "test.bp.one.remote", Version: "1.0.0", Homepage: "https://some-homepage-one"},
					{ID: "test.bp.two.remote", Version: "2.0.0", Homepage: "https://some-homepage-two"},
					{ID: "test.bp.three.remote", Version: "3.0.0"},
				},
				Base: platform.RunImageMetadata{
					TopLayer:  "some-remote-top-layer",
					Reference: "some-remote-run-image-reference",
				},
				Stack: platform.StackMetadata{
					RunImage: platform.StackRunImageMetadata{
						Image:   "some-remote-run-image",
						Mirrors: []string{"some-remote-mirror", "other-remote-mirror"},
					},
				},
				BOM: []buildpack.BOMEntry{{
					Require: buildpack.Require{
						Name:    "name-1",
						Version: "version-1",
						Metadata: map[string]interface{}{
							"RemoteData": someData{
								String: "aString",
								Bool:   true,
								Int:    123,
								Nested: struct {
									String string
								}{
									String: "anotherString",
								},
							},
						},
					},
					Buildpack: buildpack.GroupElement{ID: "test.bp.one.remote", Version: "1.0.0"},
				}},
				Processes: client.ProcessDetails{
					DefaultProcess: &launch.Process{
						Type:             "some-remote-type",
						Command:          launch.RawCommand{Entries: []string{"/some/remote command"}},
						Args:             []string{"some", "remote", "args"},
						Direct:           false,
						WorkingDirectory: "/some-test-work-dir",
					},
					OtherProcesses: []launch.Process{
						{
							Type:             "other-remote-type",
							Command:          launch.RawCommand{Entries: []string{"/other/remote/command"}},
							Args:             []string{"other", "remote", "args"},
							Direct:           true,
							WorkingDirectory: "/other-test-work-dir",
						},
					},
				},
			}

			localInfo = &client.ImageInfo{
				StackID: "test.stack.id.local",
				Buildpacks: []buildpack.GroupElement{
					{ID: "test.bp.one.local", Version: "1.0.0", Homepage: "https://some-homepage-one"},
					{ID: "test.bp.two.local", Version: "2.0.0", Homepage: "https://some-homepage-two"},
					{ID: "test.bp.three.local", Version: "3.0.0"},
				},
				Base: platform.RunImageMetadata{
					TopLayer:  "some-local-top-layer",
					Reference: "some-local-run-image-reference",
				},
				Stack: platform.StackMetadata{
					RunImage: platform.StackRunImageMetadata{
						Image:   "some-local-run-image",
						Mirrors: []string{"some-local-mirror", "other-local-mirror"},
					},
				},
				BOM: []buildpack.BOMEntry{{
					Require: buildpack.Require{
						Name:    "name-1",
						Version: "version-1",
						Metadata: map[string]interface{}{
							"LocalData": someData{
								Bool: false,
								Int:  456,
							},
						},
					},
					Buildpack: buildpack.GroupElement{ID: "test.bp.one.remote", Version: "1.0.0"},
				}},
				Processes: client.ProcessDetails{
					DefaultProcess: &launch.Process{
						Type:             "some-local-type",
						Command:          launch.RawCommand{Entries: []string{"/some/local command"}},
						Args:             []string{"some", "local", "args"},
						Direct:           false,
						WorkingDirectory: "/some-test-work-dir",
					},
					OtherProcesses: []launch.Process{
						{
							Type:             "other-local-type",
							Command:          launch.RawCommand{Entries: []string{"/other/local/command"}},
							Args:             []string{"other", "local", "args"},
							Direct:           true,
							WorkingDirectory: "/other-test-work-dir",
						},
					},
				},
			}

			outBuf = bytes.Buffer{}
		})

		when("local and remote image exits", func() {
			it("prints both local and remote image info in a human readable format", func() {
				runImageMirrors := []config.RunImage{
					{
						Image:   "un-used-run-image",
						Mirrors: []string{"un-used"},
					},
					{
						Image:   "some-local-run-image",
						Mirrors: []string{"user-configured-mirror-for-local"},
					},
					{
						Image:   "some-remote-run-image",
						Mirrors: []string{"user-configured-mirror-for-remote"},
					},
				}
				sharedImageInfo := inspectimage.GeneralInfo{
					Name:            "test-image",
					RunImageMirrors: runImageMirrors,
				}
				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, sharedImageInfo, localInfo, remoteInfo, nil, nil)
				assert.Nil(err)

				assert.Contains(outBuf.String(), expectedLocalOutput)
				assert.Contains(outBuf.String(), expectedRemoteOutput)
			})
		})

		when("only local image exists", func() {
			it("prints local image info in a human readable format", func() {
				runImageMirrors := []config.RunImage{
					{
						Image:   "un-used-run-image",
						Mirrors: []string{"un-used"},
					},
					{
						Image:   "some-local-run-image",
						Mirrors: []string{"user-configured-mirror-for-local"},
					},
					{
						Image:   "some-remote-run-image",
						Mirrors: []string{"user-configured-mirror-for-remote"},
					},
				}
				sharedImageInfo := inspectimage.GeneralInfo{
					Name:            "test-image",
					RunImageMirrors: runImageMirrors,
				}
				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, sharedImageInfo, localInfo, nil, nil, nil)
				assert.Nil(err)

				assert.Contains(outBuf.String(), expectedLocalOutput)
				assert.NotContains(outBuf.String(), expectedRemoteOutput)
			})
		})

		when("only remote image exists", func() {
			it("prints remote image info in a human readable format", func() {
				runImageMirrors := []config.RunImage{
					{
						Image:   "un-used-run-image",
						Mirrors: []string{"un-used"},
					},
					{
						Image:   "some-local-run-image",
						Mirrors: []string{"user-configured-mirror-for-local"},
					},
					{
						Image:   "some-remote-run-image",
						Mirrors: []string{"user-configured-mirror-for-remote"},
					},
				}
				sharedImageInfo := inspectimage.GeneralInfo{
					Name:            "test-image",
					RunImageMirrors: runImageMirrors,
				}
				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, sharedImageInfo, nil, remoteInfo, nil, nil)
				assert.Nil(err)

				assert.NotContains(outBuf.String(), expectedLocalOutput)
				assert.Contains(outBuf.String(), expectedRemoteOutput)
			})

			when("buildpack metadata is missing", func() {
				it.Before(func() {
					remoteInfo.Buildpacks = []buildpack.GroupElement{}
				})
				it("displays a message indicating missing metadata", func() {
					sharedImageInfo := inspectimage.GeneralInfo{
						Name:            "test-image",
						RunImageMirrors: []config.RunImage{},
					}

					humanReadableWriter := writer.NewHumanReadable()

					logger := logging.NewLogWithWriters(&outBuf, &outBuf)
					err := humanReadableWriter.Print(logger, sharedImageInfo, nil, remoteInfo, nil, nil)
					assert.Nil(err)

					assert.Contains(outBuf.String(), "(buildpack metadata not present)")
				})
			})

			when("there are no run images", func() {
				it.Before(func() {
					remoteInfo.Stack = platform.StackMetadata{}
				})
				it("displays a message indicating missing run images", func() {
					sharedImageInfo := inspectimage.GeneralInfo{
						Name:            "test-image",
						RunImageMirrors: []config.RunImage{},
					}

					humanReadableWriter := writer.NewHumanReadable()

					logger := logging.NewLogWithWriters(&outBuf, &outBuf)
					err := humanReadableWriter.Print(logger, sharedImageInfo, nil, remoteInfo, nil, nil)
					assert.Nil(err)

					assert.Contains(outBuf.String(), "Run Images:\n  (none)")
				})
			})
		})

		when("error handled cases", func() {
			when("there is a remoteErr", func() {
				var remoteErr error
				it.Before(func() {
					remoteErr = errors.New("some remote error")
				})
				it("displays the remote error and local info", func() {
					runImageMirrors := []config.RunImage{
						{
							Image:   "un-used-run-image",
							Mirrors: []string{"un-used"},
						},
						{
							Image:   "some-local-run-image",
							Mirrors: []string{"user-configured-mirror-for-local"},
						},
						{
							Image:   "some-remote-run-image",
							Mirrors: []string{"user-configured-mirror-for-remote"},
						},
					}
					sharedImageInfo := inspectimage.GeneralInfo{
						Name:            "test-image",
						RunImageMirrors: runImageMirrors,
					}
					humanReadableWriter := writer.NewHumanReadable()

					logger := logging.NewLogWithWriters(&outBuf, &outBuf)
					err := humanReadableWriter.Print(logger, sharedImageInfo, localInfo, remoteInfo, nil, remoteErr)
					assert.Nil(err)

					assert.Contains(outBuf.String(), expectedLocalOutput)
					assert.Contains(outBuf.String(), "some remote error")
				})
			})

			when("there is a localErr", func() {
				var localErr error
				it.Before(func() {
					localErr = errors.New("some local error")
				})
				it("displays the remote info and local error", func() {
					runImageMirrors := []config.RunImage{
						{
							Image:   "un-used-run-image",
							Mirrors: []string{"un-used"},
						},
						{
							Image:   "some-local-run-image",
							Mirrors: []string{"user-configured-mirror-for-local"},
						},
						{
							Image:   "some-remote-run-image",
							Mirrors: []string{"user-configured-mirror-for-remote"},
						},
					}
					sharedImageInfo := inspectimage.GeneralInfo{
						Name:            "test-image",
						RunImageMirrors: runImageMirrors,
					}
					humanReadableWriter := writer.NewHumanReadable()

					logger := logging.NewLogWithWriters(&outBuf, &outBuf)
					err := humanReadableWriter.Print(logger, sharedImageInfo, localInfo, remoteInfo, localErr, nil)
					assert.Nil(err)

					assert.Contains(outBuf.String(), expectedRemoteOutput)
					assert.Contains(outBuf.String(), "some local error")
				})
			})

			when("error cases", func() {
				when("both localInfo and remoteInfo are nil", func() {
					it("displays a 'missing image' error message", func() {
						humanReadableWriter := writer.NewHumanReadable()

						logger := logging.NewLogWithWriters(&outBuf, &outBuf)
						err := humanReadableWriter.Print(logger, inspectimage.GeneralInfo{Name: "missing-image"}, nil, nil, nil, nil)
						assert.ErrorWithMessage(err, fmt.Sprintf("unable to find image '%s' locally or remotely", "missing-image"))
					})
				})
			})
		})
	})
}

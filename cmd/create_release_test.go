package cmd_test

import (
	"errors"

	semver "github.com/cppforlife/go-semi-semantic/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-init/cmd"
	boshrel "github.com/cloudfoundry/bosh-init/release"
	fakerel "github.com/cloudfoundry/bosh-init/release/fakes"
	fakereldir "github.com/cloudfoundry/bosh-init/releasedir/fakes"
	fakeui "github.com/cloudfoundry/bosh-init/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-init/ui/table"
)

var _ = Describe("CreateReleaseCmd", func() {
	var (
		releaseReader *fakerel.FakeReader
		releaseDir    *fakereldir.FakeReleaseDir
		ui            *fakeui.FakeUI
		command       CreateReleaseCmd
	)

	BeforeEach(func() {
		releaseReader = &fakerel.FakeReader{}
		releaseDir = &fakereldir.FakeReleaseDir{}
		ui = &fakeui.FakeUI{}
		command = NewCreateReleaseCmd(releaseReader, releaseDir, ui)
	})

	Describe("Run", func() {
		var (
			opts    CreateReleaseOpts
			release *fakerel.FakeRelease
		)

		BeforeEach(func() {
			opts = CreateReleaseOpts{}

			release = &fakerel.FakeRelease{
				NameStub:               func() string { return "rel" },
				VersionStub:            func() string { return "ver" },
				CommitHashWithMarkStub: func(string) string { return "commit" },

				SetNameStub:    func(name string) { release.NameReturns(name) },
				SetVersionStub: func(ver string) { release.VersionReturns(ver) },
			}
		})

		act := func() error { return command.Run(opts) }

		Context("when manifest path is provided", func() {
			BeforeEach(func() {
				opts.Args.Manifest = FileBytesArg{Path: "/manifest-path"}
			})

			It("builds release and release archive based on manifest path", func() {
				releaseReader.ReadStub = func(path string) (boshrel.Release, error) {
					Expect(path).To(Equal("/manifest-path"))
					return release, nil
				}

				releaseDir.BuildReleaseArchiveStub = func(rel boshrel.Release) (string, error) {
					Expect(rel).To(Equal(release))
					return "/archive-path", nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
					Rows: [][]boshtbl.Value{
						{boshtbl.NewValueString("Name"), boshtbl.NewValueString("rel")},
						{boshtbl.NewValueString("Version"), boshtbl.NewValueString("ver")},
						{boshtbl.NewValueString("Commit Hash"), boshtbl.NewValueString("commit")},
						{boshtbl.NewValueString("Archive"), boshtbl.NewValueString("/archive-path")},
					},
				}))
			})

			It("returns error if reading manifest fails", func() {
				releaseReader.ReadReturns(nil, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if building release archive fails", func() {
				releaseReader.ReadReturns(release, nil)
				releaseDir.BuildReleaseArchiveReturns("", errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})

		Context("when manifest path is not provided", func() {
			It("builds release with default release name and next dev version", func() {
				releaseDir.DefaultNameReturns("default-rel-name", nil)
				releaseDir.NextDevVersionReturns(semver.MustNewVersionFromString("next-dev+ver"), nil)

				releaseDir.BuildReleaseStub = func(name string, version semver.Version, force bool) (boshrel.Release, error) {
					release.SetName(name)
					release.SetVersion(version.String())
					Expect(force).To(BeFalse())
					return release, nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
					Rows: [][]boshtbl.Value{
						{boshtbl.NewValueString("Name"), boshtbl.NewValueString("default-rel-name")},
						{boshtbl.NewValueString("Version"), boshtbl.NewValueString("next-dev+ver")},
						{boshtbl.NewValueString("Commit Hash"), boshtbl.NewValueString("commit")},
					},
				}))
			})

			It("builds release with custom release name and version", func() {
				opts.Name = "custom-name"
				opts.Version = VersionArg(semver.MustNewVersionFromString("custom-ver"))

				releaseDir.DefaultNameReturns("default-rel-name", nil)
				releaseDir.NextDevVersionReturns(semver.MustNewVersionFromString("1.1"), nil)

				releaseDir.BuildReleaseStub = func(name string, version semver.Version, force bool) (boshrel.Release, error) {
					release.SetName(name)
					release.SetVersion(version.String())
					Expect(force).To(BeFalse())
					return release, nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
					Rows: [][]boshtbl.Value{
						{boshtbl.NewValueString("Name"), boshtbl.NewValueString("custom-name")},
						{boshtbl.NewValueString("Version"), boshtbl.NewValueString("custom-ver")},
						{boshtbl.NewValueString("Commit Hash"), boshtbl.NewValueString("commit")},
					},
				}))
			})

			It("builds release forcefully with timestamp version", func() {
				opts.TimestampVersion = true
				opts.Force = true

				releaseDir.DefaultNameReturns("default-rel-name", nil)

				releaseDir.NextDevVersionStub = func(name string, timestamp bool) (semver.Version, error) {
					Expect(name).To(Equal("default-rel-name"))
					Expect(timestamp).To(BeTrue())
					return semver.MustNewVersionFromString("ts-ver"), nil
				}

				releaseDir.BuildReleaseStub = func(name string, version semver.Version, force bool) (boshrel.Release, error) {
					release.SetName(name)
					release.SetVersion(version.String())
					Expect(force).To(BeTrue())
					return release, nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
					Rows: [][]boshtbl.Value{
						{boshtbl.NewValueString("Name"), boshtbl.NewValueString("default-rel-name")},
						{boshtbl.NewValueString("Version"), boshtbl.NewValueString("ts-ver")},
						{boshtbl.NewValueString("Commit Hash"), boshtbl.NewValueString("commit")},
					},
				}))
			})

			It("builds and then finalizes release", func() {
				opts.Final = true

				releaseDir.DefaultNameReturns("default-rel-name", nil)
				releaseDir.NextDevVersionReturns(semver.MustNewVersionFromString("next-dev+ver"), nil)
				releaseDir.NextFinalVersionReturns(semver.MustNewVersionFromString("next-final+ver"), nil)

				releaseDir.BuildReleaseStub = func(name string, version semver.Version, force bool) (boshrel.Release, error) {
					release.SetName(name)
					release.SetVersion(version.String())
					Expect(force).To(BeFalse())
					return release, nil
				}

				releaseDir.FinalizeReleaseStub = func(rel boshrel.Release, force bool) error {
					Expect(rel).To(Equal(release))
					Expect(rel.Name()).To(Equal("default-rel-name"))
					Expect(rel.Version()).To(Equal("next-final+ver"))
					Expect(force).To(BeFalse())
					return nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
					Rows: [][]boshtbl.Value{
						{boshtbl.NewValueString("Name"), boshtbl.NewValueString("default-rel-name")},
						{boshtbl.NewValueString("Version"), boshtbl.NewValueString("next-final+ver")},
						{boshtbl.NewValueString("Commit Hash"), boshtbl.NewValueString("commit")},
					},
				}))
			})

			It("builds and then finalizes release with custom version", func() {
				opts.Final = true
				opts.Version = VersionArg(semver.MustNewVersionFromString("custom-ver"))

				releaseDir.DefaultNameReturns("default-rel-name", nil)
				releaseDir.NextDevVersionReturns(semver.MustNewVersionFromString("1.1"), nil)

				releaseDir.BuildReleaseStub = func(name string, version semver.Version, force bool) (boshrel.Release, error) {
					release.SetName(name)
					release.SetVersion(version.String())
					Expect(force).To(BeFalse())
					return release, nil
				}

				releaseDir.FinalizeReleaseStub = func(rel boshrel.Release, force bool) error {
					Expect(rel).To(Equal(release))
					Expect(rel.Name()).To(Equal("default-rel-name"))
					Expect(rel.Version()).To(Equal("custom-ver"))
					Expect(force).To(BeFalse())
					return nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
					Rows: [][]boshtbl.Value{
						{boshtbl.NewValueString("Name"), boshtbl.NewValueString("default-rel-name")},
						{boshtbl.NewValueString("Version"), boshtbl.NewValueString("custom-ver")},
						{boshtbl.NewValueString("Commit Hash"), boshtbl.NewValueString("commit")},
					},
				}))
			})

			It("builds release and archive if building archive is requested", func() {
				opts.Final = true
				opts.Tarball = true

				releaseDir.DefaultNameReturns("default-rel-name", nil)
				releaseDir.NextDevVersionReturns(semver.MustNewVersionFromString("next-dev+ver"), nil)
				releaseDir.NextFinalVersionReturns(semver.MustNewVersionFromString("next-final+ver"), nil)

				releaseDir.BuildReleaseStub = func(name string, version semver.Version, force bool) (boshrel.Release, error) {
					release.SetName(name)
					release.SetVersion(version.String())
					Expect(force).To(BeFalse())
					return release, nil
				}

				releaseDir.BuildReleaseArchiveStub = func(rel boshrel.Release) (string, error) {
					Expect(rel).To(Equal(release))
					return "/archive-path", nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
					Rows: [][]boshtbl.Value{
						{boshtbl.NewValueString("Name"), boshtbl.NewValueString("default-rel-name")},
						{boshtbl.NewValueString("Version"), boshtbl.NewValueString("next-final+ver")},
						{boshtbl.NewValueString("Commit Hash"), boshtbl.NewValueString("commit")},
						{boshtbl.NewValueString("Archive"), boshtbl.NewValueString("/archive-path")},
					},
				}))
			})

			It("returns error if retrieving default release name fails", func() {
				releaseDir.DefaultNameReturns("", errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if retrieving next dev version fails", func() {
				releaseDir.NextDevVersionReturns(semver.Version{}, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if retrieving next final version fails", func() {
				opts.Final = true

				releaseDir.BuildReleaseReturns(release, nil)
				releaseDir.NextFinalVersionReturns(semver.Version{}, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if building release archive fails", func() {
				opts.Tarball = true

				releaseDir.DefaultNameReturns("default-rel-name", nil)
				releaseDir.NextDevVersionReturns(semver.MustNewVersionFromString("next-dev+ver"), nil)
				releaseDir.BuildReleaseArchiveReturns("", errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})
	})
})

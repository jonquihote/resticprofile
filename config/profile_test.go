package config

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/creativeprojects/resticprofile/constants"
	"github.com/creativeprojects/resticprofile/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoProfile(t *testing.T) {
	testConfig := ""
	profile, err := getProfile("toml", testConfig, "profile", "")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, profile)
}

func TestProfileNotFound(t *testing.T) {
	testConfig := "[profile]\n"
	profile, err := getProfile("toml", testConfig, "other", "")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, profile)
}

func TestEmptyProfile(t *testing.T) {
	testConfig := "[profile]\n"
	profile, err := getProfile("toml", testConfig, "profile", "")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, profile)
	assert.Equal(t, "profile", profile.Name)
}

func TestNoInitializeValue(t *testing.T) {
	testConfig := "[profile]\n"
	profile, err := getProfile("toml", testConfig, "profile", "")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, profile)
	assert.Equal(t, false, profile.Initialize)
}

func TestInitializeValueFalse(t *testing.T) {
	testConfig := `[profile]
initialize = false
`
	profile, err := getProfile("toml", testConfig, "profile", "")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, profile)
	assert.Equal(t, false, profile.Initialize)
}

func TestInitializeValueTrue(t *testing.T) {
	testConfig := `[profile]
initialize = true
`
	profile, err := getProfile("toml", testConfig, "profile", "")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, profile)
	assert.Equal(t, true, profile.Initialize)
}

func TestInheritedInitializeValueTrue(t *testing.T) {
	testConfig := `[parent]
initialize = true

[profile]
inherit = "parent"
`
	profile, err := getProfile("toml", testConfig, "profile", "")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, profile)
	assert.Equal(t, true, profile.Initialize)
}

func TestOverriddenInitializeValueFalse(t *testing.T) {
	testConfig := `[parent]
initialize = true

[profile]
initialize = false
inherit = "parent"
`
	profile, err := getProfile("toml", testConfig, "profile", "")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, profile)
	assert.Equal(t, false, profile.Initialize)
}

func TestUnknownParent(t *testing.T) {
	testConfig := `[profile]
inherit = "parent"
`
	_, err := getProfile("toml", testConfig, "profile", "")
	assert.Error(t, err)
}

func TestMultiInheritance(t *testing.T) {
	testConfig := `
[grand-parent]
repository = "grand-parent"
first-value = 1
override-value = 1

[parent]
inherit = "grand-parent"
initialize = true
repository = "parent"
second-value = 2
override-value = 2
quiet = true

[profile]
inherit = "parent"
third-value = 3
verbose = true
quiet = false
`
	profile, err := getProfile("toml", testConfig, "profile", "")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, profile)
	assert.Equal(t, "profile", profile.Name)
	assert.Equal(t, "parent", profile.Repository.String())
	assert.Equal(t, true, profile.Initialize)
	assert.Equal(t, int64(1), profile.OtherFlags["first-value"])
	assert.Equal(t, int64(2), profile.OtherFlags["second-value"])
	assert.Equal(t, int64(3), profile.OtherFlags["third-value"])
	assert.Equal(t, int64(2), profile.OtherFlags["override-value"])
	assert.Equal(t, false, profile.Quiet)
	assert.Equal(t, true, profile.Verbose)
}

func TestProfileCommonFlags(t *testing.T) {
	assert := assert.New(t)
	testConfig := `
[profile]
quiet = true
verbose = false
repository = "test"
`
	profile, err := getProfile("toml", testConfig, "profile", "")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(profile)

	flags := profile.GetCommonFlags().ToMap()
	assert.NotNil(flags)
	assert.Contains(flags, "quiet")
	assert.NotContains(flags, "verbose")
	assert.Contains(flags, "repo")
}

func TestProfileOtherFlags(t *testing.T) {
	assert := assert.New(t)
	testConfig := `
[profile]
bool-true = true
bool-false = false
string = "test"
zero = 0
empty = ""
float = 4.2
int = 42
# comment
array0 = []
array1 = [1]
array2 = ["one", "two"]
`
	profile, err := getProfile("toml", testConfig, "profile", "")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(profile)

	flags := profile.GetCommonFlags().ToMap()
	assert.NotNil(flags)
	assert.Contains(flags, "bool-true")
	assert.NotContains(flags, "bool-false")
	assert.Contains(flags, "string")
	assert.NotContains(flags, "zero")
	assert.NotContains(flags, "empty")
	assert.Contains(flags, "float")
	assert.Contains(flags, "int")
	assert.NotContains(flags, "array0")
	assert.Contains(flags, "array1")
	assert.Contains(flags, "array2")

	assert.Equal([]string{}, flags["bool-true"])
	assert.Equal([]string{"test"}, flags["string"])
	assert.Equal([]string{strconv.FormatFloat(4.2, 'f', -1, 64)}, flags["float"])
	assert.Equal([]string{"42"}, flags["int"])
	assert.Equal([]string{"1"}, flags["array1"])
	assert.Equal([]string{"one", "two"}, flags["array2"])
}

func TestSetRootInProfileUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}
	testConfig := `
[profile]
status-file = "status"
password-file = "key"
lock = "lock"
[profile.backup]
source = ["backup", "root"]
exclude-file = "exclude"
files-from = "include"
exclude = "exclude"
iexclude = "iexclude"
[profile.copy]
password-file = "key"
[profile.dump]
password-file = "key"
`
	profile, err := getProfile("toml", testConfig, "profile", "")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, profile)

	profile.SetRootPath("/wd")
	assert.Equal(t, "status", profile.StatusFile)
	assert.Equal(t, "/wd/key", profile.PasswordFile)
	assert.Equal(t, "/wd/lock", profile.Lock)
	assert.Equal(t, "", profile.CacheDir)
	assert.ElementsMatch(t, []string{"backup", "root"}, profile.GetBackupSource())
	assert.ElementsMatch(t, []string{"/wd/exclude"}, profile.Backup.ExcludeFile)
	assert.ElementsMatch(t, []string{"/wd/include"}, profile.Backup.FilesFrom)
	assert.ElementsMatch(t, []string{"exclude"}, profile.Backup.Exclude)
	assert.ElementsMatch(t, []string{"iexclude"}, profile.Backup.Iexclude)
	assert.Equal(t, "/wd/key", profile.Copy.PasswordFile)
	assert.Equal(t, []string{"/wd/key"}, profile.Dump["password-file"])
}

func TestHostInProfile(t *testing.T) {
	assert := assert.New(t)
	testConfig := `
[profile]
initialize = true
[profile.backup]
host = true
[profile.snapshots]
host = "ConfigHost"
`
	profile, err := getProfile("toml", testConfig, "profile", "")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(profile)

	profile.SetHost("TestHost")

	flags := profile.GetCommandFlags(constants.CommandBackup).ToMap()
	assert.NotNil(flags)
	assert.Contains(flags, "host")
	assert.Equal([]string{"TestHost"}, flags["host"])

	flags = profile.GetCommandFlags(constants.CommandSnapshots).ToMap()
	assert.NotNil(flags)
	assert.Contains(flags, "host")
	assert.Equal([]string{"ConfigHost"}, flags["host"])
}

func TestHostInAllSupportedSections(t *testing.T) {
	assert := assert.New(t)

	// Sections supporting "host" flag
	sections := []string{
		constants.CommandBackup,
		constants.CommandForget,
		constants.CommandSnapshots,
		constants.CommandMount,
		constants.SectionConfigurationRetention,
		constants.CommandCopy,
		constants.CommandDump,
		constants.CommandFind,
		constants.CommandLs,
		constants.CommandRestore,
		constants.CommandStats,
		constants.CommandTag,
	}

	assertHostIs := func(expectedHost []string, profile *Profile, section string) {
		assert.NotNil(profile)

		flags := shell.NewArgs()
		if section == constants.SectionConfigurationRetention {
			flags = addOtherArgs(flags, profile.Retention.OtherFlags)
		} else {
			flags = profile.GetCommandFlags(section)
		}

		assert.NotNil(flags)
		assert.Contains(flags.ToMap(), "host")
		assert.Equal(expectedHost, flags.ToMap()["host"])
	}

	testConfig := func(section, host string) string {
		return fmt.Sprintf(`
[profile]
initialize = true
[profile.%s]
host = %s
`, section, host)
	}

	for _, section := range sections {
		// Check that host can be set globally
		profile, err := getProfile("toml", testConfig(section, "true"), "profile", "")
		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(profile)

		assertHostIs(emptyStringArray, profile, section)
		profile.SetHost("TestHost")
		assertHostIs([]string{"TestHost"}, profile, section)

		// Ensure host is set only when host value is true
		profile, err = getProfile("toml", testConfig(section, `"OtherTestHost"`), "profile", "")
		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(profile)

		assertHostIs([]string{"OtherTestHost"}, profile, section)
		profile.SetHost("TestHost")
		assertHostIs([]string{"OtherTestHost"}, profile, section)
	}
}

func TestResolveGlobSourcesInBackup(t *testing.T) {
	examples, err := filepath.Abs("../examples")
	require.NoError(t, err)
	sourcePattern := filepath.ToSlash(filepath.Join(examples, "[a-p]*"))
	testConfig := `
[profile.backup]
source = "` + sourcePattern + `"
`
	profile, err := getProfile("toml", testConfig, "profile", "./examples")
	require.NoError(t, err)
	assert.NotNil(t, profile)

	sources, err := filepath.Glob(sourcePattern)
	require.NoError(t, err)
	assert.Greater(t, len(sources), 5)
	assert.Equal(t, sources, profile.Backup.Source)
}

func TestPathAndTagInRetention(t *testing.T) {
	cwd, err := filepath.Abs(".")
	require.NoError(t, err)
	examples := filepath.Join(cwd, "../examples")
	sourcePattern := filepath.ToSlash(filepath.Join(examples, "[a-p]*"))
	backupSource, err := filepath.Glob(sourcePattern)
	require.Greater(t, len(backupSource), 5)
	require.NoError(t, err)

	backupTags := []string{"one", "two"}

	testProfile := func(t *testing.T, version Version, retention string) *Profile {
		p := ""
		if version > Version01 {
			p = "profiles."
		}

		config := `
			version = ` + fmt.Sprintf("%d", version) + `

			[` + p + `profile.backup]
			tag = ["one", "two"]
			source = ["` + sourcePattern + `"]

			[` + p + `profile.retention]
			` + retention

		profile, err := getResolvedProfile("toml", config, "profile")
		profile.SetRootPath(examples) // ensure relative paths are converted to absolute paths
		require.NoError(t, err)
		require.NotNil(t, profile)

		return profile
	}

	t.Run("Path", func(t *testing.T) {
		pathFlag := func(t *testing.T, profile *Profile) interface{} {
			flags := profile.GetRetentionFlags().ToMap()
			assert.NotNil(t, flags)
			return flags["path"]
		}

		t.Run("ImplicitCopyPath", func(t *testing.T) {
			profile := testProfile(t, Version01, ``)
			assert.Equal(t, backupSource, pathFlag(t, profile))
		})

		t.Run("ExplicitCopyPath", func(t *testing.T) {
			expectedIssues := map[string][]string{
				`path (from source) "` + sourcePattern + `"`: backupSource,
			}
			profile := testProfile(t, Version01, `path = true`)
			assert.Equal(t, backupSource, pathFlag(t, profile))
			assert.Equal(t, expectedIssues, profile.config.issues.changedPaths)
		})

		t.Run("ReplacePath", func(t *testing.T) {
			expected := []string{
				filepath.Join(cwd, "relative/custom/path"),
				cwd,
			}
			expectedIssues := map[string][]string{
				`path "relative/custom/path"`: {expected[0]},
				`path "."`:                    {expected[1]},
			}
			profile := testProfile(t, Version01, `path = ["relative/custom/path", "."]`)
			assert.Equal(t, expected, pathFlag(t, profile))
			assert.Equal(t, expectedIssues, profile.config.issues.changedPaths)
		})

		t.Run("NoPath", func(t *testing.T) {
			profile := testProfile(t, Version01, `path = false`)
			assert.Nil(t, pathFlag(t, profile))
		})
	})

	t.Run("Tag", func(t *testing.T) {
		tagFlag := func(t *testing.T, profile *Profile) interface{} {
			flags := profile.GetRetentionFlags().ToMap()
			assert.NotNil(t, flags)
			return flags["tag"]
		}

		t.Run("NoImplicitCopyTagInV1", func(t *testing.T) {
			profile := testProfile(t, Version01, ``)
			assert.Nil(t, tagFlag(t, profile))
		})

		t.Run("ImplicitCopyTagInV2", func(t *testing.T) {
			profile := testProfile(t, Version02, ``)
			assert.Equal(t, backupTags, tagFlag(t, profile))
		})

		t.Run("CopyTag", func(t *testing.T) {
			profile := testProfile(t, Version01, `tag = true`)
			assert.Equal(t, backupTags, tagFlag(t, profile))
		})

		t.Run("ReplaceTag", func(t *testing.T) {
			profile := testProfile(t, Version01, `tag = ["a", "b"]`)
			expected := []string{"a", "b"}
			assert.Equal(t, expected, tagFlag(t, profile))
		})

		t.Run("NoTag", func(t *testing.T) {
			profile := testProfile(t, Version01, `tag = false`)
			assert.Nil(t, tagFlag(t, profile))
		})
	})
}

func TestForgetCommandFlags(t *testing.T) {
	testData := []testTemplate{
		{"toml", `
[profile]
initialize = true

[profile.backup]
source = "/"

[profile.forget]
keep-daily = 1
`},
		{"json", `
{
  "profile": {
    "backup": {"source": "/"},
    "forget": {"keep-daily": 1}
  }
}`},
		{"yaml", `---
profile:
  backup:
    source: "/"
  forget:
    keep-daily: 1
`},
		{"hcl", `
"profile" = {
	backup = {
		source = "/"
	}
	forget = {
		keep-daily = 1
	}
}
`},
	}

	for _, testItem := range testData {
		format := testItem.format
		testConfig := testItem.config
		t.Run(format, func(t *testing.T) {
			profile, err := getProfile(format, testConfig, "profile", "")
			require.NoError(t, err)

			assert.NotNil(t, profile)
			assert.NotNil(t, profile.Forget)
			assert.NotEmpty(t, profile.Forget.OtherFlags["keep-daily"])
		})
	}
}

func TestSchedules(t *testing.T) {
	assert := assert.New(t)

	testConfig := func(command string, scheduled bool) string {
		schedule := ""
		if scheduled {
			schedule = `schedule = "@hourly"`
		}

		config := `
[profile]
initialize = true

[profile.%s]
%s
`
		return fmt.Sprintf(config, command, schedule)
	}

	sections := NewProfile(nil, "").SchedulableCommands()
	assert.Len(sections, 6)

	for _, command := range sections {
		// Check that schedule is supported
		profile, err := getProfile("toml", testConfig(command, true), "profile", "")
		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(profile)

		config := profile.Schedules()
		assert.Len(config, 1)
		assert.Equal(config[0].commandName, command)
		assert.Len(config[0].schedules, 1)
		assert.Equal(config[0].schedules[0], "@hourly")

		// Check that schedule is optional
		profile, err = getProfile("toml", testConfig(command, false), "profile", "")
		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(profile)
		assert.Empty(profile.Schedules())
	}
}

// schedule is moving from "retention" to "forget" section
// first test: check the schedule works in "forget" section
func TestForgetSchedule(t *testing.T) {
	testData := []testTemplate{
		{"toml", `
[profile]
initialize = true

[profile.backup]
source = "/"

[profile.forget]
schedule = "weekly"
`},
		{"json", `
{
  "profile": {
    "backup": {"source": "/"},
    "forget": {"schedule": "weekly"}
  }
}`},
		{"yaml", `---
profile:
  backup:
    source: /
  forget:
    schedule: weekly
`},
		{"hcl", `
"profile" = {
	backup = {
		source = "/"
	}
	forget = {
		schedule = "weekly"
	}
}
`},
	}

	for _, testItem := range testData {
		format := testItem.format
		testConfig := testItem.config
		t.Run(format, func(t *testing.T) {
			profile, err := getProfile(format, testConfig, "profile", "")
			require.NoError(t, err)

			assert.NotNil(t, profile)
			assert.NotNil(t, profile.Forget)
			assert.NotEmpty(t, profile.Forget.Schedule)
			assert.False(t, profile.HasDeprecatedRetentionSchedule())
		})
	}
}

// schedule is moving from "retention" to "forget" section
// second test: check the schedule deprecation in the "retention" section
func TestRetentionSchedule(t *testing.T) {
	testData := []testTemplate{
		{"toml", `
[profile]
initialize = true

[profile.backup]
source = "/"

[profile.retention]
schedule = "weekly"
`},
		{"json", `
{
  "profile": {
    "backup": {"source": "/"},
    "retention": {"schedule": "weekly"}
  }
}`},
		{"yaml", `---
profile:
  backup:
    source: /
  retention:
    schedule: weekly
`},
		{"hcl", `
"profile" = {
	backup = {
		source = "/"
	}
	retention = {
		schedule = "weekly"
	}
}
`},
	}

	for _, testItem := range testData {
		format := testItem.format
		testConfig := testItem.config
		t.Run(format, func(t *testing.T) {
			profile, err := getProfile(format, testConfig, "profile", "")
			require.NoError(t, err)

			assert.NotNil(t, profile)
			assert.NotNil(t, profile.Retention)
			assert.NotEmpty(t, profile.Retention.Schedule)
			assert.True(t, profile.HasDeprecatedRetentionSchedule())
		})
	}
}

func TestOtherFlags(t *testing.T) {
	testData := []testTemplate{
		{"toml", `
[profile]
other-flag = "1"
[profile.backup]
other-flag-backup = "backup"
[profile.retention]
other-flag-retention = true
[profile.snapshots]
other-flag-snapshots = true
[profile.check]
other-flag-check = true
[profile.forget]
other-flag-forget = true
[profile.prune]
other-flag-prune = true
[profile.mount]
other-flag-mount = true
[profile.copy]
other-flag-copy = true
[profile.dump]
other-flag-dump = true
[profile.find]
other-flag-find = true
[profile.ls]
other-flag-ls = true
[profile.restore]
other-flag-restore = true
[profile.stats]
other-flag-stats = true
[profile.tag]
other-flag-tag = true
`},
		{"json", `
{
  "profile": {
	"other-flag": "1",
    "backup": {"other-flag-backup": "backup"},
    "retention": {"other-flag-retention": true},
    "snapshots": {"other-flag-snapshots": true},
    "check": {"other-flag-check": true},
    "forget": {"other-flag-forget": true},
    "prune": {"other-flag-prune": true},
    "mount": {"other-flag-mount": true},
    "copy": {"other-flag-copy": true},
    "dump": {"other-flag-dump": true},
    "find": {"other-flag-find": true},
    "ls": {"other-flag-ls": true},
    "restore": {"other-flag-restore": true},
    "stats": {"other-flag-stats": true},
    "tag": {"other-flag-tag": true}
  }
}`},
		{"yaml", `---
profile:
  other-flag: 1
  backup:
    other-flag-backup: backup
  retention:
    other-flag-retention: true
  snapshots:
    other-flag-snapshots: true
  check:
    other-flag-check: true
  forget:
    other-flag-forget: true
  prune:
    other-flag-prune: true
  mount:
    other-flag-mount: true
  copy:
    other-flag-copy: true
  dump:
    other-flag-dump: true
  find:
    other-flag-find: true
  ls:
    other-flag-ls: true
  restore:
    other-flag-restore: true
  stats:
    other-flag-stats: true
  tag:
    other-flag-tag: true
`},
		{"hcl", `
"profile" = {
	other-flag = 1
	backup = {
		other-flag-backup = "backup"
	}
	retention = {
		other-flag-retention = true
	}
	snapshots = {
		other-flag-snapshots = true
	}
	check = {
		other-flag-check = true
	}
	forget = {
		other-flag-forget = true
	}
	prune = {
		other-flag-prune = true
	}
	mount = {
		other-flag-mount = true
	}
	copy = {
		other-flag-copy = true
	}
	dump = {
		other-flag-dump = true
	}
	find = {
		other-flag-find = true
	}
	ls = {
		other-flag-ls = true
	}
	restore = {
		other-flag-restore = true
	}
	stats = {
		other-flag-stats = true
	}
	tag = {
		other-flag-tag = true
	}
}
`},
	}

	for _, testItem := range testData {
		format := testItem.format
		testConfig := testItem.config
		t.Run(format, func(t *testing.T) {
			profile, err := getProfile(format, testConfig, "profile", "")
			require.NoError(t, err)

			require.NotNil(t, profile)
			require.NotNil(t, profile.Backup)
			require.NotNil(t, profile.Retention)
			require.NotNil(t, profile.Check)
			require.NotNil(t, profile.Forget)
			require.NotNil(t, profile.Mount)
			require.NotNil(t, profile.Prune)
			require.NotNil(t, profile.Snapshots)
			require.NotNil(t, profile.Copy)
			require.NotNil(t, profile.Dump)
			require.NotNil(t, profile.Find)
			require.NotNil(t, profile.Ls)
			require.NotNil(t, profile.Restore)
			require.NotNil(t, profile.Stats)
			require.NotNil(t, profile.Tag)

			flags := profile.GetCommonFlags()
			assert.Equal(t, 1, len(flags.ToMap()))
			assert.ElementsMatch(t, []string{"1"}, flags.ToMap()["other-flag"])

			flags = profile.GetRetentionFlags()
			assert.Equal(t, 2, len(flags.ToMap()))
			assert.ElementsMatch(t, []string{"1"}, flags.ToMap()["other-flag"])
			_, found := flags.ToMap()["other-flag-retention"]
			assert.True(t, found)

			commands := []string{
				constants.CommandBackup,
				constants.CommandCheck,
				constants.CommandCopy,
				constants.CommandDump,
				constants.CommandFind,
				constants.CommandForget,
				constants.CommandLs,
				constants.CommandPrune,
				constants.CommandMount,
				constants.CommandRestore,
				constants.CommandSnapshots,
				constants.CommandStats,
				constants.CommandTag,
			}

			for _, command := range commands {
				t.Run(command, func(t *testing.T) {
					flags = profile.GetCommandFlags(command)
					commandFlagName := "other-flag-" + command
					assert.Equal(t, 2, len(flags.ToMap()))
					assert.ElementsMatch(t, []string{"1"}, flags.ToMap()["other-flag"])
					_, found = flags.ToMap()[commandFlagName]
					assert.True(t, found, commandFlagName)
				})
			}
		})
	}
}

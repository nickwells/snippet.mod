package snippet_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/nickwells/errutil.mod/errutil"
	"github.com/nickwells/snippet.mod/snippet"
	"github.com/nickwells/testhelper.mod/testhelper"
)

const (
	testDataDir    = "testdata"
	snippetListOut = "snippetListOut"
)

var gfc = testhelper.GoldenFileCfg{
	DirNames:               []string{testDataDir, snippetListOut},
	Sfx:                    "txt",
	UpdFlagName:            "upd-snippet-list-files",
	KeepBadResultsFlagName: "keep-bad-results",
}

func init() {
	gfc.AddUpdateFlag()
	gfc.AddKeepBadResultsFlag()
}

func TestList(t *testing.T) {
	defer testhelper.MakeTempDir(snippet.EmptyDir, 0777)()
	defer testhelper.MakeTempDir(snippet.UnreadableDir, 0)()
	defer testhelper.MakeTempDir(snippet.BadDir, 0777)()
	defer testhelper.MakeTempDir(snippet.UnreadableSubDir, 0)()
	defer testhelper.TempChmod(snippet.UnreadableFile, 0333)()

	testCases := []struct {
		testhelper.ID
		dirs    []string
		expErrs errutil.ErrMap
	}{
		{
			ID: testhelper.MkID("noDirs.noErrs"),
		},
		{
			ID:   testhelper.MkID("oneDir.noErrs"),
			dirs: []string{snippet.GoodSnippets},
		},
		{
			ID: testhelper.MkID("twoDirs.oneEmpty.noErrs"),
			dirs: []string{
				snippet.GoodSnippets,
				snippet.EmptyDir,
			},
		},
		{
			ID: testhelper.MkID("twoGoodDirs.eclipses"),
			dirs: []string{
				snippet.GoodSnippets,
				snippet.MoreGoodSnippets,
			},
			expErrs: errutil.ErrMap{
				"Eclipsed snippet": []error{
					errors.New(`"hw" in "` + snippet.MoreGoodSnippets + `"` +
						` is eclipsed by the entry` +
						` in "` + snippet.GoodSnippets + `"`),
				},
			},
		},
		{
			ID: testhelper.MkID("threeDirs.oneEmpty.oneNonExistant.noErrs"),
			dirs: []string{
				snippet.GoodSnippets,
				snippet.EmptyDir,
				snippet.NoSuchDir,
			},
		},
		{
			ID: testhelper.MkID("twoDirs.oneUnreadable"),
			dirs: []string{
				snippet.GoodSnippets,
				snippet.UnreadableDir,
			},
			expErrs: errutil.ErrMap{
				`Bad snippets directory: "` + snippet.UnreadableDir + `"`: []error{
					errors.New("open " + snippet.UnreadableDir +
						": permission denied"),
				},
			},
		},
		{
			ID: testhelper.MkID("twoDirs.badSnippets"),
			dirs: []string{
				snippet.GoodSnippets,
				snippet.BadSnippets,
			},
			expErrs: errutil.ErrMap{
				`Bad snippet`: []error{
					errors.New(
						`snippet "noText"` +
							` (` + snippet.BadSnippets + `/noText)` +
							` has no text and no imports`),
					errors.New(
						`snippet "toBeMadeUnreadable":` +
							` open ` + snippet.UnreadableFile + `:` +
							` permission denied`),
				},
			},
		},
		{
			ID: testhelper.MkID("twoDirs.unreadableSubDir"),
			dirs: []string{
				snippet.GoodSnippets,
				snippet.BadDir,
			},
			expErrs: errutil.ErrMap{
				`Bad sub-directory: "unreadable.dir"`: []error{
					errors.New(`open ` + snippet.UnreadableSubDir + `:` +
						` permission denied`),
				},
			},
		},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		errs := errutil.NewErrMap()
		snippet.List(&buf, tc.dirs, errs)
		if err := errs.Matches(tc.expErrs); err != nil {
			t.Log(tc.IDStr())
			t.Log("\t: differences:", err)
			t.Errorf("\t: unexpected error map\n")
			continue
		}
		gfc.Check(t, tc.IDStr(), tc.ID.Name, buf.Bytes())
	}
}

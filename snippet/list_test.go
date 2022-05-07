package snippet_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/nickwells/errutil.mod/errutil"
	"github.com/nickwells/snippet.mod/snippet"
	"github.com/nickwells/testhelper.mod/v2/testhelper"
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

func TestConfigList(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Couldn't get the current working directory:", err)
	}
	testListCfgDir := filepath.Join(cwd, "testdata", "testListConfig")
	sNameSnip1 := filepath.Join(testListCfgDir, "snip1")
	sNameSnip2 := filepath.Join(testListCfgDir, "snip2")
	testListCfgDir2 := filepath.Join(cwd, "testdata", "testListConfig2")
	sNameSnip3 := filepath.Join(testListCfgDir2, "snip3")
	testCases := []struct {
		testhelper.ID
		dirs    []string
		expErrs errutil.ErrMap
		opts    []snippet.ListCfgOptFunc
	}{
		{
			ID:   testhelper.MkID("configList.dflt"),
			dirs: []string{snippet.GoodSnippets},
		},
		{
			ID:   testhelper.MkID("configList.hideIntro"),
			dirs: []string{snippet.GoodSnippets},
			opts: []snippet.ListCfgOptFunc{snippet.HideIntro(true)},
		},
		{
			ID:   testhelper.MkID("configList.specific-hw"),
			dirs: []string{snippet.GoodSnippets},
			opts: []snippet.ListCfgOptFunc{snippet.SetConstraints("hw")},
		},
		{
			ID:   testhelper.MkID("configList.specific-subDir1"),
			dirs: []string{snippet.GoodSnippets},
			opts: []snippet.ListCfgOptFunc{snippet.SetConstraints("subDir1")},
		},
		{
			ID:   testhelper.MkID("configList.specific-subDir1-file"),
			dirs: []string{snippet.GoodSnippets},
			opts: []snippet.ListCfgOptFunc{
				snippet.SetConstraints("subDir1/goodNoExp"),
			},
		},
		{
			ID:   testhelper.MkID("configList.specific-snip1"),
			dirs: []string{snippet.GoodSnippets},
			opts: []snippet.ListCfgOptFunc{snippet.SetConstraints(sNameSnip1)},
		},
		{
			ID:   testhelper.MkID("configList.specific-snip2"),
			dirs: []string{snippet.GoodSnippets},
			opts: []snippet.ListCfgOptFunc{snippet.SetConstraints(sNameSnip2)},
		},
		{
			ID:   testhelper.MkID("configList.specific-testListCfgDir"),
			dirs: []string{snippet.GoodSnippets},
			opts: []snippet.ListCfgOptFunc{
				snippet.SetConstraints(testListCfgDir),
			},
		},
		{
			ID:   testhelper.MkID("configList.snip3.Docs"),
			dirs: []string{snippet.GoodSnippets},
			opts: []snippet.ListCfgOptFunc{
				snippet.SetConstraints(sNameSnip3),
				snippet.SetParts(snippet.DocsPart),
			},
		},
		{
			ID:   testhelper.MkID("configList.snip3.Docs.HideIntro"),
			dirs: []string{snippet.GoodSnippets},
			opts: []snippet.ListCfgOptFunc{
				snippet.SetConstraints(sNameSnip3),
				snippet.SetParts(snippet.DocsPart),
				snippet.HideIntro(true),
			},
		},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		errs := errutil.NewErrMap()
		lc, err := snippet.NewListCfg(&buf, tc.dirs, errs, tc.opts...)
		if err != nil {
			t.Fatal("Unexpected error:", err)
		}

		lc.List()

		if err = errs.Matches(tc.expErrs); err != nil {
			var errRpt bytes.Buffer
			errs.Report(&errRpt, "Snippet errors")
			t.Log(tc.IDStr())
			t.Log("\t: differences:", err)
			t.Log("\t: error map:\n", errRpt.String())
			t.Errorf("\t: unexpected error map\n\n")
			continue
		}
		gfc.Check(t, tc.IDStr(), tc.ID.Name, buf.Bytes())
	}
}

func TestList(t *testing.T) {
	defer testhelper.MakeTempDir(snippet.EmptyDir, 0o777)()
	defer testhelper.MakeTempDir(snippet.UnreadableDir, 0)()
	defer testhelper.MakeTempDir(snippet.BadDir, 0o777)()
	defer testhelper.MakeTempDir(snippet.UnreadableSubDir, 0)()
	defer testhelper.TempChmod(snippet.UnreadableFile, 0o333)()

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
				"Duplicate snippet": []error{
					errors.New(`snippet` +
						` "` + snippet.BadSnippets + `/duplicate2"` +
						` is a duplicate of` +
						` "` + snippet.BadSnippets + `/duplicate1"`),
				},
				"Missing expected snippet": []error{
					errors.New(`snippet "noSuchSnippet"` +
						` does not exist but is 'expected' by` +
						` "badExpectations"`),
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
			var errRpt bytes.Buffer
			errs.Report(&errRpt, "Snippet errors")
			t.Log(tc.IDStr())
			t.Log("\t: differences:", err)
			t.Log("\t: error map:\n", errRpt.String())
			t.Errorf("\t: unexpected error map\n\n")
			continue
		}
		gfc.Check(t, tc.IDStr(), tc.ID.Name, buf.Bytes())
	}
}

func TestNewListCfgSetParts(t *testing.T) {
	badPart := "blah blah blah"
	testCases := []struct {
		testhelper.ID
		testhelper.ExpErr
		parts []string
	}{
		{
			ID:    testhelper.MkID("good parts"),
			parts: []string{snippet.ImportPart},
		},
		{
			ID: testhelper.MkID("bad parts"),
			ExpErr: testhelper.MkExpErr(`"` + badPart + `" is` +
				` not a valid pre-defined part of a snippet`),
			parts: []string{badPart},
		},
	}

	for _, tc := range testCases {
		_, err := snippet.NewListCfg(nil, nil, nil,
			snippet.SetParts(tc.parts...))
		testhelper.CheckExpErr(t, err, tc)
	}
}

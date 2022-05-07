package snippet

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/nickwells/errutil.mod/errutil"
	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestSnippetCache(t *testing.T) {
	type sNameErr struct {
		name   string
		expErr error
	}
	snippetDirs := []string{TestSnippets}
	const badNoText = "badNoText"

	testCases := []struct {
		testhelper.ID
		expCheckErrs errutil.ErrMap
		snippets     []sNameErr
	}{
		{
			ID: testhelper.MkID("one snippet, no expectations"),
			snippets: []sNameErr{
				{name: "goodNoExp"},
			},
		},
		{
			ID: testhelper.MkID("duplicate snippet, no expectations"),
			snippets: []sNameErr{
				{name: "goodNoExp"},
				{name: "goodNoExp"},
			},
		},
		{
			ID: testhelper.MkID("one snippet - bad - no text or imports"),
			snippets: []sNameErr{
				{
					name: badNoText,
					expErr: errors.New(`snippet "` + badNoText + `" (` +
						filepath.Join(TestSnippets, badNoText) +
						`) has no text and no imports`),
				},
			},
		},
		{
			ID: testhelper.MkID("3 snippets - 2 bad - don't exist"),
			snippets: []sNameErr{
				{name: "goodNoExp"},
				{
					name: "nonesuch",
					expErr: errors.New(`snippet "nonesuch"` +
						` is not in the snippet directory:` +
						` "` + TestSnippets + `"`),
				},
				{
					name: "nonesuch",
					expErr: errors.New(`snippet "nonesuch"` +
						` is not in the snippet directory:` +
						` "` + TestSnippets + `"`),
				},
			},
		},
		{
			ID: testhelper.MkID("3 snippets - with expectations, all met"),
			snippets: []sNameErr{
				{name: "expects1"},
				{name: "expects2"},
				{name: "expects3"},
			},
		},
		{
			ID: testhelper.MkID("2 snippets - with expectations, one missing"),
			snippets: []sNameErr{
				{name: "expects1"},
				{name: "expects2"},
			},
			expCheckErrs: errutil.ErrMap{
				`Missing snippet "expects3"`: []error{
					errors.New(`expected by "expects2"`),
				},
			},
		},
	}

	for _, tc := range testCases {
		sc := Cache{}
		for i, sne := range tc.snippets {
			s, err := sc.Add(snippetDirs, sne.name)
			id := tc.IDStr() + fmt.Sprintf(" [%d]", i)
			testhelper.DiffErr(t, id, "error from Add(...)", err, sne.expErr)
			if err == nil {
				sg, err := sc.Get(sne.name)
				if err != nil {
					t.Log(id)
					t.Logf("\t: calling Cache.Get(%s)", sne.name)
					t.Errorf("\t: unexpected err: %s", err)
				} else {
					err = s.Matches(*sg)
					if err != nil {
						t.Log(id)
						t.Log("\t: the snippet returned by Get differs")
						t.Errorf("\t: differences: %s", err)
					}
				}
				testhelper.DiffString(t, id, "snippet name", s.Name(), sne.name)
				testhelper.DiffString(t, id, "snippet path",
					s.Path(), filepath.Join(TestSnippets, sne.name))
			} else {
				sg, err := sc.Get(sne.name)
				if err == nil {
					t.Log(id)
					t.Logf("\t: calling Cache.Get(%s)", sne.name)
					t.Errorf("\t: unexpected success: %s", sg)
				}
			}
		}
		errMap := errutil.NewErrMap()
		sc.Check(errMap)
		err := errMap.Matches(tc.expCheckErrs)
		if err != nil {
			t.Log(tc.IDStr())
			t.Log("\t: checking the snippet cache")
			t.Errorf("\t: unexpected error: %s", err)
		}
	}
}

func TestSnippet(t *testing.T) {
	const completeSnip = "complete"
	testCases := []struct {
		testhelper.ID
		dirs       []string
		sName      string
		expErr     error
		expText    []string
		expName    string
		expPath    string
		expDocs    []string
		expExpects []string
		expImports []string
		expFollows []string
		expTags    map[string][]string
	}{
		{
			ID:      testhelper.MkID("complete"),
			dirs:    []string{TestSnippets},
			sName:   completeSnip,
			expText: []string{`fmt.Println("This is a snippet")`},
			expName: completeSnip,
			expPath: filepath.Join(TestSnippets, completeSnip),
			expDocs: []string{"note 1", "note 2"},
			expExpects: []string{
				"anotherSnippet1",
				"anotherSnippet2",
				"anotherSnippet3",
			},
			expImports: []string{"package/one", "package/two"},
			expFollows: []string{"anotherSnippet2", "anotherSnippet3"},
			expTags: map[string][]string{
				"Author": {
					"John Doe",
					"John Barleycorn",
					"Nedd Ludd",
				},
				"XXX": {"YYY", "YYY yyy"},
			},
		},
	}

	for _, tc := range testCases {
		id := tc.IDStr()
		sc := Cache{}
		s, err := sc.Add(tc.dirs, tc.sName)
		testhelper.DiffErr(t, id, "error", err, tc.expErr)
		testhelper.DiffStringSlice(t, id, "text", s.Text(), tc.expText)
		testhelper.DiffString(t, id, "name", s.Name(), tc.expName)
		testhelper.DiffString(t, id, "path", s.Path(), tc.expPath)
		testhelper.DiffStringSlice(t, id, "docs", s.Docs(), tc.expDocs)
		testhelper.DiffStringSlice(t, id, "expects", s.Expects(), tc.expExpects)
		testhelper.DiffStringSlice(t, id, "imports", s.Imports(), tc.expImports)
		testhelper.DiffStringSlice(t, id, "follows", s.Follows(), tc.expFollows)
		if err = cmpTags(s.Tags(), tc.expTags); err != nil {
			t.Log(id)
			t.Logf("\t: %s", err)
			t.Error("\t: the tags differ")
		}
	}
}

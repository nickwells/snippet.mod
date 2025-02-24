package snippet

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

var (
	NoSuchDir        = filepath.Join("testdata", "no.such.dir")
	EmptyDir         = filepath.Join("testdata", "empty.dir")
	UnreadableDir    = filepath.Join("testdata", "unreadable.dir")
	GoodSnippets     = filepath.Join("testdata", "good.snippets")
	MoreGoodSnippets = filepath.Join("testdata", "more.good.snippets")
	BadSnippets      = filepath.Join("testdata", "bad.snippets")
	UnreadableFile   = filepath.Join("testdata", "bad.snippets", "toBeMadeUnreadable")
	BadDir           = filepath.Join("testdata", "bad.dir")
	UnreadableSubDir = filepath.Join("testdata", "bad.dir", "unreadable.dir")

	TestSnippets = filepath.Join("testdata", "test.snippets")
)

func TestCmpSlice(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		s1     []string
		s2     []string
		name   string
		expErr error
	}{
		{
			ID:   testhelper.MkID("empty slices - no err"),
			name: "empty slices",
		},
		{
			ID:   testhelper.MkID("slices length differ s1<s2"),
			s1:   []string{"Hello", "World"},
			s2:   []string{"Hello", "World", "etc."},
			name: "different length slices",
			expErr: errors.New(`different length slices differs:
	the lengths differ: 2 != 3`),
		},
		{
			ID:   testhelper.MkID("slices length differ s2<s1"),
			s1:   []string{"Hello", "World", "etc."},
			s2:   []string{"Hello", "World"},
			name: "different length slices",
			expErr: errors.New(`different length slices differs:
	the lengths differ: 3 != 2`),
		},
		{
			ID:   testhelper.MkID("same length, different content"),
			s1:   []string{"Hello", "World"},
			s2:   []string{"Bonjour", "le Monde"},
			name: "different content slices",
			expErr: errors.New(`different content slices differs:
	entry[0] differs: "Hello" != "Bonjour"
	an additional difference was found`),
		},
		{
			ID:   testhelper.MkID("same length, different content (2 diffs)"),
			s1:   []string{"Hello", "World", "and other things"},
			s2:   []string{"Bonjour", "le Monde", "etc"},
			name: "different content slices",
			expErr: errors.New(`different content slices differs:
	entry[0] differs: "Hello" != "Bonjour"
	2 additional differences were found`),
		},
	}

	for _, tc := range testCases {
		err := cmpSlice(tc.name, tc.s1, tc.s2)
		testhelper.DiffErr(t, tc.IDStr(), "error", err, tc.expErr)
	}
}

func TestCmpTags(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		t1     map[string][]string
		t2     map[string][]string
		expErr error
	}{
		{
			ID: testhelper.MkID("empty tag maps - no error"),
		},
		{
			ID: testhelper.MkID("identical tag maps - no error"),
			t1: map[string][]string{
				"A": {"Hello, World!"},
				"B": {"the", "quality", "of", "mercy"},
			},
			t2: map[string][]string{
				"A": {"Hello, World!"},
				"B": {"the", "quality", "of", "mercy"},
			},
		},
		{
			ID: testhelper.MkID("extra tag maps - error"),
			t1: map[string][]string{
				"A": {"Hello, World!"},
				"B": {"the", "quality", "of", "mercy"},
				"C": {"the", "quality", "of", "mercy"},
			},
			t2: map[string][]string{
				"A": {"Hello, World!"},
				"B": {"the", "quality", "of", "mercy"},
				"D": {"the", "quality", "of", "mercy"},
			},
			expErr: errors.New(`the tag names differ:
	"C" in this, not in other
	"D" in other, not in this`),
		},
		{
			ID: testhelper.MkID("differing slices - error"),
			t1: map[string][]string{
				"A": {"Hello, World!"},
				"B": {"the", "quality", "of", "mercy"},
				"C": {"the", "quality", "of", "mercy"},
			},
			t2: map[string][]string{
				"A": {"Hello, World!"},
				"B": {"the", "quality", "of", "mercy"},
				"C": {"is", "not", "strained"},
			},
			expErr: errors.New(`Tag:C differs:
	the lengths differ: 4 != 3
	entry[0] differs: "the" != "is"
	2 additional differences were found`),
		},
	}

	for _, tc := range testCases {
		err := cmpTags(tc.t1, tc.t2)
		testhelper.DiffErr(t, tc.IDStr(), "error", err, tc.expErr)
	}
}

func TestTidySlice(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		slc    []string
		expSlc []string
	}{
		{
			ID: testhelper.MkID("empty"),
		},
		{
			ID:  testhelper.MkID("all blank"),
			slc: []string{"", ""},
		},
		{
			ID:     testhelper.MkID("jumbled"),
			slc:    []string{"Z", "M", "A"},
			expSlc: []string{"A", "M", "Z"},
		},
		{
			ID:     testhelper.MkID("has dups"),
			slc:    []string{"Z", "Z", "Z"},
			expSlc: []string{"Z"},
		},
		{
			ID:     testhelper.MkID("many issues"),
			slc:    []string{"Z", "", "Z", "Z", "M", "", "", "M", "A"},
			expSlc: []string{"A", "M", "Z"},
		},
	}

	for _, tc := range testCases {
		tidied := tidySlice(tc.slc)
		testhelper.DiffStringSlice(t, tc.IDStr(), "", tidied, tc.expSlc)
	}
}

func TestReadSnippetFile(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Cannot get the current working directory: ", err)
	}

	absPath := filepath.Join(wd, GoodSnippets, "hw")

	testCases := []struct {
		testhelper.ID
		dirs     []string
		sName    string
		expFName string
		expErr   error
	}{
		{
			ID:       testhelper.MkID("no directories to search"),
			sName:    "any",
			expFName: "",
			expErr:   errors.New("there are no snippet directories to search"),
		},
		{
			ID:       testhelper.MkID("absolute path with no dirs to search"),
			sName:    absPath,
			expFName: absPath,
		},
		{
			ID:       testhelper.MkID("absolute path with dirs to search"),
			dirs:     []string{GoodSnippets},
			sName:    absPath,
			expFName: absPath,
		},
		{
			ID:       testhelper.MkID("single directory no match"),
			dirs:     []string{"noDir1"},
			sName:    "any",
			expFName: "",
			expErr: errors.New(`snippet "any" is not in` +
				` the snippet directory: "noDir1"`),
		},
		{
			ID:       testhelper.MkID("multiple directories no match"),
			dirs:     []string{"noDir1", "noDir2"},
			sName:    "any",
			expFName: "",
			expErr: errors.New(`snippet "any" is not in` +
				` any snippet directory: "noDir1", "noDir2"`),
		},
		{
			ID:       testhelper.MkID("a match in a snippet directory"),
			dirs:     []string{GoodSnippets},
			sName:    "hw",
			expFName: GoodSnippets + "/hw",
		},
	}

	for _, tc := range testCases {
		_, fname, err := readSnippetFile(tc.dirs, tc.sName)
		testhelper.DiffString(t, tc.IDStr(), "error", fname, tc.expFName)
		testhelper.DiffErr(t, tc.IDStr(), "error", err, tc.expErr)
	}
}

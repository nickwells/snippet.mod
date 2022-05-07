package snippet

import (
	"bytes"
	"testing"

	"github.com/nickwells/errutil.mod/errutil"
	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestConstraintsFileMatch(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		constraints []string
		sName       string
		expVal      bool
	}{
		{
			ID:     testhelper.MkID("no constraints"),
			sName:  "file",
			expVal: true,
		},
		{
			ID:          testhelper.MkID("with constraints"),
			constraints: []string{"dir/file"},
			sName:       "dir/file",
			expVal:      true,
		},
		{
			ID:          testhelper.MkID("with constraints, no match"),
			constraints: []string{"dir/file"},
			sName:       "dir/file2",
			expVal:      false,
		},
		{
			ID:          testhelper.MkID("with multiple constraints"),
			constraints: []string{"not/matching", "dir/file"},
			sName:       "dir/file",
			expVal:      true,
		},
		{
			ID:          testhelper.MkID("with multiple constraints, no match"),
			constraints: []string{"not/matching", "not/matching/either"},
			sName:       "dir/file2",
			expVal:      false,
		},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		errs := errutil.NewErrMap()
		lc, _ := NewListCfg(&buf, []string{}, errs,
			SetConstraints(tc.constraints...))
		val := lc.specificFileMatch(tc.sName)
		testhelper.DiffBool(t, tc.IDStr(), "match result", val, tc.expVal)
	}
}

func TestConstraintsDirMatch(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		constraints []string
		subDir      string
		expVal      bool
	}{
		{
			ID:     testhelper.MkID("no specific snippets"),
			subDir: "dir",
			expVal: true,
		},
		{
			ID:          testhelper.MkID("match dir name"),
			constraints: []string{"dir"},
			subDir:      "dir",
			expVal:      true,
		},
		{
			ID:          testhelper.MkID("match base sub-dir name"),
			constraints: []string{"dir/subDir"},
			subDir:      "dir/subDir",
			expVal:      true,
		},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		errs := errutil.NewErrMap()
		lc, _ := NewListCfg(&buf, []string{}, errs,
			SetConstraints(tc.constraints...))
		val := lc.specificDirMatch(tc.subDir)
		testhelper.DiffBool(t, tc.IDStr(), "match result", val, tc.expVal)
	}
}

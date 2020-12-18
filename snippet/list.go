package snippet

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/nickwells/errutil.mod/errutil"
)

// snipLoc records where a snippet is first found. It is used to report
// snippets in one directory which cannot be used because they are hidden
// (eclipsed) by a snippet found earlier in the list of snippet directories.
type snipLoc map[string]string

// record records the location that the snippet is found. It records an error
// and returns it if the snippet is already in the snipLoc
func (sl *snipLoc) record(sName, dir string, errs *errutil.ErrMap) error {
	if otherSD, eclipsed := (*sl)[sName]; eclipsed {
		err := fmt.Errorf("%q in %q is eclipsed by the entry in %q",
			sName, dir, otherSD)
		errs.AddError("Eclipsed snippet", err)
		return err
	}
	(*sl)[sName] = dir
	return nil
}

// List will read all of the snippet directories and show the
// available snippet files. Any errors are recorded in errs.
func List(w io.Writer, dirs []string, errs *errutil.ErrMap) {
	var loc = make(snipLoc)

	for _, dir := range dirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				errs.AddError(fmt.Sprintf("Bad snippets directory: %q", dir),
					err)
			}
			continue
		}
		fmt.Fprintln(w, "in: "+dir)
		for _, f := range files {
			display(w, &loc, dir, "", f, errs)
		}
	}
}

// display reports the file if it is a regular file, descends into the sub
// directory if it is a directory and reports it as a problem otherwise
func display(w io.Writer, loc *snipLoc, dir, subDir string, f os.FileInfo, errs *errutil.ErrMap) {
	sName := f.Name()
	if subDir != "" {
		sName = filepath.Join(subDir, sName)
	}
	fName := filepath.Join(dir, sName)

	if f.Mode().IsRegular() ||
		f.Mode()&os.ModeSymlink == os.ModeSymlink {
		if err := loc.record(sName, dir, errs); err != nil {
			return
		}

		content, err := ioutil.ReadFile(fName)
		if err != nil {
			errs.AddError("Bad snippet", fmt.Errorf("snippet %q: %w", sName, err))
			return
		}
		s, err := parseSnippet(content, fName, sName)
		if err != nil {
			errs.AddError("Bad snippet", err)
			return
		}
		fmt.Fprint(w, "\t", s)
	} else if f.IsDir() {
		descend(w, loc, dir, sName, errs)
	} else {
		errs.AddError("Unexpected file type",
			fmt.Errorf("%q: %s", fName, f.Mode()))
	}
}

// descend displays the contents of the sub directory
func descend(w io.Writer, loc *snipLoc, dir, subDir string, errs *errutil.ErrMap) {
	name := filepath.Join(dir, subDir)
	files, err := ioutil.ReadDir(name)
	if err != nil {
		errs.AddError(fmt.Sprintf("Bad sub-directory: %q", subDir), err)
		return
	}
	for _, f := range files {
		display(w, loc, dir, subDir, f, errs)
	}
}

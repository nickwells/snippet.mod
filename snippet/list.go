package snippet

// we use the crypto/md5 package which is cryptographically weak but we are
// not using it for cryptographic purposes
import (
	"crypto/md5" //nolint:gosec
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nickwells/errutil.mod/errutil"
	"github.com/nickwells/pager.mod/pager"
)

// constraintCk controls whether or not to check constraints
type constraintCk int

const (
	checkConstraints constraintCk = iota
	dontCheckConstraints
)

// ListCfgOptFunc is a function which sets some part of a ListCfg structure
type ListCfgOptFunc func(lc *ListCfg) error

// SetConstraints returns a ListCfgOptFunc which will set on a ListCfg value
// the constraints on the snippets to be shown.
func SetConstraints(vals ...string) ListCfgOptFunc {
	return func(lc *ListCfg) error {
		for _, v := range vals {
			lc.constraints[v] = true
		}

		return nil
	}
}

// SetParts returns a ListCfgOptFunc which will set on a ListCfg value the
// parts of the snippets to be shown.
func SetParts(vals ...string) ListCfgOptFunc {
	return func(lc *ListCfg) error {
		for _, v := range vals {
			if _, ok := validParts[v]; !ok {
				return fmt.Errorf(
					"%q is not a valid pre-defined part of a snippet", v)
			}

			lc.parts[v] = true
		}

		return nil
	}
}

// SetTags returns a ListCfgOptFunc which will set on a ListCfg value the
// tags of the snippets to be shown.
func SetTags(vals ...string) ListCfgOptFunc {
	return func(lc *ListCfg) error {
		for _, v := range vals {
			lc.tags[v] = true
		}

		return nil
	}
}

// HideIntro returns a ListCfgOptFunc which will set up the ListCfg value to
// the given value. Setting it to true will suppress the printing of the
// snippet part names before the values.
func HideIntro(val bool) ListCfgOptFunc {
	return func(lc *ListCfg) error {
		lc.hideIntro = val
		return nil
	}
}

// ListCfg holds the configuration for controlling the listing of snippets
type ListCfg struct {
	formatCfg
	pager.Writers
	// dirs is the list of snippet dirs to search
	dirs []string
	// errs is where to record any errors found while listing
	errs *errutil.ErrMap

	// constraints (if non-empty) will constrain the snippets to show. If this
	// is empty than all snippets will be shown.
	constraints map[string]bool

	// loc records where snippets are first declared. It is used to report
	// snippets in one directory which cannot be used because they are hidden
	// (eclipsed) by a snippet found earlier in the list of snippet
	// directories.
	loc map[string]string

	// contentHash maps a hash of the snippet's content to the full pathname
	// of the snippet. It is used to report duplicate snippets. It is not a
	// fatal error for there to be duplicate snippets as they can still be
	// used but it is reported as an error to allow redundant snippets to be
	// found.
	contentHash map[[md5.Size]byte]string

	// expectedBy maps the name of a snippet to the name of the snippet
	// expecting it. It is used to report missing snippets which are expected
	// by other snippets.
	expectedBy map[string][]string

	// intro is the string to be printed before the first snippet. It will be
	// the name of the current snippet directory and then cleared by
	// printIntroOnce so as to ensure we only print this intro for
	// directories having some snippets in them.
	intro string
}

// NewListCfg returns a new ListCfg holding the configuration for snippet
// listing.
func NewListCfg(w io.Writer, dirs []string,
	errs *errutil.ErrMap, opts ...ListCfgOptFunc,
) (*ListCfg, error) {
	lc := &ListCfg{
		Writers:     pager.W(),
		dirs:        dirs,
		errs:        errs,
		constraints: map[string]bool{},

		loc:         map[string]string{},
		contentHash: map[[md5.Size]byte]string{},
		expectedBy:  map[string][]string{},
	}
	lc.SetStdW(w)
	lc.SetErrW(w)

	lc.parts = map[string]bool{}
	lc.tags = map[string]bool{}

	for _, o := range opts {
		err := o(lc)
		if err != nil {
			return nil, err
		}
	}

	return lc, nil
}

// tidy will clear out any map entries set to false and will clear the loc
// map
func (lc *ListCfg) tidy() {
	for k, v := range lc.constraints {
		if !v {
			delete(lc.constraints, k)
		}
	}

	for k, v := range lc.parts {
		if !v {
			delete(lc.parts, k)
		}
	}

	for k, v := range lc.tags {
		if !v {
			delete(lc.tags, k)
		}
	}

	lc.loc = map[string]string{}
}

// listDir reads the given directory and reports on any snippets it find
// subject to any constraints given by the ListCfg.
func (lc *ListCfg) listDir(dir string, ck constraintCk) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			lc.errs.AddError(
				fmt.Sprintf("Bad snippets directory: %q", dir),
				err)
		}

		return
	}

	if !lc.hideIntro {
		lc.intro = "in: " + dir + "\n"
	}

	for _, de := range dirEntries {
		lc.display(dir, "", de, ck)
	}
}

// List reads the given snippet directories (or specified files and
// directories) and reports them recording errors as it goes.
func (lc *ListCfg) List() {
	lc.tidy()

	pgr := pager.Start(lc)

	for sName := range lc.constraints {
		if filepath.IsAbs(sName) {
			f, err := os.Stat(sName)
			if err != nil {
				lc.errs.AddError("Bad specific snippet",
					fmt.Errorf("snippet %q: %w", sName, err))
				continue
			}

			if f.IsDir() {
				lc.listDir(sName, dontCheckConstraints)
			} else {
				lc.displaySnippet("", sName, sName)
			}
		}
	}

	for _, dir := range lc.dirs {
		lc.listDir(dir, checkConstraints)
	}

	lc.checkExpectedSnippetsExist()
	pgr.Done()
}

// checkExpectedSnippetsExist checks that all the snippets which are expected
// by some snippet are defined somewhere.
func (lc *ListCfg) checkExpectedSnippetsExist() {
	if len(lc.constraints) > 0 {
		return
	}

	var ebKeys []string
	for k := range lc.expectedBy {
		ebKeys = append(ebKeys, k)
	}

	sort.Strings(ebKeys)

	for _, k := range ebKeys {
		if _, ok := lc.loc[k]; !ok {
			lc.errs.AddError("Missing expected snippet",
				fmt.Errorf("snippet %q does not exist but is 'expected' by %q",
					k, strings.Join(lc.expectedBy[k], ", ")))
		}
	}
}

// snippetIsEclipsed records the location that the snippet is found. It records
// an error and returns it if the snippet is already in the snipLoc
func (lc *ListCfg) snippetIsEclipsed(sName, dir string) bool {
	otherSD, eclipsed := (lc.loc)[sName]

	if eclipsed && otherSD != dir {
		lc.errs.AddError("Eclipsed snippet",
			fmt.Errorf("%q in %q is eclipsed by the entry in %q",
				sName, dir, otherSD))

		return true
	}

	(lc.loc)[sName] = dir

	return false
}

// recordSnippetContentHash records all the snippets having the same
// content. These could be simple aliases or else redundant copies. They will
// be recorded as errors though the duplicate snippets are still reported and
// can be used.
func (lc *ListCfg) recordSnippetContentHash(content []byte, fName string) {
	hash := md5.Sum(content) //nolint:gosec
	otherFile, isDup := (lc.contentHash)[hash]

	if isDup {
		lc.errs.AddError("Duplicate snippet",
			fmt.Errorf("snippet %q is a duplicate of %q", fName, otherFile))
		return
	}

	(lc.contentHash)[hash] = fName
}

// recordExpectedBy cross references all the snippets expected by a snippet
// back to the snippet that expects them. The full set of expected snippets
// is checked for existence once all the snippets have been read.
func (lc *ListCfg) recordExpectedBy(s *S, sName string) {
	for _, exp := range s.expects {
		lc.expectedBy[exp] = append(lc.expectedBy[exp], sName)
	}
}

// List will read all of the snippet directories and show the
// available snippet files. Any errors are recorded in errs.
func List(w io.Writer, dirs []string, errs *errutil.ErrMap) {
	lc, _ := NewListCfg(w, dirs, errs)
	lc.List()
}

// displaySnippet reads the named snippet, records its location, parses it
// and prints it. Any errors detected are recorded and the snippet will not
// be displayed.
func (lc *ListCfg) displaySnippet(dir, fName, sName string) {
	content, err := os.ReadFile(fName) //nolint:gosec
	if err != nil {
		lc.errs.AddError(
			"Bad snippet",
			fmt.Errorf("snippet %q: %w", sName, err))

		return
	}

	if lc.snippetIsEclipsed(sName, dir) {
		return
	}

	lc.recordSnippetContentHash(content, fName)

	s, err := parseSnippet(content, fName, sName)
	if err != nil {
		lc.errs.AddError("Bad snippet", err)
		return
	}

	lc.recordExpectedBy(s, sName)

	text := lc.snippetToString(s)
	if text != "" {
		lc.printIntroOnce()
		fmt.Fprint(lc.StdW(), text)
	}
}

// printIntroOnce prints the intro on the ListCfg writer and sets it to
// "". The next call with the same string will have no effect.
func (lc *ListCfg) printIntroOnce() {
	if lc.intro == "" {
		return
	}

	fmt.Fprint(lc.StdW(), lc.intro)

	lc.intro = ""
}

// display reports the file if it is a regular file, descends into the sub
// directory if it is a directory and reports it as a problem otherwise
func (lc *ListCfg) display(dir, subDir string, de fs.DirEntry, ck constraintCk) {
	sName := de.Name()
	if subDir != "" {
		sName = filepath.Join(subDir, sName)
	}

	fName := filepath.Join(dir, sName)

	if de.Type().IsRegular() ||
		de.Type()&os.ModeSymlink == os.ModeSymlink {
		if ck == checkConstraints &&
			!lc.specificFileMatch(sName) {
			return
		}

		lc.displaySnippet(dir, fName, sName)
	} else if de.IsDir() {
		if ck == checkConstraints {
			if !lc.specificDirMatch(sName) {
				return
			}

			if lc.constraints[sName] {
				ck = dontCheckConstraints // turn off subsequent checking
			}
		}

		lc.descend(dir, sName, ck)
	} else {
		lc.errs.AddError("Unexpected file type",
			fmt.Errorf("%q: %s", fName, de.Type()))
	}
}

// descend displays the contents of the sub directory
func (lc *ListCfg) descend(dir, subDir string, ck constraintCk) {
	name := filepath.Join(dir, subDir)

	dirEntries, err := os.ReadDir(name)
	if err != nil {
		lc.errs.AddError(fmt.Sprintf("Bad sub-directory: %q", subDir), err)
		return
	}

	for _, de := range dirEntries {
		lc.display(dir, subDir, de, ck)
	}
}

// specificFileMatch returns true if either there are no specific snippets to
// be matched or there is a match for the snippet name directly.
func (lc *ListCfg) specificFileMatch(sName string) bool {
	if len(lc.constraints) == 0 {
		return true
	}

	if lc.constraints[sName] {
		return true
	}

	return false
}

// specificDirMatch returns true if:
//
// - there are no specific snippets to be matched
//
// - there is a match for the snippet name directly
//
// - either the subDir name or some leading part is in the Specific map.
func (lc *ListCfg) specificDirMatch(subDir string) bool {
	if len(lc.constraints) == 0 {
		return true
	}

	if lc.constraints[subDir] {
		return true
	}

	for k := range lc.constraints {
		if strings.HasPrefix(k, subDir+"/") {
			return true
		}
	}

	return false
}

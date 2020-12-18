package snippet

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	CommentStr = "snippet:"
	NoteStr    = "Note:"
	ImportStr  = "Import:"
	ExpectStr  = "Expect:"
	AfterStr   = "ComesAfter:"
	TagStr     = "Tag:"

	commentREStr = `^\s*//\s*` + CommentStr
	noteREStr    = commentREStr + `\s*` + NoteStr + `\s*`
	importREStr  = commentREStr + `\s*` + ImportStr + `\s*`
	expectREStr  = commentREStr + `\s*` + ExpectStr + `\s*`
	afterREStr   = commentREStr + `\s*` + AfterStr + `\s*`
	tagREStr     = commentREStr + `\s*` + TagStr + `\s*`
)

var (
	commentRE = regexp.MustCompile(commentREStr)
	noteRE    = regexp.MustCompile(noteREStr)
	importRE  = regexp.MustCompile(importREStr)
	expectRE  = regexp.MustCompile(expectREStr)
	afterRE   = regexp.MustCompile(afterREStr)
	tagRE     = regexp.MustCompile(tagREStr)
)

// S records the details of the snippet
type S struct {
	name    string
	path    string
	text    []string
	docs    []string
	expects []string
	imports []string
	follows []string
	tags    map[string][]string
}

// Matches returns an error if the two snippets differ, nil otherwise
func (s S) Matches(other S) error {
	if s.name != other.name {
		return fmt.Errorf("the names differ: this: %q, other: %q",
			s.name, other.name)
	}
	if s.path != other.path {
		return fmt.Errorf("the paths differ: this: %q, other: %q",
			s.path, other.path)
	}
	if err := cmpSlice("docs", s.docs, other.docs); err != nil {
		return err
	}
	if err := cmpSlice("expects", s.expects, other.expects); err != nil {
		return err
	}
	if err := cmpSlice("imports", s.imports, other.imports); err != nil {
		return err
	}
	if err := cmpSlice("follows", s.follows, other.follows); err != nil {
		return err
	}
	if err := cmpTags(s.tags, other.tags); err != nil {
		return err
	}
	return nil
}

// cmpTags returns an error if the two tag maps are different, nil otherwise
func cmpTags(a, b map[string][]string) error {
	differingTags := []string{}
	for k := range a {
		if _, ok := b[k]; !ok {
			differingTags = append(differingTags,
				fmt.Sprintf("%q in this, not in other", k))
		}
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			differingTags = append(differingTags,
				fmt.Sprintf("%q in other, not in this", k))
		}
	}
	if len(differingTags) > 0 {
		sort.Strings(differingTags)
		return fmt.Errorf("the tag names differ:\n\t%s",
			strings.Join(differingTags, "\n\t"))
	}

	for tag, vals := range a {
		err := cmpSlice(tag, vals, b[tag])
		if err != nil {
			return err
		}
	}
	return nil
}

// cmpSlice returns an error if the two slices are different, nil otherwise.
func cmpSlice(name string, a, b []string) error {
	diffs := []string{}
	if len(a) != len(b) {
		diffs = append(diffs,
			fmt.Sprintf("the lengths differ: %d != %d", len(a), len(b)))
	}
	maxBIdx := len(b) - 1
	var diffCount int
	var i int
	var s string
	for i, s = range a {
		if i <= maxBIdx {
			if s != b[i] {
				if diffCount == 0 {
					diffs = append(diffs,
						fmt.Sprintf("entry[%d] differs: %q != %q", i, s, b[i]))
				}
				diffCount++
			}
		}
	}
	if diffCount == 2 {
		diffs = append(diffs, "an additional difference was found")
	} else if diffCount > 2 {
		diffs = append(diffs,
			fmt.Sprintf("%d additional differences were found", diffCount-1))
	}
	if len(diffs) > 0 {
		return fmt.Errorf("%s differs:\n\t%s",
			name, strings.Join(diffs, "\n\t"))
	}
	return nil
}

// Name returns the snippet name.
func (s S) Name() string {
	return s.name
}

// Path returns the pathname of the file containing the snippet.
func (s S) Path() string {
	return s.path
}

// Text returns the text of the snippet - every line not starting with the
// snippet comment (// snippet:).
func (s S) Text() []string {
	rval := make([]string, len(s.text))
	copy(rval, s.text)
	return rval
}

// Docs returns the documentary notes for the snippet.
func (s S) Docs() []string {
	rval := make([]string, len(s.docs))
	copy(rval, s.docs)
	return rval
}

// Expects returns the list of other snippets that are expected to be used if
// this snippet is used.
func (s S) Expects() []string {
	rval := make([]string, len(s.expects))
	copy(rval, s.expects)
	return rval
}

// Imports returns the list of packages that are expected to be imported if
// this snippet is used.
func (s S) Imports() []string {
	rval := make([]string, len(s.imports))
	copy(rval, s.imports)
	return rval
}

// Follows returns the list of other snippets that this snippet should
// come after in any code that uses it.
func (s S) Follows() []string {
	rval := make([]string, len(s.follows))
	copy(rval, s.follows)
	return rval
}

// Tags returns the tags of the snippet - those comments marked as tags. Any
// tag text will be split around the first ':' and the first part will be
// used as a label for the second part.
func (s S) Tags() map[string][]string {
	rval := map[string][]string{}
	for k, v := range s.tags {
		c := make([]string, len(v))
		copy(c, v)
		rval[k] = c
	}
	return rval
}

// String returns a string representation of the snippet
func (s S) String() string {
	rval := s.name + "\n"

	maxKeyLen := 0

	parts := []struct {
		intro   string
		entries []string
	}{
		{intro: "", entries: s.docs},
		{intro: "Imports", entries: s.imports},
		{intro: "Expect", entries: s.expects},
		{intro: "Must Follow", entries: s.follows},
	}
	for _, p := range parts {
		if len(p.intro) > maxKeyLen {
			maxKeyLen = len(p.intro)
		}
	}

	var tagKeys []string
	for k := range s.tags {
		tagKeys = append(tagKeys, k)
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}
	sort.Strings(tagKeys)

	for _, p := range parts {
		intro := fmt.Sprintf("%*s: ", maxKeyLen, p.intro)
		rval += formatSlice(intro, p.entries)
	}
	for _, k := range tagKeys {
		intro := fmt.Sprintf("%*s: ", maxKeyLen, k)
		rval += formatSlice(intro, s.tags[k])
	}
	return rval
}

// formatSlice will return the content concatenated together on separate
// lines with the first line prefixed by the intro
func formatSlice(intro string, content []string) string {
	if len(content) == 0 {
		return ""
	}
	rval := ""
	blanks := strings.Repeat(" ", len(intro))
	for _, l := range content {
		rval += "\t\t" + intro + l + "\n"
		intro = blanks
	}
	return rval
}

// readSnippetFile will open and read the contents of a snippet file and
// return the contents together with the full pathname of the file it was
// read from. If the snippet file cannot be found in any of the snippet
// directories or the absolute pathname cannot be opened an error is
// returned.
func readSnippetFile(dirs []string, sName string) ([]byte, string, error) {
	if filepath.IsAbs(sName) {
		content, err := ioutil.ReadFile(sName)
		return content, sName, err
	}

	if len(dirs) == 0 {
		return nil, "", errors.New("there are no snippet directories to search")
	}

	for _, dir := range dirs {
		fName := filepath.Join(dir, sName)
		content, err := ioutil.ReadFile(fName)
		if err == nil {
			return content, fName, nil
		}
	}

	if len(dirs) == 1 {
		return nil, "",
			fmt.Errorf("snippet %q is not in the snippet directory: %q",
				sName, dirs[0])
	}
	return nil, "",
		fmt.Errorf("snippet %q is not in any snippet directory: \"%s\"",
			sName, strings.Join(dirs, `", "`))
}

// parseSnippet will construct the snippet from the content.
func parseSnippet(content []byte, fName, sName string) (*S, error) {
	s := &S{
		name: sName,
		path: fName,
		tags: map[string][]string{},
	}

	buf := bytes.NewBuffer(content)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		l := scanner.Text()
		if commentRE.FindStringIndex(l) != nil {
			if addMatchToSlices(l, importRE, &s.imports) {
				continue
			}
			if addMatchToSlices(l, expectRE, &s.expects) {
				continue
			}
			if addMatchToSlices(l, afterRE, &s.expects, &s.follows) {
				continue
			}
			if addWholeMatchToSlice(l, noteRE, &s.docs) {
				continue
			}
			if s.addTag(l) {
				continue
			}
		} else {
			s.text = append(s.text, l)
		}
	}

	s.tidy()

	if len(s.text) == 0 &&
		len(s.imports) == 0 {
		return nil,
			fmt.Errorf("snippet %q (%s) has no text and no imports",
				sName, fName)
	}

	return s, nil
}

// tidy sorts and removes duplicates from the imports, expects and
// follows slices. It also removes any empty entries.
func (s *S) tidy() {
	s.imports = tidySlice(s.imports)
	s.expects = tidySlice(s.expects)
	s.follows = tidySlice(s.follows)
}

// tidySlice sorts the slice, removes any blank or duplicate entries and
// returns it.
func tidySlice(s []string) []string {
	sort.Strings(s)
	i := 0
	last := ""
	for _, str := range s {
		if str != "" && str != last {
			s[i] = str
			i++
			last = str
		}
	}
	return s[:i]
}

// addTag will look for the snippet documentation tag in the line and if it
// finds one it will parse out the tag name and value and add it to the
// snippet tags map.
func (s *S) addTag(line string) bool {
	loc := tagRE.FindStringIndex(line)
	if loc == nil {
		return false
	}

	text := strings.TrimSpace(line[loc[1]:])
	parts := strings.SplitN(text, ":", 2)
	var tag, value string
	tag = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		value = strings.TrimSpace(parts[1])
	}
	s.tags[tag] = append(s.tags[tag], value)
	return true
}

// addMatchToSlices tests the string for a match against the regexp. If it
// matches then the remainder of the string after the matched portion is
// trimmed of white space. If the resulting string is non-empty it is added
// to the slices. It returns true if the string matched the regex and false
// otherwise.
func addMatchToSlices(s string, re *regexp.Regexp, slcs ...*[]string) bool {
	loc := re.FindStringIndex(s)
	if loc == nil {
		return false
	}
	text := strings.TrimSpace(s[loc[1]:])
	if len(text) > 0 {
		for _, slc := range slcs {
			*slc = append(*slc, text)
		}
	}
	return true
}

// addWholeMatchToSlice behaves as per addMatchToSlices but doesn't trim
// the line or ignore empty lines.
func addWholeMatchToSlice(s string, re *regexp.Regexp, slc *[]string) bool {
	if loc := re.FindStringIndex(s); loc != nil {
		text := s[loc[1]:]
		*slc = append(*slc, text)
		return true
	}
	return false
}

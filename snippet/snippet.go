package snippet

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// These are named parts of the snippet for listing
const (
	NamePart = "name"
	PathPart = "path"
	TextPart = "text"

	DocsPart   = "note"
	ImportPart = "imports"
	ExpectPart = "expects"
	FollowPart = "follows"
	TagPart    = "tag"
)

// These correspond to semantic comments in the snippet
const (
	CommentStr = "snippet:"
	NoteStr    = DocsPart + ":"
	ImportStr  = ImportPart + ":"
	ExpectStr  = ExpectPart + ":"
	AfterStr   = FollowPart + ":"
	TagStr     = TagPart + ":"
)

// A regexp matching a snippet comment. Note that this is case-blind because
// of the leading "(?i)"
const commentREStr = `^(?i)\s*//\s*` + CommentStr

var snippetParts = []string{
	DocsPart,
	ImportPart,
	ExpectPart,
	FollowPart,
	TagPart,
}

var altPartNames = map[string][]string{
	DocsPart:   {"notes", "doc", "docs"},
	ImportPart: {"import"},
	ExpectPart: {"expect", "comesbefore"},
	FollowPart: {"follow", "comesafter"},
	TagPart:    {"tags"},
}

// AltPartNames returns a slice of alternative names for the given part. Note
// that the slice may be empty.
func AltPartNames(part string) []string {
	return altPartNames[part]
}

var validParts = map[string]string{
	NamePart:   "the snippet name",
	PathPart:   "the name of the snippet file",
	TextPart:   "the snippet code to be used",
	DocsPart:   "how the snippet should be used",
	ExpectPart: "snippets used with this",
	ImportPart: "packages this snippet imports",
	FollowPart: "snippets coming before this",
	TagPart:    "colon-separated name/value pairs",
}

// ValidParts returns a map which has an entry for all the valid parts of a
// snippet with a brief description of the part and how it is used.
func ValidParts() map[string]string {
	rval := make(map[string]string)

	maps.Copy(rval, validParts)

	return rval
}

var commentRE = regexp.MustCompile(commentREStr)

var snippetPartREs = map[string]*regexp.Regexp{}

// altNames returns a fragment of a regular expression which represents the
// allowed alternative names of the snippet part for the given named part. If
// a snippet part has no alternative names an empty string is returned.
func altNames(name string) string {
	alt := ""

	if altNames, ok := altPartNames[name]; ok && len(altNames) != 0 {
		alt = "|" + strings.Join(altNames, "|")
	}

	return alt
}

// init - this populates the map of named snippet parts to the corresponding
// regular expression.
func init() {
	for _, partName := range snippetParts {
		reStr := commentREStr +
			`\s*` + `(?:` + partName + altNames(partName) + `):\s*`
		snippetPartREs[partName] = regexp.MustCompile(reStr)
	}
}

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

	return cmpTags(s.tags, other.tags)
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
		err := cmpSlice("Tag:"+tag, vals, b[tag])
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

	const maxDiffsShown = 1

	for i, s := range a {
		if i <= maxBIdx {
			if s != b[i] {
				if diffCount < maxDiffsShown {
					diffs = append(diffs,
						fmt.Sprintf("entry[%d] differs: %q != %q", i, s, b[i]))
				}

				diffCount++
			}
		}
	}

	if diffCount == maxDiffsShown+1 {
		diffs = append(diffs, "an additional difference was found")
	} else if diffCount > maxDiffsShown+1 {
		diffs = append(diffs,
			fmt.Sprintf("%d additional differences were found",
				diffCount-maxDiffsShown))
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
	fc := formatCfg{}
	return fc.snippetToString(&s)
}

// readSnippetFile will open and read the contents of a snippet file and
// return the contents together with the full pathname of the file it was
// read from. If the snippet file cannot be found in any of the snippet
// directories or the absolute pathname cannot be opened an error is
// returned.
func readSnippetFile(dirs []string, sName string) ([]byte, string, error) {
	if filepath.IsAbs(sName) {
		content, err := os.ReadFile(sName) //nolint:gosec
		return content, sName, err
	}

	if len(dirs) == 0 {
		return nil, "", errors.New("there are no snippet directories to search")
	}

	for _, dir := range dirs {
		fName := filepath.Join(dir, sName)

		content, err := os.ReadFile(fName) //nolint:gosec
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

	scanner := bufio.NewScanner(bytes.NewBuffer(content))
	for scanner.Scan() {
		l := scanner.Text()
		if commentRE.FindStringIndex(l) != nil {
			if addMatchToSlices(l, snippetPartREs[ImportPart], &s.imports) {
				continue
			}

			if addMatchToSlices(l, snippetPartREs[ExpectPart], &s.expects) {
				continue
			}

			if addMatchToSlices(l, snippetPartREs[FollowPart],
				&s.expects, &s.follows) {
				continue
			}

			if addWholeMatchToSlice(l, snippetPartREs[DocsPart], &s.docs) {
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
	loc := snippetPartREs[TagPart].FindStringIndex(line)
	if loc == nil {
		return false
	}

	text := strings.TrimSpace(line[loc[1]:])

	tag, value, hasVal := strings.Cut(text, ":")
	if hasVal {
		value = strings.TrimSpace(value)
	}

	tag = strings.TrimSpace(tag)
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

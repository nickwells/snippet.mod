package snippet

import (
	"fmt"
	"sort"
	"strings"
)

const (
	nameIndent = 4
	dfltIndent = 8
)

// formatCfg holds the configuration values controlling how we generate a
// string reflecting a snippet value
type formatCfg struct {
	// parts records the parts of the snippet to be shown. If empty then all
	// parts are shown.
	parts map[string]bool
	// tags records the tag parts of the snippet to be shown. If empty
	// then all tags are shown.
	tags map[string]bool

	// hideIntro controls whether introductory strings are printed before the
	// parts of the snippet
	hideIntro bool
}

type partsToShow struct {
	intro  string
	indent int
	values []string
}

// initPartsToShow constructs the list of parts to show and returns it
func (fc *formatCfg) initPartsToShow(s *S) []partsToShow { //nolint:cyclop
	parts := []partsToShow{}

	partsAndTagsEmpty := len(fc.parts) == 0 && len(fc.tags) == 0

	if partsAndTagsEmpty || fc.parts[NamePart] {
		indent := nameIndent
		parts = append(parts,
			partsToShow{
				intro:  "",
				indent: indent,
				values: []string{s.name},
			})
	}
	if fc.parts[PathPart] {
		parts = append(parts,
			partsToShow{
				intro:  "Pathname:",
				values: []string{s.path},
			})
	}
	if partsAndTagsEmpty || fc.parts[DocsPart] {
		parts = append(parts,
			partsToShow{
				intro:  "Note:",
				values: s.docs,
			})
	}
	if partsAndTagsEmpty || fc.parts[ImportPart] {
		parts = append(parts,
			partsToShow{
				intro:  "Imports:",
				values: s.imports,
			})
	}
	if partsAndTagsEmpty || fc.parts[FollowPart] {
		parts = append(parts,
			partsToShow{
				intro:  "Follows:",
				values: s.follows,
			})
	}
	if partsAndTagsEmpty || fc.parts[ExpectPart] {
		expectedParts := make([]string, 0, len(s.expects))
		for _, e := range s.expects {
			addName := true
			for _, f := range s.follows {
				if e == f {
					addName = false
					break
				}
			}
			if addName {
				expectedParts = append(expectedParts, e)
			}
		}
		parts = append(parts,
			partsToShow{
				intro:  "Expects:",
				values: expectedParts,
			})
	}

	tagKeys := getTagKeys(s)

	if fc.parts[TagPart] {
		parts = append(parts,
			partsToShow{
				intro:  "Tags:",
				values: tagKeys,
			})
	}

	for _, k := range tagKeys {
		if partsAndTagsEmpty || fc.tags[k] {
			parts = append(parts,
				partsToShow{
					intro:  k + ":",
					values: s.tags[k],
				})
		}
	}

	if fc.parts[TextPart] {
		parts = append(parts,
			partsToShow{
				intro:  "Text:",
				values: s.text,
			})
	}

	return parts
}

// getTagKeys returns a sorted list of tag names
func getTagKeys(s *S) []string {
	var tagKeys []string
	for k := range s.tags {
		tagKeys = append(tagKeys, k)
	}
	sort.Strings(tagKeys)
	return tagKeys
}

// maxIntroLen returns the length of the longest intro.
func maxIntroLen(parts []partsToShow) int {
	maxIntroLen := 0
	for _, p := range parts {
		if len(p.intro) > maxIntroLen {
			maxIntroLen = len(p.intro)
		}
	}
	return maxIntroLen
}

// snippetToString returns a string showing the Snippet formatted according
// to the formatCfg
func (fc *formatCfg) snippetToString(s *S) string {
	parts := fc.initPartsToShow(s)
	rval := "\n"

	if fc.hideIntro {
		for _, p := range parts {
			for _, l := range p.values {
				rval += l + "\n"
			}
		}
		return rval
	}

	maxLen := maxIntroLen(parts)
	for _, p := range parts {
		var intro, blanks string
		if p.intro != "" {
			intro = fmt.Sprintf("%*s ", maxLen, p.intro)
		}
		indent := p.indent
		if indent == 0 {
			indent = dfltIndent
		}
		intro = strings.Repeat(" ", indent) + intro
		blanks = strings.Repeat(" ", len(intro))

		for _, l := range p.values {
			rval += intro + l + "\n"
			intro = blanks
		}
	}

	return rval
}

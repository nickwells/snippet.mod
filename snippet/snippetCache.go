package snippet

import (
	"fmt"

	"github.com/nickwells/errutil.mod/errutil"
)

// Cache holds a collection of snippets by name
type Cache map[string]*S

// Add will check that the snippet is not already in the cache and if not it
// will search for the snippet file in the snippetDirs, parse the file and
// generate a snippet which it will then store in the cache. It returns the
// snippet and any error; if the error is non-nil the snippet will be nil.
func (c *Cache) Add(snippetDirs []string, sName string) (*S, error) {
	s, ok := (*c)[sName]
	if ok {
		return s, nil
	}

	content, fName, err := readSnippetFile(snippetDirs, sName)
	if err != nil {
		return nil, err
	}

	s, err = parseSnippet(content, fName, sName)
	if err != nil {
		return nil, err
	}

	(*c)[sName] = s

	return s, nil
}

// Get will retrieve the named snippet from the cache, returning an error if
// it is not present.
func (c Cache) Get(sName string) (*S, error) {
	s, ok := c[sName]
	if !ok {
		return nil, fmt.Errorf("%q is not in the snippet cache", sName)
	}

	return s, nil
}

// Check will check that all the snippets in the Cache have all their
// expected snippets also in the cache
func (c Cache) Check(em *errutil.ErrMap) {
	for sName, s := range c {
		for _, expected := range s.expects {
			_, ok := c[expected]
			if !ok {
				em.AddError(
					fmt.Sprintf("Missing snippet %q", expected),
					fmt.Errorf("expected by %q", sName))
			}
		}
	}
}

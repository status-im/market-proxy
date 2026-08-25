package fetcher_by_id

import "errors"

// ErrNotFound is returned when the cache holds no item for the requested id.
//
// It exists so callers can branch on the condition with errors.Is instead of
// matching the message text, which silently reclassifies any unrelated error
// whose wording happens to contain the same words.
var ErrNotFound = errors.New("item not found")

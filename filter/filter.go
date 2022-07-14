package filter

import "fmt"

var (
	ErrNoResults        = fmt.Errorf("given filter does not match any item")
	ErrAmbiguousResults = fmt.Errorf("given filter applies to more than one result")
)

type Filter[T any] interface {
	AppliesTo(T) bool
}

func Find[T any, F Filter[T]](filter F, items []T) []T {
	var result []T
	for _, item := range items {
		if filter.AppliesTo(item) {
			result = append(result, item)
		}
	}
	return result
}

func FindOne[T any, F Filter[T]](filter F, items []T) (v T, err error) {
	filtered := Find[T, F](filter, items)

	if len(filtered) == 0 {
		return v, ErrNoResults
	}

	if len(filtered) > 1 {
		return v, ErrAmbiguousResults
	}

	return filtered[0], nil
}

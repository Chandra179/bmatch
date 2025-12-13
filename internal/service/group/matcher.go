package group

import (
	"context"
)

// GroupMatcher defines the interface for finding matching groups
type GroupMatcher interface {
	FindMatches(ctx context.Context, userProfile UserProfile, filters DiscoverGroupsRequest) ([]GroupMatch, error)
}

// PostgresMatcher implements GroupMatcher using PostgreSQL with GIN indexes
type PostgresMatcher struct {
	repo Repository
}

func NewPostgresMatcher(repo Repository) *PostgresMatcher {
	return &PostgresMatcher{
		repo: repo,
	}
}

// FindMatches finds groups matching user profile and calculates Jaccard similarity
func (m *PostgresMatcher) FindMatches(ctx context.Context, userProfile UserProfile, filters DiscoverGroupsRequest) ([]GroupMatch, error) {
	// Get candidate groups from database using GIN index
	groups, err := m.repo.FindGroupsByTags(ctx, userProfile.Tags, filters)
	if err != nil {
		return nil, err
	}

	// Calculate Jaccard similarity for each group
	matches := make([]GroupMatch, 0, len(groups))
	for _, group := range groups {
		score := CalculateJaccardScore(userProfile.Tags, group.Tags)
		matches = append(matches, GroupMatch{
			Group:           group,
			SimilarityScore: score,
		})
	}

	// Sort by similarity score (descending)
	sortByScore(matches)

	return matches, nil
}

// CalculateJaccardScore calculates the Jaccard similarity index
// J(A,B) = |A ∩ B| / |A ∪ B|
func CalculateJaccardScore(userTags, groupTags []string) float64 {
	if len(userTags) == 0 && len(groupTags) == 0 {
		return 0.0
	}

	intersection := intersect(userTags, groupTags)
	union := union(userTags, groupTags)

	if len(union) == 0 {
		return 0.0
	}

	return float64(len(intersection)) / float64(len(union))
}

// intersect returns the intersection of two string slices
func intersect(a, b []string) []string {
	set := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range a {
		set[item] = true
	}

	for _, item := range b {
		if set[item] {
			result = append(result, item)
			delete(set, item) // Prevent duplicates
		}
	}

	return result
}

// union returns the union of two string slices
func union(a, b []string) []string {
	set := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range a {
		if !set[item] {
			set[item] = true
			result = append(result, item)
		}
	}

	for _, item := range b {
		if !set[item] {
			set[item] = true
			result = append(result, item)
		}
	}

	return result
}

// sortByScore sorts matches by similarity score in descending order
func sortByScore(matches []GroupMatch) {
	// Simple bubble sort for small datasets
	// Can be replaced with sort.Slice for larger datasets
	n := len(matches)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if matches[j].SimilarityScore < matches[j+1].SimilarityScore {
				matches[j], matches[j+1] = matches[j+1], matches[j]
			}
		}
	}
}
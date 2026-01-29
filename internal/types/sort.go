package types

type SortBy string

const (
	SortByRelevance SortBy = "relevance"
	SortByDate      SortBy = "date"
	SortByViews     SortBy = "views"
	SortByRating    SortBy = "rating"
)

func (s SortBy) GetSPParam() string {
	switch s {
	case SortByRelevance:
		return ""
	case SortByDate:
		return "CAI%253D"
	case SortByViews:
		return "CAM%253D"
	case SortByRating:
		return "CAE%253D"
	default:
		return ""
	}
}

func (s SortBy) GetDisplayName() string {
	switch s {
	case SortByRelevance:
		return "Relevance"
	case SortByDate:
		return "Date"
	case SortByViews:
		return "Views"
	case SortByRating:
		return "Rating"
	default:
		return ""
	}
}

func (s SortBy) Next() SortBy {
	switch s {
	case SortByRelevance:
		return SortByDate
	case SortByDate:
		return SortByViews
	case SortByViews:
		return SortByRating
	case SortByRating:
		return SortByRelevance
	default:
		return SortByRelevance
	}
}

// ParseSortBy converts a string to SortBy type
func ParseSortBy(s string) SortBy {
	switch s {
	case "date":
		return SortByDate
	case "views":
		return SortByViews
	case "rating":
		return SortByRating
	case "relevance":
		return SortByRelevance
	default:
		return SortByRelevance
	}
}

func (s SortBy) Prev() SortBy {
	switch s {
	case SortByRelevance:
		return SortByRating
	case SortByDate:
		return SortByRelevance
	case SortByViews:
		return SortByDate
	case SortByRating:
		return SortByViews
	default:
		return SortByRelevance
	}
}

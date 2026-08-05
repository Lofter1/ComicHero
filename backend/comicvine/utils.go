package comicvine

import (
	"fmt"
	"strconv"
	"strings"
)

// IDParser provides utilities for parsing Comic Vine IDs
type IDParser struct{}

// ParseID extracts the numeric ID from a Comic Vine API URL or ID string
// Examples:
// - "4000-1443" -> 1443
// - "1443" -> 1443
// - "https://comicvine.gamespot.com/api/character/4000-1443/" -> 1443
func ParseID(id string) (int, error) {
	// Remove URL prefix if present
	if strings.Contains(id, "/") {
		parts := strings.Split(strings.Trim(id, "/"), "/")
		id = parts[len(parts)-1]
	}

	// Handle "4000-XXXX" format
	if strings.Contains(id, "-") {
		parts := strings.Split(id, "-")
		if len(parts) == 2 {
			id = parts[1]
		}
	}

	return strconv.Atoi(id)
}

// FormatID formats a numeric ID into Comic Vine's "<prefix>-<id>" format for the
// given resource type (e.g. FormatID(ResourceVolume, 12345) -> "4050-12345").
func FormatID(resourceType ResourceType, id int) string {
	return fmt.Sprintf("%s-%d", idPrefixFor(resourceType), id)
}

// URLBuilder helps construct Comic Vine URLs
type URLBuilder struct {
	BaseURL string
}

// NewURLBuilder creates a new URLBuilder with the default Comic Vine API URL
func NewURLBuilder() *URLBuilder {
	return &URLBuilder{
		BaseURL: defaultBaseURL,
	}
}

// ResourceURL builds a URL for a specific resource
func (b *URLBuilder) ResourceURL(resourceType ResourceType, id int) string {
	return fmt.Sprintf("%s/%s/%s/", b.BaseURL, resourceType, FormatID(resourceType, id))
}

// SearchURL builds a search URL
func (b *URLBuilder) SearchURL(query string) string {
	return fmt.Sprintf("%s/search/?query=%s&format=json", b.BaseURL, query)
}

// ImageURLS represents different size variants of a Comic Vine image
type ImageURLS struct {
	Icon        string
	Medium      string
	Screen      string
	ScreenLarge string
	Small       string
	Super       string
	Thumb       string
	Tiny        string
	Original    string
}

// GetImageURLs returns all available image URLs for a Comic Vine image
func GetImageURLs(img Image) ImageURLS {
	return ImageURLS{
		Icon:        img.IconURL,
		Medium:      img.MediumURL,
		Screen:      img.ScreenURL,
		ScreenLarge: img.ScreenLargeURL,
		Small:       img.SmallURL,
		Super:       img.SuperURL,
		Thumb:       img.ThumbURL,
		Tiny:        img.TinyURL,
		Original:    img.OriginalURL,
	}
}

// GetBestImage returns the best available image URL
func GetBestImage(img Image) string {
	if img.SuperURL != "" {
		return img.SuperURL
	}
	if img.ScreenLargeURL != "" {
		return img.ScreenLargeURL
	}
	if img.MediumURL != "" {
		return img.MediumURL
	}
	if img.SmallURL != "" {
		return img.SmallURL
	}
	if img.ThumbURL != "" {
		return img.ThumbURL
	}
	return img.IconURL
}

// Gender represents character gender
type Gender int

const (
	GenderUnknown Gender = 0
	GenderMale    Gender = 1
	GenderFemale  Gender = 2
	GenderOther   Gender = 3
)

// String returns the string representation of a Gender
func (g Gender) String() string {
	switch g {
	case GenderMale:
		return "Male"
	case GenderFemale:
		return "Female"
	case GenderOther:
		return "Other"
	default:
		return "Unknown"
	}
}

// FormatComicVineDate parses a Comic Vine date string
func FormatComicVineDate(dateStr string) string {
	// Comic Vine dates are typically in "YYYY-MM-DD" format
	// This is a placeholder for any date formatting needs
	return dateStr
}

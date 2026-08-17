// Package canvas defines coordinate, region, and anchor value objects.
package canvas

import "errors"

var (
	ErrInvalidCoordinate = errors.New("coordinate outside board bounds")
	ErrInvalidRegion     = errors.New("region outside board bounds or non-positive size")
)

// Coordinate is a point in canvas pixel space.
type Coordinate struct {
	X float64
	Y float64
}

// Region is an axis-aligned rectangle in canvas pixel space.
type Region struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// Validate checks coordinate lies within the given board size.
func (c Coordinate) Validate(w, h int) error {
	if c.X < 0 || c.Y < 0 || c.X > float64(w) || c.Y > float64(h) {
		return ErrInvalidCoordinate
	}
	return nil
}

// Validate checks region lies within the board size and has positive width/height.
func (r Region) Validate(w, h int) error {
	if r.Width <= 0 || r.Height <= 0 {
		return ErrInvalidRegion
	}
	if r.X < 0 || r.Y < 0 {
		return ErrInvalidRegion
	}
	if r.X+r.Width > float64(w) || r.Y+r.Height > float64(h) {
		return ErrInvalidRegion
	}
	return nil
}

// Anchor is an attachment point for an annotation on a version.
type Anchor struct {
	VersionID string
	Region    *Region
	Point     *Coordinate
}

func NewPointAnchor(versionID string, pt Coordinate) Anchor {
	return Anchor{VersionID: versionID, Point: &pt}
}

func NewRegionAnchor(versionID string, r Region) Anchor {
	return Anchor{VersionID: versionID, Region: &r}
}

// Type describes whether the anchor is point-based or region-based.
type AnchorType string

const (
	AnchorTypePoint  AnchorType = "point"
	AnchorTypeRegion AnchorType = "region"
)

func (a Anchor) Type() AnchorType {
	if a.Region != nil {
		return AnchorTypeRegion
	}
	return AnchorTypePoint
}

// MigratedAnchor records that an anchor has been re-located to a new version.
type MigratedAnchor struct {
	FromVersionID string
	ToVersionID   string
	OldPoint      *Coordinate
	NewPoint      *Coordinate
	OldRegion     *Region
	NewRegion     *Region
	Reason        string
	By            string
}

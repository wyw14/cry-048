package canvas

import "testing"

func TestCoordinateValidate(t *testing.T) {
	w, h := 100, 100
	cases := []struct {
		name string
		c    Coordinate
		ok   bool
	}{
		{"origin", Coordinate{X: 0, Y: 0}, true},
		{"inside", Coordinate{X: 50, Y: 50}, true},
		{"top-right corner", Coordinate{X: 100, Y: 100}, true},
		{"negative x", Coordinate{X: -1, Y: 50}, false},
		{"negative y", Coordinate{X: 50, Y: -1}, false},
		{"over x", Coordinate{X: 101, Y: 50}, false},
		{"over y", Coordinate{X: 50, Y: 101}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.c.Validate(w, h)
			if (err == nil) != c.ok {
				t.Fatalf("got err=%v want ok=%v", err, c.ok)
			}
		})
	}
}

func TestRegionValidate(t *testing.T) {
	w, h := 200, 200
	cases := []struct {
		name string
		r    Region
		ok   bool
	}{
		{"ok", Region{X: 10, Y: 10, Width: 100, Height: 100}, true},
		{"zero width", Region{X: 10, Y: 10, Width: 0, Height: 100}, false},
		{"zero height", Region{X: 10, Y: 10, Width: 100, Height: 0}, false},
		{"negative x", Region{X: -10, Y: 10, Width: 100, Height: 100}, false},
		{"extends beyond", Region{X: 150, Y: 10, Width: 100, Height: 100}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.r.Validate(w, h)
			if (err == nil) != c.ok {
				t.Fatalf("got err=%v want ok=%v", err, c.ok)
			}
		})
	}
}

func TestAnchorType(t *testing.T) {
	p := NewPointAnchor("v", Coordinate{X: 1, Y: 1})
	if p.Type() != AnchorTypePoint {
		t.Fatalf("point anchor type=%s want point", p.Type())
	}
	r := NewRegionAnchor("v", Region{X: 1, Y: 1, Width: 10, Height: 10})
	if r.Type() != AnchorTypeRegion {
		t.Fatalf("region anchor type=%s want region", r.Type())
	}
}

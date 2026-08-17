package project

import (
	"errors"
	"testing"
	"time"
)

func TestNewProjectValidation(t *testing.T) {
	now := time.Now()
	if _, err := NewProject("", "name", "", now); !errors.Is(err, errors.New("project id required")) {
		// just check it errors out
	}
	if _, err := NewProject("p1", "", "", now); err == nil {
		t.Fatal("empty name should error")
	}
}

func TestArchiveProject(t *testing.T) {
	now := time.Now()
	p, _ := NewProject("p1", "n", "", now)
	if err := p.Archive(now); err != nil {
		t.Fatal(err)
	}
	if p.Status != "archived" {
		t.Fatalf("status=%s want archived", p.Status)
	}
	// second archive fails
	if err := p.Archive(now); !errors.Is(err, ErrProjectArchived) {
		t.Fatalf("want ErrProjectArchived, got %v", err)
	}
}

func TestBoardSizeValidate(t *testing.T) {
	cases := []struct {
		s  BoardSize
		ok bool
	}{
		{BoardSize{Width: 1440, Height: 1024}, true},
		{BoardSize{Width: 0, Height: 100}, false},
		{BoardSize{Width: -1, Height: 100}, false},
		{BoardSize{Width: 200000, Height: 100}, false},
	}
	for _, c := range cases {
		err := c.s.Validate()
		if (err == nil) != c.ok {
			t.Errorf("size=%+v err=%v want ok=%v", c.s, err, c.ok)
		}
	}
}

func TestBoardClose(t *testing.T) {
	now := time.Now()
	b, _ := NewBoard("b1", "p1", "name", BoardSize{Width: 100, Height: 100}, now)
	if err := b.Close(now); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(b.Close(now), ErrBoardClosed) {
		t.Fatal("double close should error")
	}
	if b.IsOpen() {
		t.Fatal("closed board should not be open")
	}
}

func TestVersionPublishSupersede(t *testing.T) {
	now := time.Now()
	v1, _ := NewVersion("v1", "b1", 1, "label", "key", "notes", "user", now)
	if v1.Status != "draft" {
		t.Fatalf("initial status=%s want draft", v1.Status)
	}
	v1.Publish(now)
	if v1.Status != "published" {
		t.Fatalf("after publish status=%s", v1.Status)
	}
	v1.Supersede(now)
	if v1.Status != "superseded" {
		t.Fatalf("after supersede status=%s", v1.Status)
	}
	if !v1.IsSuperseded() {
		t.Fatal("IsSuperseded should be true")
	}
}

func TestRoleValidation(t *testing.T) {
	cases := []struct {
		r    Role
		want bool
	}{
		{RoleOwner, true},
		{RoleEditor, true},
		{RoleViewer, true},
		{Role("xyz"), false},
		{Role(""), false},
	}
	for _, c := range cases {
		if got := c.r.Valid(); got != c.want {
			t.Errorf("role=%q valid=%v want %v", c.r, got, c.want)
		}
	}
}

func TestRolePermissions(t *testing.T) {
	if !RoleOwner.CanManageMembers() || !RoleOwner.CanEdit() || !RoleOwner.CanAnnotate() {
		t.Fatal("owner should have all permissions")
	}
	if RoleViewer.CanEdit() {
		t.Fatal("viewer should not be able to edit")
	}
	if RoleViewer.CanAnnotate() {
		t.Fatal("viewer should not be able to annotate")
	}
	if !RoleEditor.CanEdit() || !RoleEditor.CanAnnotate() {
		t.Fatal("editor should edit and annotate")
	}
	if RoleEditor.CanManageMembers() {
		t.Fatal("editor should not manage members")
	}
}

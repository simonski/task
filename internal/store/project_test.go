package store

import "testing"

func TestCreateListAndGetProject(t *testing.T) {
	db := testDB(t)

	project, err := CreateProject(db, "Customer Portal", "Portal work", 1)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if project.Slug != "customer-portal" {
		t.Fatalf("CreateProject().Slug = %q, want customer-portal", project.Slug)
	}

	projects, err := ListProjects(db)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("ListProjects() len = %d, want 2", len(projects))
	}

	if projects[0].Slug != "default-project" {
		t.Fatalf("ListProjects()[0].Slug = %q, want default-project", projects[0].Slug)
	}

	bySlug, err := GetProject(db, "customer-portal")
	if err != nil {
		t.Fatalf("GetProject(slug) error = %v", err)
	}
	if bySlug.ID != project.ID {
		t.Fatalf("GetProject(slug).ID = %d, want %d", bySlug.ID, project.ID)
	}

	byID, err := GetProject(db, "2")
	if err != nil {
		t.Fatalf("GetProject(id) error = %v", err)
	}
	if byID.ID != project.ID {
		t.Fatalf("GetProject(id).ID = %d, want %d", byID.ID, project.ID)
	}
}

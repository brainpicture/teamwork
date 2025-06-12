package internal

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// ProjectStatus represents the status of a project
type ProjectStatus string

const (
	StatusPlanning  ProjectStatus = "planning"
	StatusActive    ProjectStatus = "active"
	StatusPaused    ProjectStatus = "paused"
	StatusCompleted ProjectStatus = "completed"
	StatusCancelled ProjectStatus = "cancelled"
)

// ProjectRole represents the role of a user in a project
type ProjectRole string

const (
	RoleOwner  ProjectRole = "owner"
	RoleAdmin  ProjectRole = "admin"
	RoleMember ProjectRole = "member"
	RoleViewer ProjectRole = "viewer"
)

// Project represents a project in the database
type Project struct {
	ID          int           `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      ProjectStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	UserRole    ProjectRole   `json:"user_role,omitempty"` // Role of current user in this project
}

// ProjectUser represents a user's membership in a project
type ProjectUser struct {
	ID        int         `json:"id"`
	ProjectID int         `json:"project_id"`
	UserID    int         `json:"user_id"`
	Role      ProjectRole `json:"role"`
	JoinedAt  time.Time   `json:"joined_at"`
}

// CreateProject creates a new project and assigns the creator as owner
func (db *DB) CreateProject(creatorUserID int, title, description string) (*Project, error) {
	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// Create project
	query := `
		INSERT INTO projects (title, description) 
		VALUES (?, ?)
	`

	result, err := tx.Exec(query, title, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %v", err)
	}

	projectID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get project ID: %v", err)
	}

	// Add creator as owner
	_, err = tx.Exec(
		"INSERT INTO project_users (project_id, user_id, role) VALUES (?, ?, ?)",
		projectID, creatorUserID, RoleOwner,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add project owner: %v", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Set this project as the user's current project
	if err = db.SetUserCurrentProject(creatorUserID, int(projectID)); err != nil {
		// Log error but don't fail the creation
		log.Printf("Warning: failed to set current project for user %d: %v", creatorUserID, err)
	}

	return db.GetProjectByIDForUser(int(projectID), creatorUserID)
}

// GetProjectByIDForUser retrieves a project by its ID with user's role
func (db *DB) GetProjectByIDForUser(projectID, userID int) (*Project, error) {
	query := `
		SELECT p.id, p.title, p.description, p.status, 
		       p.created_at, p.updated_at, pu.role
		FROM projects p
		JOIN project_users pu ON p.id = pu.project_id
		WHERE p.id = ? AND pu.user_id = ?
	`

	project := &Project{}

	err := db.QueryRow(query, projectID, userID).Scan(
		&project.ID, &project.Title, &project.Description,
		&project.Status, &project.CreatedAt, &project.UpdatedAt,
		&project.UserRole,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %v", err)
	}

	return project, nil
}

// GetUserProjects retrieves all projects for a specific user
func (db *DB) GetUserProjects(userID int) ([]*Project, error) {
	query := `
		SELECT p.id, p.title, p.description, p.status, 
		       p.created_at, p.updated_at, pu.role
		FROM projects p
		JOIN project_users pu ON p.id = pu.project_id
		WHERE pu.user_id = ? 
		ORDER BY p.created_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user projects: %v", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		project := &Project{}

		err := rows.Scan(
			&project.ID, &project.Title, &project.Description,
			&project.Status, &project.CreatedAt, &project.UpdatedAt,
			&project.UserRole,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %v", err)
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// GetUserProjectsByStatus retrieves projects for a user filtered by status
func (db *DB) GetUserProjectsByStatus(userID int, status ProjectStatus) ([]*Project, error) {
	query := `
		SELECT p.id, p.title, p.description, p.status, 
		       p.created_at, p.updated_at, pu.role
		FROM projects p
		JOIN project_users pu ON p.id = pu.project_id
		WHERE pu.user_id = ? AND p.status = ?
		ORDER BY p.created_at DESC
	`

	rows, err := db.Query(query, userID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get user projects by status: %v", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		project := &Project{}

		err := rows.Scan(
			&project.ID, &project.Title, &project.Description,
			&project.Status, &project.CreatedAt, &project.UpdatedAt,
			&project.UserRole,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %v", err)
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// UpdateProject updates an existing project
func (db *DB) UpdateProject(projectID, userID int, title, description string, status ProjectStatus) error {
	// First check if user has permission to update this project
	userRole, err := db.GetUserRoleInProject(projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to check user permissions: %v", err)
	}

	// Only owners and admins can update projects
	if userRole != RoleOwner && userRole != RoleAdmin {
		return fmt.Errorf("insufficient permissions to update project")
	}

	query := `
		UPDATE projects 
		SET title = ?, description = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err = db.Exec(query, title, description, status, projectID)
	if err != nil {
		return fmt.Errorf("failed to update project: %v", err)
	}

	return nil
}

// UpdateProjectStatus updates only the status of a project
func (db *DB) UpdateProjectStatus(projectID, userID int, status ProjectStatus) error {
	// Check user permissions
	userRole, err := db.GetUserRoleInProject(projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to check user permissions: %v", err)
	}
	if userRole != RoleOwner && userRole != RoleAdmin && userRole != RoleMember {
		return fmt.Errorf("insufficient permissions: viewers cannot update project status")
	}

	query := `
		UPDATE projects 
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := db.Exec(query, status, projectID)
	if err != nil {
		return fmt.Errorf("failed to update project status: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project not found")
	}

	return nil
}

// DeleteProject deletes a project (only owners can delete)
func (db *DB) DeleteProject(projectID, userID int) error {
	// Check user permissions
	userRole, err := db.GetUserRoleInProject(projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to check user permissions: %v", err)
	}
	if userRole != RoleOwner {
		return fmt.Errorf("insufficient permissions: only owners can delete projects")
	}

	query := "DELETE FROM projects WHERE id = ?"

	result, err := db.Exec(query, projectID)
	if err != nil {
		return fmt.Errorf("failed to delete project: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project not found")
	}

	return nil
}

// GetProjectCount returns the total number of projects for a user
func (db *DB) GetProjectCount(userID int) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM project_users 
		WHERE user_id = ?
	`

	var count int
	err := db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get project count: %v", err)
	}

	return count, nil
}

// GetProjectCountByStatus returns the number of projects with a specific status for a user
func (db *DB) GetProjectCountByStatus(userID int, status ProjectStatus) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM projects p
		JOIN project_users pu ON p.id = pu.project_id
		WHERE pu.user_id = ? AND p.status = ?
	`

	var count int
	err := db.QueryRow(query, userID, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get project count by status: %v", err)
	}

	return count, nil
}

// AddUserToProject adds a user to a project with specified role
func (db *DB) AddUserToProject(projectID, userID, inviterUserID int, role ProjectRole) error {
	// Check inviter permissions
	inviterRole, err := db.GetUserRoleInProject(projectID, inviterUserID)
	if err != nil {
		return fmt.Errorf("failed to check inviter permissions: %v", err)
	}
	if inviterRole != RoleOwner && inviterRole != RoleAdmin {
		return fmt.Errorf("insufficient permissions: only owners and admins can add users")
	}

	query := `
		INSERT INTO project_users (project_id, user_id, role) 
		VALUES (?, ?, ?)
	`

	_, err = db.Exec(query, projectID, userID, role)
	if err != nil {
		return fmt.Errorf("failed to add user to project: %v", err)
	}

	return nil
}

// RemoveUserFromProject removes a user from a project
func (db *DB) RemoveUserFromProject(projectID, userID, removerUserID int) error {
	// Check remover permissions
	removerRole, err := db.GetUserRoleInProject(projectID, removerUserID)
	if err != nil {
		return fmt.Errorf("failed to check remover permissions: %v", err)
	}
	if removerRole != RoleOwner && removerRole != RoleAdmin {
		return fmt.Errorf("insufficient permissions: only owners and admins can remove users")
	}

	// Don't allow removing the last owner
	if userID != removerUserID {
		userRole, err := db.GetUserRoleInProject(projectID, userID)
		if err != nil {
			return fmt.Errorf("failed to check user role: %v", err)
		}
		if userRole == RoleOwner {
			ownerCount, err := db.GetProjectOwnerCount(projectID)
			if err != nil {
				return fmt.Errorf("failed to check owner count: %v", err)
			}
			if ownerCount <= 1 {
				return fmt.Errorf("cannot remove the last owner")
			}
		}
	}

	query := "DELETE FROM project_users WHERE project_id = ? AND user_id = ?"

	_, err = db.Exec(query, projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove user from project: %v", err)
	}

	return nil
}

// UpdateUserRoleInProject updates a user's role in a project
func (db *DB) UpdateUserRoleInProject(projectID, userID, updaterUserID int, newRole ProjectRole) error {
	// Check updater permissions
	updaterRole, err := db.GetUserRoleInProject(projectID, updaterUserID)
	if err != nil {
		return fmt.Errorf("failed to check updater permissions: %v", err)
	}
	if updaterRole != RoleOwner && updaterRole != RoleAdmin {
		return fmt.Errorf("insufficient permissions: only owners and admins can update roles")
	}

	query := `
		UPDATE project_users 
		SET role = ? 
		WHERE project_id = ? AND user_id = ?
	`

	_, err = db.Exec(query, newRole, projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to update user role: %v", err)
	}

	return nil
}

// GetUserRoleInProject returns the role of a user in a specific project
func (db *DB) GetUserRoleInProject(projectID, userID int) (ProjectRole, error) {
	query := `
		SELECT role 
		FROM project_users 
		WHERE project_id = ? AND user_id = ?
	`

	var role ProjectRole
	err := db.QueryRow(query, projectID, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("user not found in project")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user role: %v", err)
	}

	return role, nil
}

// GetProjectUsers returns all users in a project with their roles
func (db *DB) GetProjectUsers(projectID int) ([]*ProjectUser, error) {
	query := `
		SELECT id, project_id, user_id, role, joined_at
		FROM project_users 
		WHERE project_id = ?
		ORDER BY joined_at ASC
	`

	rows, err := db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project users: %v", err)
	}
	defer rows.Close()

	var projectUsers []*ProjectUser
	for rows.Next() {
		pu := &ProjectUser{}
		err := rows.Scan(&pu.ID, &pu.ProjectID, &pu.UserID, &pu.Role, &pu.JoinedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project user: %v", err)
		}
		projectUsers = append(projectUsers, pu)
	}

	return projectUsers, nil
}

// GetProjectOwnerCount returns the number of owners in a project
func (db *DB) GetProjectOwnerCount(projectID int) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM project_users 
		WHERE project_id = ? AND role = ?
	`

	var count int
	err := db.QueryRow(query, projectID, RoleOwner).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get owner count: %v", err)
	}

	return count, nil
}

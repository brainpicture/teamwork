package internal

import (
	"database/sql"
	"fmt"
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

// ProjectPriority represents the priority of a project
type ProjectPriority string

const (
	PriorityLow    ProjectPriority = "low"
	PriorityMedium ProjectPriority = "medium"
	PriorityHigh   ProjectPriority = "high"
	PriorityUrgent ProjectPriority = "urgent"
)

// Project represents a project in the database
type Project struct {
	ID          int             `json:"id"`
	UserID      int             `json:"user_id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      ProjectStatus   `json:"status"`
	Priority    ProjectPriority `json:"priority"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Deadline    *time.Time      `json:"deadline,omitempty"`
}

// CreateProject creates a new project for a user
func (db *DB) CreateProject(userID int, title, description string, priority ProjectPriority, deadline *time.Time) (*Project, error) {
	query := `
		INSERT INTO projects (user_id, title, description, priority, deadline) 
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := db.Exec(query, userID, title, description, priority, deadline)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %v", err)
	}

	projectID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get project ID: %v", err)
	}

	return db.GetProjectByID(int(projectID))
}

// GetProjectByID retrieves a project by its ID
func (db *DB) GetProjectByID(projectID int) (*Project, error) {
	query := `
		SELECT id, user_id, title, description, status, priority, created_at, updated_at, deadline
		FROM projects WHERE id = ?
	`

	project := &Project{}
	var deadline sql.NullTime

	err := db.QueryRow(query, projectID).Scan(
		&project.ID, &project.UserID, &project.Title, &project.Description,
		&project.Status, &project.Priority, &project.CreatedAt, &project.UpdatedAt,
		&deadline,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %v", err)
	}

	if deadline.Valid {
		project.Deadline = &deadline.Time
	}

	return project, nil
}

// GetUserProjects retrieves all projects for a specific user
func (db *DB) GetUserProjects(userID int) ([]*Project, error) {
	query := `
		SELECT id, user_id, title, description, status, priority, created_at, updated_at, deadline
		FROM projects WHERE user_id = ? ORDER BY created_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user projects: %v", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		project := &Project{}
		var deadline sql.NullTime

		err := rows.Scan(
			&project.ID, &project.UserID, &project.Title, &project.Description,
			&project.Status, &project.Priority, &project.CreatedAt, &project.UpdatedAt,
			&deadline,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %v", err)
		}

		if deadline.Valid {
			project.Deadline = &deadline.Time
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// GetUserProjectsByStatus retrieves projects for a user filtered by status
func (db *DB) GetUserProjectsByStatus(userID int, status ProjectStatus) ([]*Project, error) {
	query := `
		SELECT id, user_id, title, description, status, priority, created_at, updated_at, deadline
		FROM projects WHERE user_id = ? AND status = ? ORDER BY created_at DESC
	`

	rows, err := db.Query(query, userID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get user projects by status: %v", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		project := &Project{}
		var deadline sql.NullTime

		err := rows.Scan(
			&project.ID, &project.UserID, &project.Title, &project.Description,
			&project.Status, &project.Priority, &project.CreatedAt, &project.UpdatedAt,
			&deadline,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %v", err)
		}

		if deadline.Valid {
			project.Deadline = &deadline.Time
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// UpdateProject updates an existing project
func (db *DB) UpdateProject(project *Project) error {
	query := `
		UPDATE projects 
		SET title = ?, description = ?, status = ?, priority = ?, deadline = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	_, err := db.Exec(query, project.Title, project.Description, project.Status,
		project.Priority, project.Deadline, project.ID, project.UserID)
	if err != nil {
		return fmt.Errorf("failed to update project: %v", err)
	}

	return nil
}

// UpdateProjectStatus updates only the status of a project
func (db *DB) UpdateProjectStatus(projectID, userID int, status ProjectStatus) error {
	query := `
		UPDATE projects 
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, status, projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to update project status: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project not found or user not authorized")
	}

	return nil
}

// DeleteProject deletes a project (only by the owner)
func (db *DB) DeleteProject(projectID, userID int) error {
	query := `DELETE FROM projects WHERE id = ? AND user_id = ?`

	result, err := db.Exec(query, projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete project: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project not found or user not authorized")
	}

	return nil
}

// GetProjectCount returns the total number of projects for a user
func (db *DB) GetProjectCount(userID int) (int, error) {
	query := `SELECT COUNT(*) FROM projects WHERE user_id = ?`

	var count int
	err := db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get project count: %v", err)
	}

	return count, nil
}

// GetProjectCountByStatus returns the number of projects for a user by status
func (db *DB) GetProjectCountByStatus(userID int, status ProjectStatus) (int, error) {
	query := `SELECT COUNT(*) FROM projects WHERE user_id = ? AND status = ?`

	var count int
	err := db.QueryRow(query, userID, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get project count by status: %v", err)
	}

	return count, nil
}

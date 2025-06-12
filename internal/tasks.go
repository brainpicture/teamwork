package internal

import (
	"database/sql"
	"fmt"
	"time"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskTodo       TaskStatus = "todo"
	TaskInProgress TaskStatus = "in_progress"
	TaskReview     TaskStatus = "review"
	TaskDone       TaskStatus = "done"
	TaskCancelled  TaskStatus = "cancelled"
)

// TaskPriority represents the priority level of a task
type TaskPriority string

const (
	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
	PriorityUrgent TaskPriority = "urgent"
)

// Task represents a task in the database
type Task struct {
	ID           int          `json:"id"`
	ProjectID    int          `json:"project_id"`
	UserID       int          `json:"user_id"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Status       TaskStatus   `json:"status"`
	Priority     TaskPriority `json:"priority"`
	Deadline     *time.Time   `json:"deadline,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
	ProjectTitle string       `json:"project_title,omitempty"` // For display purposes
}

// CreateTask creates a new task in a project
func (db *DB) CreateTask(projectID, userID int, title, description string, priority TaskPriority, deadline *time.Time) (*Task, error) {
	// First check if user has access to this project
	userRole, err := db.GetUserRoleInProject(projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project access: %v", err)
	}
	if userRole == "" {
		return nil, fmt.Errorf("user does not have access to this project")
	}

	query := `
		INSERT INTO tasks (project_id, user_id, title, description, priority, deadline)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := db.Exec(query, projectID, userID, title, description, priority, deadline)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %v", err)
	}

	taskID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get task ID: %v", err)
	}

	return db.GetTaskByID(int(taskID), userID)
}

// GetTaskByID retrieves a task by its ID (with project access check)
func (db *DB) GetTaskByID(taskID, userID int) (*Task, error) {
	query := `
		SELECT t.id, t.project_id, t.user_id, t.title, t.description, 
		       t.status, t.priority, t.deadline, t.created_at, t.updated_at, 
		       t.completed_at, p.title
		FROM tasks t
		JOIN projects p ON t.project_id = p.id
		JOIN project_users pu ON p.id = pu.project_id
		WHERE t.id = ? AND pu.user_id = ?
	`

	task := &Task{}
	var deadline, completedAt sql.NullTime

	err := db.QueryRow(query, taskID, userID).Scan(
		&task.ID, &task.ProjectID, &task.UserID, &task.Title, &task.Description,
		&task.Status, &task.Priority, &deadline, &task.CreatedAt, &task.UpdatedAt,
		&completedAt, &task.ProjectTitle,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %v", err)
	}

	if deadline.Valid {
		task.Deadline = &deadline.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}

	return task, nil
}

// GetUserTasks retrieves all tasks for a user across all their projects
func (db *DB) GetUserTasks(userID int) ([]*Task, error) {
	query := `
		SELECT t.id, t.project_id, t.user_id, t.title, t.description, 
		       t.status, t.priority, t.deadline, t.created_at, t.updated_at, 
		       t.completed_at, p.title
		FROM tasks t
		JOIN projects p ON t.project_id = p.id
		JOIN project_users pu ON p.id = pu.project_id
		WHERE pu.user_id = ?
		ORDER BY t.created_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tasks: %v", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		var deadline, completedAt sql.NullTime

		err := rows.Scan(
			&task.ID, &task.ProjectID, &task.UserID, &task.Title, &task.Description,
			&task.Status, &task.Priority, &deadline, &task.CreatedAt, &task.UpdatedAt,
			&completedAt, &task.ProjectTitle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %v", err)
		}

		if deadline.Valid {
			task.Deadline = &deadline.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetProjectTasks retrieves all tasks for a specific project
func (db *DB) GetProjectTasks(projectID, userID int) ([]*Task, error) {
	// Check if user has access to this project
	userRole, err := db.GetUserRoleInProject(projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project access: %v", err)
	}
	if userRole == "" {
		return nil, fmt.Errorf("user does not have access to this project")
	}

	query := `
		SELECT t.id, t.project_id, t.user_id, t.title, t.description, 
		       t.status, t.priority, t.deadline, t.created_at, t.updated_at, 
		       t.completed_at, p.title
		FROM tasks t
		JOIN projects p ON t.project_id = p.id
		WHERE t.project_id = ?
		ORDER BY t.created_at DESC
	`

	rows, err := db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project tasks: %v", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		var deadline, completedAt sql.NullTime

		err := rows.Scan(
			&task.ID, &task.ProjectID, &task.UserID, &task.Title, &task.Description,
			&task.Status, &task.Priority, &deadline, &task.CreatedAt, &task.UpdatedAt,
			&completedAt, &task.ProjectTitle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %v", err)
		}

		if deadline.Valid {
			task.Deadline = &deadline.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTasksByStatus retrieves tasks by status for a user
func (db *DB) GetTasksByStatus(userID int, status TaskStatus) ([]*Task, error) {
	query := `
		SELECT t.id, t.project_id, t.user_id, t.title, t.description, 
		       t.status, t.priority, t.deadline, t.created_at, t.updated_at, 
		       t.completed_at, p.title
		FROM tasks t
		JOIN projects p ON t.project_id = p.id
		JOIN project_users pu ON p.id = pu.project_id
		WHERE pu.user_id = ? AND t.status = ?
		ORDER BY t.created_at DESC
	`

	rows, err := db.Query(query, userID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by status: %v", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		var deadline, completedAt sql.NullTime

		err := rows.Scan(
			&task.ID, &task.ProjectID, &task.UserID, &task.Title, &task.Description,
			&task.Status, &task.Priority, &deadline, &task.CreatedAt, &task.UpdatedAt,
			&completedAt, &task.ProjectTitle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %v", err)
		}

		if deadline.Valid {
			task.Deadline = &deadline.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// UpdateTask updates an existing task
func (db *DB) UpdateTask(taskID, userID int, title, description string, status TaskStatus, priority TaskPriority, deadline *time.Time) error {
	// First get the task to check project access
	task, err := db.GetTaskByID(taskID, userID)
	if err != nil {
		return fmt.Errorf("failed to get task: %v", err)
	}
	if task == nil {
		return fmt.Errorf("task not found or no access")
	}

	// Check if user has permission to update tasks in this project
	userRole, err := db.GetUserRoleInProject(task.ProjectID, userID)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %v", err)
	}
	if userRole == "" {
		return fmt.Errorf("no access to project")
	}

	// Set completed_at if status is changing to done
	var completedAt *time.Time
	if status == TaskDone && task.Status != TaskDone {
		now := time.Now()
		completedAt = &now
	} else if status != TaskDone && task.Status == TaskDone {
		// Reset completed_at if moving away from done
		completedAt = nil
	}

	query := `
		UPDATE tasks 
		SET title = ?, description = ?, status = ?, priority = ?, deadline = ?, 
		    completed_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err = db.Exec(query, title, description, status, priority, deadline, completedAt, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task: %v", err)
	}

	return nil
}

// UpdateTaskStatus updates only the status of a task
func (db *DB) UpdateTaskStatus(taskID, userID int, status TaskStatus) error {
	// Get current task
	task, err := db.GetTaskByID(taskID, userID)
	if err != nil {
		return fmt.Errorf("failed to get task: %v", err)
	}
	if task == nil {
		return fmt.Errorf("task not found or no access")
	}

	// Set completed_at if status is changing to done
	var completedAt *time.Time
	if status == TaskDone && task.Status != TaskDone {
		now := time.Now()
		completedAt = &now
	} else if status != TaskDone && task.Status == TaskDone {
		completedAt = nil
	}

	query := `
		UPDATE tasks 
		SET status = ?, completed_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err = db.Exec(query, status, completedAt, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task status: %v", err)
	}

	return nil
}

// DeleteTask deletes a task
func (db *DB) DeleteTask(taskID, userID int) error {
	// First check if user has access to the task
	task, err := db.GetTaskByID(taskID, userID)
	if err != nil {
		return fmt.Errorf("failed to get task: %v", err)
	}
	if task == nil {
		return fmt.Errorf("task not found or no access")
	}

	// Check if user has permission to delete tasks in this project
	userRole, err := db.GetUserRoleInProject(task.ProjectID, userID)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %v", err)
	}

	// Only owners, admins, or task creator can delete tasks
	if userRole != RoleOwner && userRole != RoleAdmin && task.UserID != userID {
		return fmt.Errorf("insufficient permissions to delete task")
	}

	_, err = db.Exec("DELETE FROM tasks WHERE id = ?", taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task: %v", err)
	}

	return nil
}

// GetTasksWithDeadline retrieves tasks that have deadlines (for notifications)
func (db *DB) GetTasksWithDeadline(userID int, daysBefore int) ([]*Task, error) {
	query := `
		SELECT t.id, t.project_id, t.user_id, t.title, t.description, 
		       t.status, t.priority, t.deadline, t.created_at, t.updated_at, 
		       t.completed_at, p.title
		FROM tasks t
		JOIN projects p ON t.project_id = p.id
		JOIN project_users pu ON p.id = pu.project_id
		WHERE pu.user_id = ? AND t.deadline IS NOT NULL 
		      AND t.deadline <= DATE_ADD(NOW(), INTERVAL ? DAY)
		      AND t.status NOT IN ('done', 'cancelled')
		ORDER BY t.deadline ASC
	`

	rows, err := db.Query(query, userID, daysBefore)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks with deadline: %v", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		var deadline, completedAt sql.NullTime

		err := rows.Scan(
			&task.ID, &task.ProjectID, &task.UserID, &task.Title, &task.Description,
			&task.Status, &task.Priority, &deadline, &task.CreatedAt, &task.UpdatedAt,
			&completedAt, &task.ProjectTitle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %v", err)
		}

		if deadline.Valid {
			task.Deadline = &deadline.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

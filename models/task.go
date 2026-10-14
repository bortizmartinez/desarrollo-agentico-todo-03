package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TitleMaxLength es el número máximo de caracteres permitido en el título.
const TitleMaxLength = 200

// AssigneeMaxLength es el número máximo de caracteres permitido en el responsable.
const AssigneeMaxLength = 100

// Valores de prioridad admitidos.
const (
	PriorityUrgent = "urgente"
	PriorityHigh   = "alta"
	PriorityMedium = "media"
	PriorityLow    = "baja"
)

// PriorityOption asocia un valor de prioridad con su etiqueta visible.
type PriorityOption struct {
	Value string
	Label string
}

// Priorities enumera las prioridades en el orden en que se ofrecen.
var Priorities = []PriorityOption{
	{Value: PriorityUrgent, Label: "Urgente"},
	{Value: PriorityHigh, Label: "Alta"},
	{Value: PriorityMedium, Label: "Media"},
	{Value: PriorityLow, Label: "Baja"},
}

// Task representa una tarea de la lista.
type Task struct {
	ID        uuid.UUID `json:"id" db:"id" form:"-"`
	Title     string    `json:"title" db:"title" form:"title"`
	Priority  string    `json:"priority" db:"priority" form:"priority"`
	Assignee  string    `json:"assignee" db:"assignee" form:"assignee"`
	Completed bool      `json:"completed" db:"completed" form:"-"`
	CreatedAt time.Time `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" form:"-"`
}

// Tasks es una colección de Task.
type Tasks []Task

// PriorityLabel devuelve la etiqueta visible de la prioridad de la tarea.
func (t Task) PriorityLabel() string {
	for _, p := range Priorities {
		if p.Value == t.Priority {
			return p.Label
		}
	}
	return t.Priority
}

// Validate se ejecuta en cada llamada a los métodos "pop.Validate*".
func (t *Task) Validate(tx *pop.Connection) (*validate.Errors, error) {
	verrs := validate.NewErrors()

	title := strings.TrimSpace(t.Title)
	if title == "" {
		verrs.Add("title", "El título es obligatorio.")
	} else if len([]rune(title)) > TitleMaxLength {
		verrs.Add("title", fmt.Sprintf("El título no puede superar los %d caracteres.", TitleMaxLength))
	}

	if t.Priority != "" && !validPriority(t.Priority) {
		verrs.Add("priority", "La prioridad no es válida.")
	}

	assignee := strings.TrimSpace(t.Assignee)
	if len([]rune(assignee)) > AssigneeMaxLength {
		verrs.Add("assignee", fmt.Sprintf("El responsable no puede superar los %d caracteres.", AssigneeMaxLength))
	}

	return verrs, nil
}

// BeforeSave asigna la prioridad por defecto cuando no se ha indicado ninguna.
func (t *Task) BeforeSave(tx *pop.Connection) error {
	if t.Priority == "" {
		t.Priority = PriorityMedium
	}
	return nil
}

func validPriority(value string) bool {
	for _, p := range Priorities {
		if p.Value == value {
			return true
		}
	}
	return false
}

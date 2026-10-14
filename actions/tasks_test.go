package actions

import (
	"net/url"
	"strings"

	"todo/models"

	"github.com/gofrs/uuid"
)

func (as *ActionSuite) Test_TasksIndex_EmptyState() {
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "No hay tareas todavía")
}

func (as *ActionSuite) Test_TasksIndex_ListsTasks() {
	as.LoadFixture("tres tareas")

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "Comprar material de oficina")
	as.Contains(body, "Llamar al gestor")
	as.Contains(body, "Publicar la oferta de empleo")
	as.Contains(body, "Completada")
}

func (as *ActionSuite) Test_TasksCreate_Valid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "  Nueva tarea de prueba  "})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal("Nueva tarea de prueba", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksCreate_EmptyTitle_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: ""})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksCreate_OnlySpaces_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "     "})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksToggle_CompletesAndReopens() {
	task := &models.Task{Title: "Pendiente de completar"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)

	res = as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksToggle_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s/toggle", id).Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksToggle_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/toggle").Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksDestroy() {
	task := &models.Task{Title: "Tarea que se elimina"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s", task.ID).Delete()
	as.Equal(303, res.Code)

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksDestroy_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s", id).Delete()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) taskForEditing(completed bool) *models.Task {
	task := &models.Task{Title: "Título original", Completed: completed}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	as.NoError(as.DB.Reload(task))
	return task
}

func (as *ActionSuite) Test_TasksIndex_EditLink() {
	task := as.taskForEditing(false)
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "/tasks/"+task.ID.String()+"/edit")
	as.Contains(res.Body.String(), "Editar tarea: Título original")
}

func (as *ActionSuite) Test_TasksEdit_PrefillsTitle() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `value="Título original"`)
	as.Contains(res.Body.String(), `value="PUT"`)
	as.Contains(res.Body.String(), "Guardar cambios")
	as.Contains(res.Body.String(), "Cancelar")
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksEdit_NotFound() {
	res := as.HTML("/tasks/%s/edit", uuid.Must(uuid.NewV4())).Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/edit").Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_EscapesTitle() {
	task := &models.Task{Title: `<img src=x onerror="alert(1)">`}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.NotContains(res.Body.String(), `<img src=x`)
	as.Contains(res.Body.String(), "&lt;img")
}

func (as *ActionSuite) Test_TasksUpdate_PersistsTitleOnly() {
	task := as.taskForEditing(true)
	id, createdAt, updatedAt := task.ID, task.CreatedAt, task.UpdatedAt
	res := as.HTML("/tasks/%s", id).Put(url.Values{
		"title": {"  Título actualizado  "}, "completed": {"false"},
		"id": {uuid.Must(uuid.NewV4()).String()}, "created_at": {"2000-01-01T00:00:00Z"},
	})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))
	as.NoError(as.DB.Reload(task))
	as.Equal("Título actualizado", task.Title)
	as.Equal(id, task.ID)
	as.True(task.Completed)
	as.True(task.CreatedAt.Equal(createdAt))
	as.False(task.UpdatedAt.Before(updatedAt))
	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(1, count)
}

func (as *ActionSuite) Test_TasksUpdate_PendingTaskRemainsPending() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Nuevo título"}, "completed": {"true"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Nuevo título", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_EmptyTitle() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {""}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.Contains(res.Body.String(), `aria-describedby="title-error"`)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_OnlySpaces() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"   "}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_TitleTooLong() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength+1)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título no puede superar los 200 caracteres.")
	as.Contains(res.Body.String(), title)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_TitleAtLimit() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal(title, task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_NotFound() {
	res := as.HTML("/tasks/%s", uuid.Must(uuid.NewV4())).Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid").Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksCreate_WithPriority() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"Revisar contrato"}, "priority": {models.PriorityUrgent}})
	as.Equal(303, res.Code)

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal("Revisar contrato", task.Title)
	as.Equal(models.PriorityUrgent, task.Priority)
}

func (as *ActionSuite) Test_TasksCreate_WithoutPriority_DefaultsToMedium() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"Sin prioridad"}})
	as.Equal(303, res.Code)

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal(models.PriorityMedium, task.Priority)
}

func (as *ActionSuite) Test_TasksCreate_InvalidPriority() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"Tarea"}, "priority": {"altísima"}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "La prioridad no es válida.")
	as.Contains(res.Body.String(), `aria-describedby="priority-error"`)

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksIndex_ShowsPriority() {
	task := &models.Task{Title: "Llamar al cliente", Priority: models.PriorityHigh}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	body := res.Body.String()
	as.Contains(body, `class="priority priority-alta"`)
	as.Contains(body, `<option value="media" selected>Media</option>`)
}

func (as *ActionSuite) Test_TasksEdit_HasNoPrioritySelector() {
	task := &models.Task{Title: "Con prioridad", Priority: models.PriorityLow}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.NotContains(res.Body.String(), `name="priority"`)
}

func (as *ActionSuite) Test_TasksUpdate_IgnoresPriority() {
	task := &models.Task{Title: "Urgente", Priority: models.PriorityUrgent}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Urgente revisado"}, "priority": {models.PriorityLow}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Urgente revisado", task.Title)
	as.Equal(models.PriorityUrgent, task.Priority)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidPriorityIsIgnored() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Nuevo"}, "priority": {"altísima"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Nuevo", task.Title)
	as.Equal(models.PriorityMedium, task.Priority)
}

func (as *ActionSuite) Test_TasksCreate_WithAssignee() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"Revisar contrato"}, "assignee": {"  Ana Pérez  "}})
	as.Equal(303, res.Code)

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal("Ana Pérez", task.Assignee)
}

func (as *ActionSuite) Test_TasksCreate_WithoutAssignee_IsEmpty() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"Sin responsable"}})
	as.Equal(303, res.Code)

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal("", task.Assignee)
}

func (as *ActionSuite) Test_TasksCreate_AssigneeTooLong_Invalid() {
	assignee := strings.Repeat("a", models.AssigneeMaxLength+1)
	res := as.HTML("/tasks").Post(url.Values{"title": {"Tarea"}, "assignee": {assignee}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El responsable no puede superar los 100 caracteres.")
	as.Contains(res.Body.String(), `aria-describedby="assignee-error"`)

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksIndex_ShowsAssignee() {
	withAssignee := &models.Task{Title: "Con responsable", Assignee: "Luis Gómez"}
	verrs, err := as.DB.ValidateAndCreate(withAssignee)
	as.NoError(err)
	as.False(verrs.HasAny())

	withoutAssignee := &models.Task{Title: "Sin responsable"}
	verrs, err = as.DB.ValidateAndCreate(withoutAssignee)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "Luis Gómez")
}

func (as *ActionSuite) Test_TasksEdit_PrefillsAssignee() {
	task := &models.Task{Title: "Con responsable", Assignee: "María López"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `value="María López"`)
}

func (as *ActionSuite) Test_TasksUpdate_ChangesAssignee() {
	task := &models.Task{Title: "Urgente", Priority: models.PriorityUrgent}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Urgente"}, "assignee": {"  Carlos Ruiz  "}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Carlos Ruiz", task.Assignee)
	as.Equal(models.PriorityUrgent, task.Priority)
}

func (as *ActionSuite) Test_TasksUpdate_ClearsAssignee() {
	task := &models.Task{Title: "Con responsable", Assignee: "Pedro Sánchez"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Con responsable"}, "assignee": {""}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("", task.Assignee)
}

func (as *ActionSuite) Test_TasksUpdate_AssigneeTooLong_Invalid() {
	task := as.taskForEditing(false)
	assignee := strings.Repeat("a", models.AssigneeMaxLength+1)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Título original"}, "assignee": {assignee}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El responsable no puede superar los 100 caracteres.")
	as.NoError(as.DB.Reload(task))
	as.Equal("", task.Assignee)
}

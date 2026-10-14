# Feature: Campo Responsable en tareas

## Encargo y criterios de aceptación

Issue [#1](https://github.com/bortizmartinez/desarrollo-agentico-todo-03/issues/1): añadir a las tareas un campo **Responsable** (texto libre, la persona encargada de la tarea).

1. El formulario de creación tiene un campo de texto opcional «Responsable», junto al selector de prioridad.
2. El responsable se guarda sin espacios al principio ni al final. Si se deja vacío, la tarea queda sin responsable.
3. Si tiene más de 100 caracteres, se responde con 422 y el mensaje «El responsable no puede superar los 100 caracteres.», y no se guarda nada.
4. La lista muestra el responsable de cada tarea que lo tenga, junto a la etiqueta de prioridad. Las tareas sin responsable no muestran nada.
5. La pantalla de edición permite cambiar el responsable o quitarlo (dejando el campo vacío). La edición sigue sin permitir cambiar la prioridad, el estado `completed`, el `id` ni `created_at`.
6. Las tareas existentes quedan sin responsable tras la migración.

Decisiones ya tomadas en la issue:

- Es texto libre: no existe modelo de usuarios y no se crea ninguno.
- Se requiere una migración nueva. No se modifica `20261014000000_add_priority_to_tasks`.
- No se ordena ni se filtra por responsable.

## Comportamiento actual y propuesta

Actualmente `Task` tiene `Title`, `Priority`, `Completed`, `CreatedAt`, `UpdatedAt`. No existe ningún campo de responsable ni rastro de que haya existido antes (no hay commits ni archivos con "responsable" o "assignee" en el histórico). Sin embargo, `actions/tasks_test.go` ya contiene dos aserciones negativas defensivas (`as.NotContains(body, 'name="assignee"')` en `Test_TasksIndex_ShowsPriority` y `Test_TasksEdit_HasNoPrioritySelector`) añadidas junto con la prioridad; confirman que el nombre interno esperado para este campo es `assignee`, aunque la etiqueta visible sea «Responsable». Esta feature añade justamente ese campo, así que esas dos aserciones quedan obsoletas y se eliminan (no se modifica el resto del test).

Propuesta: añadir `Assignee string` al modelo con una constante `AssigneeMaxLength = 100`, columna `assignee` en BD (string, tamaño 100, default `''`, NOT NULL igual que `title`), validación de longitud, recorte de espacios en las acciones (igual que `Title`), inclusión en el formulario de alta y en el de edición, y renderizado condicional en el índice.

## Alcance

**Incluido**

- Migración nueva que añade la columna `assignee` a `tasks`.
- Modelo: campo `Assignee`, constante `AssigneeMaxLength`, validación de longitud.
- Acción `TasksCreate`: recorta y persiste `assignee`.
- Acción `TasksUpdate`: añade `assignee` al struct de entrada editable, recorta y persiste; sigue sin aceptar `priority`, `completed`, `id` ni `created_at`.
- Template `tasks/index.plush.html`: campo de texto «Responsable» en el formulario de alta (junto al selector de prioridad) y, en cada tarea de la lista, el responsable si lo tiene.
- Template `tasks/edit.plush.html`: campo «Responsable» editable (puede vaciarse).
- CSS: estilo mínimo reutilizando clases/variables existentes (`field-label`, `task-meta`), sin rediseñar el layout. Subir `?v=` de `app.css` en `templates/application.plush.html`.
- Tests de modelo y de acciones para los criterios de aceptación.
- Eliminar las dos aserciones `NotContains(..., 'name="assignee"')` que esta feature hace obsoletas.

**Excluido**

- Modelo de usuarios, autocompletado o validación contra una lista de personas.
- Ordenar o filtrar por responsable.
- Cambios en la migración `20261014000000_add_priority_to_tasks`.

## Diseño por capas

**Persistencia**: migración `migrations/20261014020000_add_assignee_to_tasks.{up,down}.fizz`.

```
# up
add_column("tasks", "assignee", "string", {"size": 100, "default": ""})
```

```
# down
drop_column("tasks", "assignee")
```

Con `default: ""` y sin `null: true` la columna queda NOT NULL con cadena vacía por defecto, igual que el patrón de `title`; las filas existentes quedan con `assignee = ""`, que el resto del código trata como "sin responsable" (igual que ya se hace con `Title == ""` en otros sitios, p. ej. `if t.Priority == ""`).

**Modelo** (`models/task.go`):

- Constante `AssigneeMaxLength = 100`.
- Campo `Assignee string `json:"assignee" db:"assignee" form:"assignee"`` en `Task`.
- En `Validate`, tras la validación de prioridad:
  ```go
  assignee := strings.TrimSpace(t.Assignee)
  if len([]rune(assignee)) > AssigneeMaxLength {
      verrs.Add("assignee", fmt.Sprintf("El responsable no puede superar los %d caracteres.", AssigneeMaxLength))
  }
  ```
  (mismo patrón defensivo que `title`: la acción ya recorta antes de persistir, aquí se recorta de nuevo solo para medir la longitud sin mutar `t.Assignee`).
- No hace falta `BeforeSave` adicional: la cadena vacía ya es el valor por defecto correcto.

**Acciones** (`actions/tasks.go`):

- `TasksCreate`: añadir `task.Assignee = strings.TrimSpace(task.Assignee)` junto al recorte de `Title`. El bind ya puebla `Assignee` vía el tag `form:"assignee"`.
- `TasksUpdate`: ampliar el struct de entrada:
  ```go
  input := struct {
      Title    string `form:"title" json:"title"`
      Assignee string `form:"assignee" json:"assignee"`
  }{}
  ```
  y añadir `task.Assignee = strings.TrimSpace(input.Assignee)`. Esto preserva que `priority`, `completed`, `id` y `created_at` sigan sin poder modificarse (no se añaden al struct de entrada).

**Templates Plush**:

- `tasks/index.plush.html`, en `.new-task-meta` junto al selector de prioridad, añadir un campo de texto:
  ```html
  <div class="field-group">
    <label for="task-assignee" class="field-label">Responsable</label>
    <input
      id="task-assignee"
      type="text"
      name="assignee"
      value="<%= task.Assignee %>"
      maxlength="100"
      autocomplete="off"
      <%= if (len(errors.Get("assignee")) > 0) { %>aria-invalid="true" aria-describedby="assignee-error"<% } %>
    >
  </div>
  ```
  y el bloque de error correspondiente:
  ```html
  <%= if (len(errors.Get("assignee")) > 0) { %>
    <p id="assignee-error" class="field-error" role="alert">
      <%= for (msg) in errors.Get("assignee") { %><%= msg %><% } %>
    </p>
  <% } %>
  ```
  En la lista, dentro de `.task-meta`, tras la prioridad:
  ```html
  <%= if (t.Assignee != "") { %>
    <span class="assignee"><span class="sr-only">Responsable: </span><%= t.Assignee %></span>
  <% } %>
  ```
- `tasks/edit.plush.html`: añadir el campo tras el título, marcando el error solo si es de `assignee` (no reutilizar `errors.HasAny()`, que en esta plantilla solo cubre `title`):
  ```html
  <label for="task-assignee">Responsable</label>
  <input
    id="task-assignee"
    type="text"
    name="assignee"
    value="<%= task.Assignee %>"
    maxlength="100"
    autocomplete="off"
    <%= if (len(errors.Get("assignee")) > 0) { %>aria-invalid="true" aria-describedby="assignee-error"<% } %>
  >
  <%= if (len(errors.Get("assignee")) > 0) { %>
    <p id="assignee-error" class="field-error" role="alert">
      <%= for (msg) in errors.Get("assignee") { %><%= msg %><% } %>
    </p>
  <% } %>
  ```

**CSS** (`public/assets/app.css`): añadir una regla mínima para `.field-group` (envoltorio label+input, igual gap que `.new-task-meta` ya maneja) y `.assignee` (texto discreto en `.task-meta`, reutilizando `var(--color-muted)` ya usado por `.task-meta`). Subir `?v=4` a `?v=5` en `templates/application.plush.html`.

## Archivos relevantes

- `migrations/20261014020000_add_assignee_to_tasks.up.fizz` / `.down.fizz` — nueva columna.
- `models/task.go` — campo, constante, validación.
- `models/task_test.go` — tests de modelo.
- `actions/tasks.go` — create/update.
- `actions/tasks_test.go` — tests de acción; eliminar las dos aserciones obsoletas sobre `name="assignee"`.
- `templates/tasks/index.plush.html` — formulario de alta y listado.
- `templates/tasks/edit.plush.html` — formulario de edición.
- `public/assets/app.css` — estilos mínimos.
- `templates/application.plush.html` — versión de la hoja de estilos.

## Implementación paso a paso

1. Crear la migración `20261014020000_add_assignee_to_tasks` (up/down) y comprobar que `soda migrate -e test` la aplica.
2. Añadir `Assignee` y `AssigneeMaxLength` al modelo, con la validación de longitud. Añadir tests de modelo (vacío, recorte, límite, por encima del límite).
3. Actualizar `TasksCreate` y `TasksUpdate` para recortar y persistir `assignee`, sin tocar los campos no editables.
4. Actualizar `tasks/index.plush.html` (campo de alta + listado) y `tasks/edit.plush.html` (campo editable).
5. Añadir estilos mínimos y subir la versión de `app.css`.
6. Eliminar las dos aserciones `NotContains(..., 'name="assignee"')` en `actions/tasks_test.go` y añadir los tests nuevos de acción.
7. Ejecutar la validación completa (fase 5 del flujo) y la prueba de humo (fase 6).

## Tests y validación

Modelo (`models/task_test.go`):

- `Test_Task_Assignee_DefaultsEmpty`: crear sin `Assignee`, se guarda como `""`.
- `Test_Task_Assignee_TrimsWhitespace`: crear con espacios alrededor, tras recargar queda recortado (vía el mismo patrón de recorte que usan las acciones; si se valida directamente contra el modelo sin pasar por la acción, documentar que el recorte ocurre en la capa de acción, no en `BeforeSave`).
- `Test_Task_Assignee_AtLimit`: 100 caracteres, válido.
- `Test_Task_Assignee_TooLong_Invalid`: 101 caracteres, error con el mensaje exacto, no se persiste.

Acción (`actions/tasks_test.go`):

- `Test_TasksCreate_WithAssignee`: crea con `assignee` y espacios alrededor; se persiste recortado.
- `Test_TasksCreate_WithoutAssignee_IsEmpty`: crea sin `assignee`; queda `""`.
- `Test_TasksCreate_AssigneeTooLong_Invalid`: 101 caracteres → 422, mensaje exacto, no se persiste nada (ni la tarea).
- `Test_TasksIndex_ShowsAssignee`: tarea con responsable se muestra en el índice; tarea sin responsable no muestra el bloque.
- `Test_TasksEdit_PrefillsAssignee`: el formulario de edición precarga el responsable actual.
- `Test_TasksUpdate_ChangesAssignee`: editar cambia el responsable y conserva la prioridad.
- `Test_TasksUpdate_ClearsAssignee`: editar con `assignee` vacío quita el responsable.
- Eliminar las dos aserciones obsoletas `NotContains(..., 'name="assignee"')`.

Validación de todo el flujo (reproduce CI): `gofmt -l .`, `go vet ./...`, `soda migrate -e test`, `go test -p 1 ./...`, `go build ./...`. Prueba de humo: enviar el formulario de `/tasks` con `assignee` con espacios y unicode, comprobar que aparece recortado en el índice.

## Riesgos y recuperación

- **Riesgo**: que el nombre de campo `assignee` no sea realmente el esperado (es una inferencia a partir de los tests defensivos existentes, no un requisito explícito de la issue). **Mitigación**: es solo un identificador interno (atributo `name` del input, columna de BD); no afecta a ningún criterio de aceptación observable, que solo habla de «Responsable» como etiqueta visible.
- **Riesgo**: romper el default de `title` NOT NULL si fizz interpreta mal `"size": 100` para `assignee` sin `null: true`. **Mitigación**: seguir exactamente el patrón ya usado por `title` (sin `null`, con tamaño) en `20260901000000_create_tasks.up.fizz`.
- **Recuperación**: si la migración de bajada no revierte limpiamente, `drop_column` es la operación inversa exacta de `add_column`; no hay transformación de datos que revertir.

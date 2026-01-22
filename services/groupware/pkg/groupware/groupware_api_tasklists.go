package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

// When the request succeeds.
// swagger:response GetTaskLists200
type SwaggerGetTaskLists200 struct {
	// in: body
	Body []jmap.TaskList
}

// swagger:route GET /groupware/accounts/{account}/tasklists tasklist tasklists
// Get all tasklists of an account.
//
// responses:
//
//	200: GetTaskLists200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetTaskLists(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needTaskWithAccount()
		if !ok {
			return resp
		}
		var _ string = accountId

		var body []jmap.TaskList = AllTaskLists
		return etagResponse(single(accountId), body, req.session.State, TaskListResponseObjectType, TaskListsState, "")
	})
}

// When the request succeeds.
// swagger:response GetTaskListById200
type SwaggerGetTaskListById200 struct {
	// in: body
	Body struct {
		*jmap.TaskList
	}
}

// swagger:route GET /groupware/accounts/{account}/tasklists/{tasklistid} tasklist tasklist_by_id
// Get a tasklist by its identifier.
//
// responses:
//
//	200: GetTaskListById200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetTaskListById(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needTaskWithAccount()
		if !ok {
			return resp
		}
		var _ string = accountId

		tasklistId, err := req.PathParam(UriParamTaskListId)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		// TODO replace with proper implementation
		for _, tasklist := range AllTaskLists {
			if tasklist.Id == tasklistId {
				return response(single(accountId), tasklist, req.session.State, "")
			}
		}
		return etagNotFoundResponse(single(accountId), req.session.State, TaskListResponseObjectType, TaskListsState, "")
	})
}

// When the request succeeds.
// swagger:response GetTasksInTaskList200
type SwaggerGetTasksInTaskList200 struct {
	// in: body
	Body []jmap.Task
}

// swagger:route GET /groupware/accounts/{account}/tasklists/{tasklistid}/tasks task tasks_in_tasklist
// Get all the tasks in a tasklist of an account by its identifier.
//
// responses:
//
//	200: GetTasksInTaskList200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetTasksInTaskList(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needTaskWithAccount()
		if !ok {
			return resp
		}
		var _ string = accountId

		tasklistId, err := req.PathParam(UriParamTaskListId)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		// TODO replace with proper implementation
		tasks, ok := TaskMapByTaskListId[tasklistId]
		if !ok {
			return notFoundResponse(single(accountId), req.session.State)
		}
		return etagResponse(single(accountId), tasks, req.session.State, TaskResponseObjectType, TaskState, "")
	})
}

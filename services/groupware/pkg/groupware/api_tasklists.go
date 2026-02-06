package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

// Get all tasklists of an account.
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

// Get a tasklist by its identifier.
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

// Get all the tasks in a tasklist of an account by its identifier.
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

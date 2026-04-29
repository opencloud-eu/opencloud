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
		meta := TaskListsMeta{SessionState: req.session.State}
		return req.respond(accountId, body, TaskListResponseObjectType, meta)
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
			return req.error(accountId, err)
		}
		// TODO replace with proper implementation
		meta := TaskListsMeta{SessionState: req.session.State}
		for _, tasklist := range AllTaskLists {
			if tasklist.Id == tasklistId {
				return req.respond(accountId, tasklist, TaskListResponseObjectType, meta)
			}
		}
		return req.etaggedNotFound(accountId, req.session.State, TaskListResponseObjectType, TaskListsState)
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
			return req.error(accountId, err)
		}
		// TODO replace with proper implementation
		meta := TaskMeta{SessionState: req.session.State}
		tasks, ok := TaskMapByTaskListId[tasklistId]
		if !ok {
			return req.notFound(accountId, TaskResponseObjectType, meta)
		}
		return req.respond(accountId, tasks, TaskResponseObjectType, meta)
	})
}

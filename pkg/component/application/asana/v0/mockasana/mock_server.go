package mockasana

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Router(middlewares ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	for _, m := range middlewares {
		r.Use(m)
	}

	r.Get("/goals/{goalGID}", getGoal)
	r.Put("/goals/{goalGID}", updateGoal)
	r.Delete("/goals/{goalGID}", deleteGoal)
	r.Post("/goals", createGoal)

	r.Get("/projects/{projectGID}", getProject)
	r.Put("/projects/{projectGID}", updateProject)
	r.Delete("/projects/{projectGID}", deleteProject)
	r.Post("/projects/{projectGID}/duplicate", duplicateProject)
	r.Post("/projects", createProject)

	r.Get("/portfolios/{portfolioGID}", getPortfolio)
	r.Put("/portfolios/{portfolioGID}", updatePortfolio)
	r.Delete("/portfolios/{portfolioGID}", deletePortfolio)
	r.Post("/portfolios", createPortfolio)

	r.Get("/tasks/{taskGID}", getTask)
	r.Put("/tasks/{taskGID}", updateTask)
	r.Delete("/tasks/{taskGID}", deleteTask)
	r.Post("/tasks", createTask)
	r.Post("/tasks/{taskGID}/duplicate", duplicateTask)
	r.Post("/tasks/{taskGID}/setParent", setParentTask)
	r.Post("/tasks/{taskGID}/addTag", taskAddTag)
	r.Post("/tasks/{taskGID}/removeTag", taskRemoveTag)
	r.Post("/tasks/{taskGID}/addFollowers", taskAddFollowers)
	r.Post("/tasks/{taskGID}/removeFollowers", taskRemoveFollowers)
	r.Post("/tasks/{taskGID}/addProject", taskAddProject)
	r.Post("/tasks/{taskGID}/removeProject", taskRemoveProject)

	return r
}

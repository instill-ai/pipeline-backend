package mockasana

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func getPortfolio(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	portfolioGID := chi.URLParam(req, "portfolioGID")

	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}
	if portfolioGID == "" {
		http.Error(res, "portfolioGID is required", http.StatusBadRequest)
		return
	}

	for _, portfolio := range FakePortfolio {
		if portfolio.GID == portfolioGID {
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": portfolio,
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "portfolio not found", http.StatusNotFound)
}

func updatePortfolio(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}
	portfolioGID := chi.URLParam(req, "portfolioGID")
	if portfolioGID == "" {
		http.Error(res, "portfolioGID is required", http.StatusBadRequest)
		return
	}

	var portfolio map[string]RawPortfolio

	if err = json.NewDecoder(req.Body).Decode(&portfolio); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	for i, v := range FakePortfolio {
		if v.GID == portfolioGID {
			updatePortfolio := portfolio["data"]
			if updatePortfolio.Name != "" {
				FakePortfolio[i].Name = updatePortfolio.Name
			}
			if updatePortfolio.Color != "" {
				FakePortfolio[i].Color = updatePortfolio.Color
			}
			FakePortfolio[i].Public = updatePortfolio.Public

			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": FakePortfolio[i],
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "portfolio not found", http.StatusNotFound)
}

func createPortfolio(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}

	var portfolio map[string]RawPortfolio

	if err = json.NewDecoder(req.Body).Decode(&portfolio); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	newID := "123456789"
	newPortfolio := portfolio["data"]
	if newPortfolio.Name == "" {
		http.Error(res, "Name is required", http.StatusBadRequest)
		return
	}
	newPortfolio.GID = newID
	newPortfolio.Owner = FakeUser[0]
	newPortfolio.DueOn = "2021-01-01"
	newPortfolio.StartOn = "2021-01-01"
	newPortfolio.CreatedBy = FakeUser[0]
	newPortfolio.CurrentStatus = map[string]string{"title": "On track"}
	newPortfolio.CustomFields = map[string]string{"field": "value"}
	newPortfolio.CustomFieldSettings = map[string]string{"field": "value"}
	FakePortfolio = append(FakePortfolio, newPortfolio)

	if err = json.NewEncoder(res).Encode(
		map[string]interface{}{
			"data": newPortfolio,
		},
	); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deletePortfolio(res http.ResponseWriter, req *http.Request) {
	var err error
	portfolioGID := chi.URLParam(req, "portfolioGID")
	if portfolioGID == "" {
		http.Error(res, "portfolioGID is required", http.StatusBadRequest)
		return
	}

	for i, v := range FakePortfolio {
		if v.GID == portfolioGID {
			FakePortfolio = append(FakePortfolio[:i], FakePortfolio[i+1:]...)
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": RawPortfolio{},
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	http.Error(res, "portfolio not found", http.StatusNotFound)
}

package github

const (
	fakeHost = "https://fake-github.com"
)

func middleWare(req string) int {
	if req == "rate_limit" {
		return 403
	}
	if req == "not_found" {
		return 404
	}
	if req == "unprocessable_entity" {
		return 422
	}
	if req == "no_pr" {
		return 201
	}
	return 200
}

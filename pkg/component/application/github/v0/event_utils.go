package github

func convertRawRepository(r rawRepository) repository {
	return repository{
		ID:              r.ID,
		NodeID:          r.NodeID,
		Name:            r.Name,
		FullName:        r.FullName,
		Private:         r.Private,
		Owner:           convertRawUser(r.Owner),
		HTMLURL:         r.HTMLURL,
		Description:     r.Description,
		Fork:            r.Fork,
		URL:             r.URL,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
		PushedAt:        r.PushedAt,
		Homepage:        r.Homepage,
		Size:            r.Size,
		StargazersCount: r.StargazersCount,
		WatchersCount:   r.WatchersCount,
		Language:        r.Language,
		HasIssues:       r.HasIssues,
		HasProjects:     r.HasProjects,
		HasDownloads:    r.HasDownloads,
		HasWiki:         r.HasWiki,
		HasPages:        r.HasPages,
		ForksCount:      r.ForksCount,
		Archived:        r.Archived,
		OpenIssuesCount: r.OpenIssuesCount,
		License:         convertRawLicense(r.License),
		Forks:           r.Forks,
		OpenIssues:      r.OpenIssues,
		Watchers:        r.Watchers,
		DefaultBranch:   r.DefaultBranch,
		IsTemplate:      r.IsTemplate,
		Topics:          r.Topics,
		Visibility:      r.Visibility,
	}
}

func convertRawUser(r rawUser) user {
	return user(r)
}

func convertRawLicense(r *rawLicense) *license {
	if r == nil {
		return nil
	}
	return &license{
		Key:    r.Key,
		Name:   r.Name,
		SPDXID: r.SPDXID,
		URL:    r.URL,
		NodeID: r.NodeID,
	}
}

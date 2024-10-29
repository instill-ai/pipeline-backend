package instillartifact

// TODO: Change to Instill Format

type UploadFileInput struct {
	Options UploadData `json:"options"`
}

type UploadData struct {
	Option    string `json:"option"`
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
	// Base64 encoded file content
	File        string   `json:"file"`
	FileName    string   `json:"file-name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type UploadFileOutput struct {
	File   FileOutput `json:"file"`
	Status bool       `json:"status"`
}

type FileOutput struct {
	FileUID    string `json:"file-uid"`
	FileName   string `json:"file-name"`
	FileType   string `json:"file-type"`
	CreateTime string `json:"create-time"`
	UpdateTime string `json:"update-time"`
	Size       int64  `json:"size"`
	CatalogID  string `json:"catalog-id"`
}

type UploadFilesInput struct {
	Options UploadMultipleData `json:"options"`
}

type UploadMultipleData struct {
	Option    string `json:"option"`
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
	// Base64 encoded file content
	Files       []string `json:"files"`
	FileNames   []string `json:"file-names"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type UploadFilesOutput struct {
	Files  []FileOutput `json:"files"`
	Status bool         `json:"status"`
}

type GetFilesMetadataInput struct {
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
}

type GetFilesMetadataOutput struct {
	Files []FileOutput `json:"files"`
}

type GetChunksMetadataInput struct {
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
	FileUID   string `json:"file-uid"`
}

type GetChunksMetadataOutput struct {
	Chunks []ChunkOutput `json:"chunks"`
}

type ChunkOutput struct {
	ChunkUID        string `json:"chunk-uid"`
	Retrievable     bool   `json:"retrievable"`
	StartPosition   uint32 `json:"start-position"`
	EndPosition     uint32 `json:"end-position"`
	TokenCount      uint32 `json:"token-count"`
	CreateTime      string `json:"create-time"`
	OriginalFileUID string `json:"original-file-uid"`
}

type GetFileInMarkdownInput struct {
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
	FileUID   string `json:"file-uid"`
}

type GetFileInMarkdownOutput struct {
	OriginalFileUID string `json:"original-file-uid"`
	Content         string `json:"content"`
	CreateTime      string `json:"create-time"`
	UpdateTime      string `json:"update-time"`
}

type SearchChunksInput struct {
	Namespace  string `json:"namespace"`
	CatalogID  string `json:"catalog-id"`
	TextPrompt string `json:"text-prompt"`
	TopK       uint32 `json:"top-k"`
}

type SearchChunksOutput struct {
	Chunks []SimilarityChunk `json:"chunks"`
}

type SimilarityChunk struct {
	ChunkUID        string  `json:"chunk-uid"`
	SimilarityScore float32 `json:"similarity-score"`
	TextContent     string  `json:"text-content"`
	SourceFileName  string  `json:"source-file-name"`
}

type QueryInput struct {
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
	Question  string `json:"question"`
	TopK      int32  `json:"top-k"`
}

type QueryOutput struct {
	Answer string            `json:"answer"`
	Chunks []SimilarityChunk `json:"chunks"`
}

type MatchFileStatusInput struct {
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
	FileUID   string `json:"file-uid"`
}

type MatchFileStatusOutput struct {
	Succeeded bool `json:"succeeded"`
}

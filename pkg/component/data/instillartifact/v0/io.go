package instillartifact

// TODO: Change to Instill Format

// UploadFileInput is the input for uploading a file
type UploadFileInput struct {
	// Options for uploading a file
	Options UploadData `json:"options"`
}

// UploadData is the data for uploading a file
type UploadData struct {
	// Option for uploading a file
	Option string `json:"option"`
	// Namespace for uploading a file
	Namespace string `json:"namespace"`
	// Catalog ID for uploading a file
	CatalogID string `json:"catalog-id"`
	// Base64 encoded file content
	File string `json:"file"`
	// File name
	FileName string `json:"file-name"`
	// Description of the file
	Description string `json:"description"`
	// Tags for the file
	Tags []string `json:"tags"`
}

// UploadFileOutput is the output for uploading a file
type UploadFileOutput struct {
	// File output
	File FileOutput `json:"file"`
	// Status of the trigger file processing
	Status bool `json:"status"`
}

// FileOutput is the output for a file
type FileOutput struct {
	// File UID
	FileUID string `json:"file-uid"`
	// File name
	FileName string `json:"file-name"`
	// File type
	FileType string `json:"file-type"`
	// CreateTime is the time the file was created
	CreateTime string `json:"create-time"`
	// UpdateTime is the time the file was updated
	UpdateTime string `json:"update-time"`
	// Size of the file
	Size int64 `json:"size"`
	// Tags for the file
	CatalogID string `json:"catalog-id"`
}

// GetFilesMetadataInput is the input for getting files metadata
type UploadFilesInput struct {
	// Options for uploading multiple files
	Options UploadMultipleData `json:"options"`
}

// UploadMultipleData is the data for uploading multiple files
type UploadMultipleData struct {
	// Option for uploading multiple files
	Option string `json:"option"`
	// Namespace for uploading multiple files
	Namespace string `json:"namespace"`
	// Catalog ID for uploading multiple files
	CatalogID string `json:"catalog-id"`
	// Base64 encoded files
	Files []string `json:"files"`
	// File names
	FileNames []string `json:"file-names"`
	// Descriptions of the files
	Description string `json:"description"`
	// Tags for the files
	Tags []string `json:"tags"`
}

// UploadFilesOutput is the output for uploading multiple files
type UploadFilesOutput struct {
	// Files output
	Files []FileOutput `json:"files"`
	// Status of the trigger file processing
	Status bool `json:"status"`
}

// GetFilesMetadataInput is the input for getting files metadata
type GetFilesMetadataInput struct {
	// Namespace for getting files metadata
	Namespace string `json:"namespace"`
	// Catalog ID for getting files metadata
	CatalogID string `json:"catalog-id"`
}

// GetFilesMetadataOutput is the output for getting files metadata
type GetFilesMetadataOutput struct {
	// Files output
	Files []FileOutput `json:"files"`
}

// GetFileMetadataInput is the input for getting a file metadata
type GetChunksMetadataInput struct {
	// Namespace for getting chunks metadata
	Namespace string `json:"namespace"`
	// Catalog ID for getting chunks metadata
	CatalogID string `json:"catalog-id"`
	// File UID for getting chunks metadata
	FileUID string `json:"file-uid"`
}

// GetFileMetadataOutput is the output for getting a file metadata
type GetChunksMetadataOutput struct {
	// Chunks output
	Chunks []ChunkOutput `json:"chunks"`
}

// ChunkOutput is the output for a chunk
type ChunkOutput struct {
	// Chunk UID
	ChunkUID string `json:"chunk-uid"`
	// Retrievable means if the chunk is retrievable
	Retrievable bool `json:"retrievable"`
	// Start position of the chunk
	StartPosition uint32 `json:"start-position"`
	// End position of the chunk
	EndPosition uint32 `json:"end-position"`
	// TokenCount is the number of tokens in the chunk
	TokenCount uint32 `json:"token-count"`
	// CreateTime is the time the chunk was created
	CreateTime string `json:"create-time"`
	// OriginalFileUID is the original file UID
	OriginalFileUID string `json:"original-file-uid"`
}

// GetFileInMarkdownInput is the input for getting a file in markdown
type GetFileInMarkdownInput struct {
	// Namespace for getting a file in markdown
	Namespace string `json:"namespace"`
	// Catalog ID for getting a file in markdown
	CatalogID string `json:"catalog-id"`
	// File UID for getting a file in markdown
	FileUID string `json:"file-uid"`
}

// GetFileInMarkdownOutput is the output for getting a file in markdown
type GetFileInMarkdownOutput struct {
	// OriginalFileUID is the original file UID
	OriginalFileUID string `json:"original-file-uid"`
	// Content of the file in markdown
	Content string `json:"content"`
	// CreateTime is the time the file was created
	CreateTime string `json:"create-time"`
	// UpdateTime is the time the file was updated
	UpdateTime string `json:"update-time"`
}

// SearchChunksInput is the input for searching chunks
type SearchChunksInput struct {
	// Namespace for searching chunks
	Namespace string `json:"namespace"`
	// Catalog ID for searching chunks
	CatalogID string `json:"catalog-id"`
	// Text prompt for searching chunks
	TextPrompt string `json:"text-prompt"`
	// TopK for searching chunks
	TopK uint32 `json:"top-k"`
}

// SearchChunksOutput is the output for searching chunks
type SearchChunksOutput struct {
	// Chunks output
	Chunks []SimilarityChunk `json:"chunks"`
}

// SimilarityChunk is the output for a similarity chunk
type SimilarityChunk struct {
	// Chunk UID
	ChunkUID string `json:"chunk-uid"`
	// Similarity score
	SimilarityScore float32 `json:"similarity-score"`
	// Text content of the chunk
	TextContent string `json:"text-content"`
	// Source file name
	SourceFileName string `json:"source-file-name"`
}

// QueryInput is the input for querying
type QueryInput struct {
	// Namespace for querying
	Namespace string `json:"namespace"`
	// Catalog ID for querying
	CatalogID string `json:"catalog-id"`
	// Question for querying
	Question string `json:"question"`
	// TopK for querying
	TopK int32 `json:"top-k"`
}

// QueryOutput is the output for querying
type QueryOutput struct {
	// Answer for the query
	Answer string `json:"answer"`
	// Chunks is related chunks
	Chunks []SimilarityChunk `json:"chunks"`
}

// MatchFileStatusInput is the input for matching file status
type MatchFileStatusInput struct {
	// Namespace for matching file status
	Namespace string `json:"namespace"`
	// Catalog ID for matching file status
	CatalogID string `json:"catalog-id"`
	// File UID for matching file status
	FileUID string `json:"file-uid"`
}

// MatchFileStatusOutput is the output for matching file status
type MatchFileStatusOutput struct {
	// Succeeded means if the file processing status is succeeded
	Succeeded bool `json:"succeeded"`
}

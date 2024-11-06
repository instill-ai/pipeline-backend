package mock

//go:generate minimock -g -i github.com/instill-ai/pipeline-backend/pkg/component/base.UsageHandler -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/pipeline-backend/pkg/component/base.InputReader -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/pipeline-backend/pkg/component/base.OutputWriter -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/pipeline-backend/pkg/component/base.ErrorHandler -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/pipeline-backend/pkg/component/operator/document/v0.commandRunner -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i io.WriteCloser -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/protogen-go/artifact/artifact/v1alpha.ArtifactPublicServiceClient -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/pipeline-backend/pkg/component/data/googledrive/v0/client.IDriveService -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/protogen-go/artifact/artifact/v1alpha.ArtifactPublicServiceServer -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/protogen-go/app/app/v1alpha.AppPublicServiceServer -o ./ -s "_mock.gen.go"

// Ollama mock is generated in the source package to avoid import cycles.
//go:generate minimock -i github.com/instill-ai/pipeline-backend/pkg/component/ai/ollama/v0.OllamaClientInterface -o ../../ai/ollama/v0 -s "_mock.gen.go" -p ollama

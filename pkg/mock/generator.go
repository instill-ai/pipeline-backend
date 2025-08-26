package mock

//go:generate minimock -g -i github.com/instill-ai/pipeline-backend/pkg/repository.Repository -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/pipeline-backend/pkg/acl.ACLClient -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/pipeline-backend/pkg/service.Converter -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/protogen-go/core/mgmt/v1beta.MgmtPrivateServiceClient -o ./ -s "_mock.gen.go"

package store

import (
	"fmt"
	"sync"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/ai/anthropic/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/cohere/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/fireworksai/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/groq/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/huggingface/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/instill/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/mistralai/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/ollama/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/openai/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/stabilityai/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/universalai/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/asana/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/email/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/freshdesk/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/github/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/googlesearch/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/hubspot/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/jira/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/numbers/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/slack/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/whatsapp/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/bigquery/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/chroma/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/elasticsearch/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/googlecloudstorage/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/googledrive/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/instillartifact/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/milvus/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/mongodb/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/pinecone/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/qdrant/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/redis/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/sql/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/weaviate/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/zilliz/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/generic/collection/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/generic/restapi/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/audio/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/base64/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/document/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/image/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/json/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/text/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/video/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/web/v0"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

var (
	once      sync.Once
	compStore *Store
)

// Store holds in-memory information about the initialized components.
type Store struct {
	componentUIDs   []uuid.UUID
	componentUIDMap map[uuid.UUID]*component
	componentIDMap  map[string]*component
}

type component struct {
	comp base.IComponent
}

// ComponentSecrets contains the global config secrets of each
// implemented component (referenced by ID). Components may use these secrets
// to skip the component configuration step and have a ready-to-run
// config.
type ComponentSecrets map[string]map[string]any

// Init initializes the components implemented in this repository and loads
// their information to memory.
func Init(
	logger *zap.Logger,
	secrets ComponentSecrets,
	usageHandlerCreator base.UsageHandlerCreator,
) *Store {
	baseComp := base.Component{
		Logger:          logger,
		NewUsageHandler: usageHandlerCreator,
	}

	once.Do(func() {
		compStore = &Store{
			componentUIDMap: map[uuid.UUID]*component{},
			componentIDMap:  map[string]*component{},
		}
		compStore.Import(base64.Init(baseComp))
		compStore.Import(json.Init(baseComp))
		compStore.Import(image.Init(baseComp))
		compStore.Import(text.Init(baseComp))
		compStore.Import(document.Init(baseComp))
		compStore.Import(audio.Init(baseComp))
		compStore.Import(video.Init(baseComp))

		compStore.Import(github.Init(baseComp))
		{
			// StabilityAI
			conn := stabilityai.Init(baseComp)

			// Secret doesn't allow hyphens
			conn = conn.WithInstillCredentials(secrets["stabilityai"])
			compStore.Import(conn)
		}

		compStore.Import(instill.Init(baseComp))
		compStore.Import(huggingface.Init(baseComp))

		{
			// OpenAI
			conn := openai.Init(baseComp)
			conn = conn.WithInstillCredentials(secrets[conn.GetDefinitionID()])
			compStore.Import(conn)
		}
		{
			conn := universalai.Init(baseComp)
			// Please apply more keys when we add more vendors
			conn = conn.WithInstillCredentials("openai", secrets["openai"])
			compStore.Import(conn)
		}
		{
			// Anthropic
			conn := anthropic.Init(baseComp)
			conn = conn.WithInstillCredentials(secrets[conn.GetDefinitionID()])
			compStore.Import(conn)
		}
		{
			// Mistral
			conn := mistralai.Init(baseComp)
			// Secret doesn't allow hyphens
			conn = conn.WithInstillCredentials(secrets["mistralai"])
			compStore.Import(conn)
		}
		{
			// Cohere
			conn := cohere.Init(baseComp)
			conn = conn.WithInstillCredentials(secrets[conn.GetDefinitionID()])
			compStore.Import(conn)
		}
		{
			// FireworksAI
			conn := fireworksai.Init(baseComp)
			// Secret doesn't allow hyphens
			conn = conn.WithInstillCredentials(secrets["fireworksai"])
			compStore.Import(conn)
		}

		{
			// Groq
			conn := groq.Init(baseComp)
			conn = conn.WithInstillCredentials(secrets[conn.GetDefinitionID()])
			compStore.Import(conn)
		}

		{
			// Numbers
			conn := numbers.Init(baseComp)
			conn = conn.WithNumbersSecret(secrets[conn.GetDefinitionID()])
			compStore.Import(conn)
		}

		// compStore.Import(instillapp.Init(baseComp))
		compStore.Import(bigquery.Init(baseComp))
		compStore.Import(googlecloudstorage.Init(baseComp))
		compStore.Import(googlesearch.Init(baseComp))
		compStore.Import(pinecone.Init(baseComp))
		compStore.Import(redis.Init(baseComp))
		compStore.Import(elasticsearch.Init(baseComp))
		compStore.Import(mongodb.Init(baseComp))
		compStore.Import(sql.Init(baseComp))
		compStore.Import(weaviate.Init(baseComp))
		compStore.Import(milvus.Init(baseComp))
		compStore.Import(zilliz.Init(baseComp))
		compStore.Import(chroma.Init(baseComp))
		compStore.Import(qdrant.Init(baseComp))
		compStore.Import(instillartifact.Init(baseComp))
		compStore.Import(restapi.Init(baseComp))
		compStore.Import(collection.Init(baseComp))
		compStore.Import(web.Init(baseComp))
		compStore.Import(slack.Init(baseComp))
		compStore.Import(email.Init(baseComp))
		compStore.Import(jira.Init(baseComp))
		compStore.Import(ollama.Init(baseComp))
		compStore.Import(hubspot.Init(baseComp))
		compStore.Import(whatsapp.Init(baseComp))
		compStore.Import(freshdesk.Init(baseComp))
		compStore.Import(asana.Init(baseComp))
		compStore.Import(googledrive.Init(baseComp))
	})
	return compStore
}

// Import loads the component definitions into memory.
func (s *Store) Import(comp base.IComponent) {
	c := &component{comp: comp}
	s.componentUIDMap[comp.GetDefinitionUID()] = c
	s.componentIDMap[comp.GetDefinitionID()] = c
	s.componentUIDs = append(s.componentUIDs, comp.GetDefinitionUID())
}

// ExecutionParams contains the information needed to execute a
// component.
type ExecutionParams struct {
	// Component ID is the ID of the component *as defined in the recipe*.
	ComponentID string

	// ComponentDefinitionID determines the type of component to be executed.
	ComponentDefinitionID string

	// SystemVariables contains information about the pipeline trigger in which
	// the component is being executed.
	SystemVariables map[string]any

	// Setup may contain the configuration to connect to an external service.
	Setup *structpb.Struct

	// Task determines the task that the execution will carry out. It defines
	// the input and output of the execution.
	Task string
}

// CreateExecution initializes the execution of a component.
func (s *Store) CreateExecution(p ExecutionParams) (*base.ExecutionWrapper, error) {
	c, ok := s.componentIDMap[p.ComponentDefinitionID]
	if !ok {
		return nil, ErrComponentDefinitionNotFound
	}

	x, err := c.comp.CreateExecution(base.ComponentExecution{
		Component:       c.comp,
		ComponentID:     p.ComponentID,
		SystemVariables: p.SystemVariables,
		Setup:           p.Setup,
		Task:            p.Task,
	})
	if err != nil {
		return nil, fmt.Errorf("creating component execution: %w", err)
	}

	return &base.ExecutionWrapper{IExecution: x}, nil
}

func (s *Store) HandleVerificationEvent(defID string, header map[string][]string, req *structpb.Struct, setup map[string]any) (bool, *structpb.Struct, error) {
	if c, ok := s.componentIDMap[defID]; ok {
		return c.comp.HandleVerificationEvent(header, req, setup)
	}
	return false, nil, fmt.Errorf("component definition not found")
}

// GetDefinitionByUID returns a component definition by its UID.
func (s *Store) GetDefinitionByUID(defUID uuid.UUID, sysVars map[string]any, compConfig *base.ComponentConfig) (*pb.ComponentDefinition, error) {
	if c, ok := s.componentUIDMap[defUID]; ok {
		def, err := c.comp.GetDefinition(sysVars, compConfig)
		if err != nil {
			return nil, err
		}
		return proto.Clone(def).(*pb.ComponentDefinition), err
	}
	return nil, ErrComponentDefinitionNotFound
}

// GetDefinitionByID returns a component definition by its ID.
func (s *Store) GetDefinitionByID(defID string, sysVars map[string]any, compConfig *base.ComponentConfig) (*pb.ComponentDefinition, error) {
	if c, ok := s.componentIDMap[defID]; ok {
		def, err := c.comp.GetDefinition(sysVars, compConfig)
		if err != nil {
			return nil, err
		}
		return proto.Clone(def).(*pb.ComponentDefinition), err
	}
	return nil, ErrComponentDefinitionNotFound
}

// ListDefinitions returns all the loaded component definitions.
func (s *Store) ListDefinitions(sysVars map[string]any, returnTombstone bool) []*pb.ComponentDefinition {
	defs := []*pb.ComponentDefinition{}
	for _, uid := range s.componentUIDs {
		c := s.componentUIDMap[uid]
		def, err := c.comp.GetDefinition(sysVars, nil)
		if err == nil {
			if !def.Tombstone || returnTombstone {
				defs = append(defs, proto.Clone(def).(*pb.ComponentDefinition))
			}
		}
	}
	return defs
}

func (s *Store) IsSecretField(defUID uuid.UUID, target string) (bool, error) {
	if c, ok := s.componentUIDMap[defUID]; ok {
		return c.comp.IsSecretField(target), nil
	}
	return false, ErrComponentDefinitionNotFound
}

// ErrComponentDefinitionNotFound is returned when trying to access an
// inexistent component definition.
var ErrComponentDefinitionNotFound = fmt.Errorf("component definition not found")

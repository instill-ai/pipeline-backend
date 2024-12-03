package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	temporalclient "go.temporal.io/sdk/client"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/anthropic/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/cohere/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/fireworksai/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/groq/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/huggingface/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/instill/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/mistralai/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/ollama/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/openai/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/perplexity/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/stabilityai/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/ai/universalai/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/asana/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/email/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/freshdesk/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/github/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/googlesearch/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/hubspot/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/instillapp/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/jira/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/leadiq/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/numbers/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/slack/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/smartlead/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/application/whatsapp/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/bigquery/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/chroma/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/elasticsearch/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/googlecloudstorage/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/googledrive/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/googlesheets/v0"
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
	"github.com/instill-ai/pipeline-backend/pkg/component/generic/http/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/generic/scheduler/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/audio/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/base64/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/document/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/image/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/json/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/text/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/video/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/web/v0"
	"github.com/instill-ai/pipeline-backend/pkg/external"

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

type InitParams struct {
	Logger              *zap.Logger
	Secrets             config.ComponentSecrets
	UsageHandlerCreator base.UsageHandlerCreator
	BinaryFetcher       external.BinaryFetcher
	TemporalClient      temporalclient.Client
}

// Init initializes the components implemented in this repository and loads
// their information to memory.
func Init(param InitParams) *Store {
	baseComp := base.Component{
		Logger:          param.Logger,
		NewUsageHandler: param.UsageHandlerCreator,
		BinaryFetcher:   param.BinaryFetcher,
		TemporalClient:  param.TemporalClient,
	}
	secrets := param.Secrets

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
			// perplexity
			conn := perplexity.Init(baseComp)
			conn = conn.WithInstillCredentials(secrets[conn.GetDefinitionID()])
			compStore.Import(conn)
		}

		{
			// LeadIQ
			conn := leadiq.Init(baseComp)
			conn = conn.WithInstillCredentials(secrets[conn.GetDefinitionID()])
			compStore.Import(conn)
		}

		{
			// Numbers
			conn := numbers.Init(baseComp)
			conn = conn.WithNumbersSecret(secrets[conn.GetDefinitionID()])
			compStore.Import(conn)
		}

		compStore.Import(instillapp.Init(baseComp))
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
		compStore.Import(http.Init(baseComp))
		compStore.Import(collection.Init(baseComp))
		compStore.Import(scheduler.Init(baseComp))
		compStore.Import(web.Init(baseComp))
		{
			// GitHub
			conn := github.Init(baseComp)
			conn.WithOAuthConfig(secrets[conn.GetDefinitionID()])
			compStore.Import(conn)
		}
		{
			// Slack
			conn := slack.Init(baseComp)
			conn.WithOAuthConfig(secrets[conn.GetDefinitionID()])
			compStore.Import(conn)
		}
		{
			// Google Drive
			conn := googledrive.Init(baseComp)
			conn.WithOAuthConfig(secrets["googledrive"])
			compStore.Import(conn)
		}
		{
			// Google Sheets
			conn := googlesheets.Init(baseComp)
			conn.WithOAuthConfig(secrets["googlesheets"])
			compStore.Import(conn)
		}
		compStore.Import(email.Init(baseComp))
		compStore.Import(jira.Init(baseComp))
		compStore.Import(ollama.Init(baseComp))
		compStore.Import(hubspot.Init(baseComp))
		compStore.Import(whatsapp.Init(baseComp))
		compStore.Import(freshdesk.Init(baseComp))
		compStore.Import(asana.Init(baseComp))
		compStore.Import(smartlead.Init(baseComp))
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

func (s *Store) IdentifyEvent(ctx context.Context, defID string, rawEvent *base.RawEvent) (identifierResult *base.IdentifierResult, err error) {
	if c, ok := s.componentIDMap[defID]; ok {
		return c.comp.IdentifyEvent(ctx, rawEvent)
	}
	return nil, fmt.Errorf("component definition not found")
}

func (s *Store) ParseEvent(ctx context.Context, defID string, rawEvent *base.RawEvent) (parsedEvent *base.ParsedEvent, err error) {
	if c, ok := s.componentIDMap[defID]; ok {
		return c.comp.ParseEvent(ctx, rawEvent)
	}
	return nil, fmt.Errorf("component definition not found")
}

func (s *Store) RegisterEvent(ctx context.Context, defID string, settings *base.RegisterEventSettings) (identifiers []base.Identifier, err error) {
	if c, ok := s.componentIDMap[defID]; ok {
		return c.comp.RegisterEvent(ctx, settings)
	}
	return nil, fmt.Errorf("component definition not found")
}

func (s *Store) UnregisterEvent(ctx context.Context, defID string, settings *base.UnregisterEventSettings, identifiers []base.Identifier) error {
	if c, ok := s.componentIDMap[defID]; ok {
		return c.comp.UnregisterEvent(ctx, settings, identifiers)
	}
	return fmt.Errorf("component definition not found")
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

// IsSecretField checks whether a property in a component definition is a
// secret field.
func (s *Store) IsSecretField(defUID uuid.UUID, target string) (bool, error) {
	c, ok := s.componentUIDMap[defUID]
	if !ok {
		return false, ErrComponentDefinitionNotFound
	}

	return c.comp.IsSecretField(target), nil
}

// SupportsOAuth checks whether a component supports OAuth connections.
func (s *Store) SupportsOAuth(defUID uuid.UUID) (bool, error) {
	c, ok := s.componentUIDMap[defUID]
	if !ok {
		return false, ErrComponentDefinitionNotFound
	}

	return c.comp.SupportsOAuth(), nil
}

// ErrComponentDefinitionNotFound is returned when trying to access an
// inexistent component definition.
var ErrComponentDefinitionNotFound = fmt.Errorf("component definition not found")

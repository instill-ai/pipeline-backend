package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.einride.tech/aip/filtering"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/x/paginate"

	component "github.com/instill-ai/pipeline-backend/pkg/component/store"
	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func (s *service) GetOperatorDefinitionByID(ctx context.Context, id string) (*pb.OperatorDefinition, error) {
	compDef, err := s.getComponentDefinitionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	switch compDef.Type {
	case pb.ComponentType_COMPONENT_TYPE_OPERATOR:
		return convertComponentDefToOperatorDef(compDef), nil
	default:
		return nil, errdomain.ErrNotFound
	}
}

func (s *service) implementedOperatorDefinitions(ctx context.Context) ([]*pb.OperatorDefinition, error) {

	allDefs := s.component.ListDefinitions(nil, false)

	implemented := make([]*pb.OperatorDefinition, 0, len(allDefs))
	for _, def := range allDefs {
		if def.Type == pb.ComponentType_COMPONENT_TYPE_OPERATOR {
			if implementedReleaseStages[def.GetReleaseStage()] {
				implemented = append(implemented, convertComponentDefToOperatorDef(def))
			}
		}
	}

	return implemented, nil
}

func (s *service) ListOperatorDefinitions(ctx context.Context, req *pb.ListOperatorDefinitionsRequest) (*pb.ListOperatorDefinitionsResponse, error) {
	pageSize := s.pageSizeInRange(req.GetPageSize())
	prevLastUID, err := s.lastUIDFromToken(req.GetPageToken())
	if err != nil {
		return nil, err
	}

	// The client of this use case is the console pipeline builder, so we want
	// to filter out the unimplemented definitions (that are present in the
	// ListComponentDefinitions method, used also for the marketing website).
	//
	// TODO we can use only the component definition list and let the clients
	// do the filtering in the query params.
	defs, err := s.implementedOperatorDefinitions(ctx)
	if err != nil {
		return nil, err
	}

	startIdx := 0
	lastUID := ""
	for idx, def := range defs {
		if def.Uid == prevLastUID {
			startIdx = idx + 1
			break
		}
	}
	page := make([]*pb.OperatorDefinition, 0, pageSize)
	for i := 0; i < pageSize && startIdx+i < len(defs); i++ {
		def := proto.Clone(defs[startIdx+i]).(*pb.OperatorDefinition)
		page = append(page, def)
		lastUID = def.Uid
	}

	nextPageToken := ""

	if startIdx+len(page) < len(defs) {
		nextPageToken = paginate.EncodeToken(time.Time{}, lastUID)
	}

	for _, def := range page {
		s.applyViewToOperatorDefinition(def, req.GetView())
	}

	resp := &pb.ListOperatorDefinitionsResponse{
		NextPageToken:       nextPageToken,
		TotalSize:           int32(len(page)),
		OperatorDefinitions: page,
	}

	return resp, nil
}

func (s *service) filterConnectorDefinitions(defs []*pb.ConnectorDefinition, filter filtering.Filter) []*pb.ConnectorDefinition {
	if filter.CheckedExpr == nil {
		return defs
	}

	filtered := make([]*pb.ConnectorDefinition, 0, len(defs))
	expr, _ := s.repository.TranspileFilter(filter)
	typeMap := map[string]bool{}
	for i, v := range expr.Vars {
		if i == 0 {
			typeMap[string(v.(protoreflect.Name))] = true
			continue
		}

		typeMap[string(v.([]any)[0].(protoreflect.Name))] = true
	}

	for _, def := range defs {
		if _, ok := typeMap[def.Type.String()]; ok {
			filtered = append(filtered, def)
		}
	}

	return filtered
}

func (s *service) lastUIDFromToken(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	_, id, err := paginate.DecodeToken(token)
	if err != nil {
		return "", fmt.Errorf("%w: invalid page token: %w", errdomain.ErrInvalidArgument, err)
	}

	return id, nil
}

func (s *service) pageSizeInRange(pageSize int32) int {
	if pageSize <= 0 {
		return repository.DefaultPageSize
	}

	if pageSize > repository.MaxPageSize {
		return repository.MaxPageSize
	}

	return int(pageSize)
}

func (s *service) pageInRange(page int32) int {
	if page <= 0 {
		return 0
	}

	return int(page)
}

func (s *service) applyViewToConnectorDefinition(cd *pb.ConnectorDefinition, v pb.ComponentDefinition_View) {
	cd.VendorAttributes = nil
	if v <= pb.ComponentDefinition_VIEW_BASIC {
		cd.Spec = nil
	}
}

func (s *service) applyViewToOperatorDefinition(od *pb.OperatorDefinition, v pb.ComponentDefinition_View) {
	od.Name = fmt.Sprintf("operator-definitions/%s", od.Id)
	if v <= pb.ComponentDefinition_VIEW_BASIC {
		od.Spec = nil
	}
}

// ListComponentDefinitions returns a paginated list of components.
func (s *service) ListComponentDefinitions(ctx context.Context, req *pb.ListComponentDefinitionsRequest) (*pb.ListComponentDefinitionsResponse, error) {
	pageSize := s.pageSizeInRange(req.GetPageSize())
	page := s.pageInRange(req.GetPage())

	var compType pb.ComponentType
	var releaseStage pb.ComponentDefinition_ReleaseStage
	declarations, err := filtering.NewDeclarations(
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("qTitle", filtering.TypeString),
		filtering.DeclareEnumIdent("releaseStage", releaseStage.Type()),
		filtering.DeclareEnumIdent("componentType", compType.Type()),
	)
	if err != nil {
		return nil, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}

	p := repository.ListComponentDefinitionsParams{
		Offset: page * pageSize,
		Limit:  pageSize,
		Filter: filter,
	}

	uids, totalSize, err := s.repository.ListComponentDefinitionUIDs(ctx, p)
	if err != nil {
		return nil, err
	}

	defs := make([]*pb.ComponentDefinition, len(uids))

	vars, err := recipe.GenerateSystemVariables(ctx, recipe.SystemVariables{})
	if err != nil {
		return nil, err
	}

	for i, uid := range uids {
		def, err := s.component.GetDefinitionByUID(uid.UID, vars, nil)
		if err != nil {
			return nil, err
		}
		if req.GetView() != pb.ComponentDefinition_VIEW_FULL {
			def.Spec = nil
		}

		// Temporary solution to filter out the instill-app component for local-ce:dev edition
		// In the future the non-CE definitions shouldn't be in the CE repositories and Cloud should extend the CE component store to initialize the private components.
		if skipComponentInCE(def.GetId()) {
			continue
		}

		defs[i] = def
	}

	resp := &pb.ListComponentDefinitionsResponse{
		PageSize:             int32(pageSize),
		Page:                 int32(page),
		TotalSize:            int32(totalSize),
		ComponentDefinitions: defs,
	}

	return resp, nil
}

var skipComponentsInCE = map[string]bool{
	"instill-app": true,
}

func skipComponentInCE(componentID string) bool {
	if skipComponentsInCE[componentID] && config.Config.Server.Edition == "local-ce:dev" {
		return true
	}
	return false
}

var implementedReleaseStages = map[pb.ComponentDefinition_ReleaseStage]bool{
	pb.ComponentDefinition_RELEASE_STAGE_ALPHA: true,
	pb.ComponentDefinition_RELEASE_STAGE_BETA:  true,
	pb.ComponentDefinition_RELEASE_STAGE_GA:    true,
}

func (s *service) implementedConnectorDefinitions(ctx context.Context) ([]*pb.ConnectorDefinition, error) {

	allDefs := s.component.ListDefinitions(nil, false)

	implemented := make([]*pb.ConnectorDefinition, 0, len(allDefs))
	for _, def := range allDefs {
		if def.Type == pb.ComponentType_COMPONENT_TYPE_AI || def.Type == pb.ComponentType_COMPONENT_TYPE_DATA ||
			def.Type == pb.ComponentType_COMPONENT_TYPE_APPLICATION || def.Type == pb.ComponentType_COMPONENT_TYPE_GENERIC {
			if implementedReleaseStages[def.GetReleaseStage()] {
				implemented = append(implemented, convertComponentDefToConnectorDef(def))
			}
		}
	}

	return implemented, nil
}

func (s *service) ListConnectorDefinitions(ctx context.Context, req *pb.ListConnectorDefinitionsRequest) (*pb.ListConnectorDefinitionsResponse, error) {
	pageSize := s.pageSizeInRange(req.GetPageSize())
	prevLastUID, err := s.lastUIDFromToken(req.GetPageToken())
	if err != nil {
		return nil, err
	}

	var connType pb.ConnectorType
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareEnumIdent("connectorType", connType.Type()),
	}...)
	if err != nil {
		return nil, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}

	// The client of this use case is the console pipeline builder, so we want
	// to filter out the unimplemented definitions (that are present in the
	// ListComponentDefinitions method, used also for the marketing website).
	//
	// TODO we can use only the component definition list and let the clients
	// do the filtering in the query params.
	defs, err := s.implementedConnectorDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	defs = s.filterConnectorDefinitions(defs, filter)

	startIdx := 0
	lastUID := ""
	for idx, def := range defs {
		if def.Uid == prevLastUID {
			startIdx = idx + 1
			break
		}
	}

	page := make([]*pb.ConnectorDefinition, 0, pageSize)
	for i := 0; i < pageSize && startIdx+i < len(defs); i++ {
		def := proto.Clone(defs[startIdx+i]).(*pb.ConnectorDefinition)
		page = append(page, def)
		lastUID = def.Uid
	}

	nextPageToken := ""

	if startIdx+len(page) < len(defs) {
		nextPageToken = paginate.EncodeToken(time.Time{}, lastUID)
	}

	pageDefs := make([]*pb.ConnectorDefinition, 0, len(page))
	for _, def := range page {
		// Due to the instill-model requiring dynamic definition, we need to set
		// the definition with dynamic values by fetching it via system
		// variables.
		if def.Id == "instill-model" {
			vars, err := recipe.GenerateSystemVariables(ctx, recipe.SystemVariables{})
			if err != nil {
				return nil, err
			}
			updatedDef, err := s.component.GetDefinitionByID(def.Id, vars, nil)
			if err == nil {
				def = convertComponentDefToConnectorDef(updatedDef)
			}
		}
		s.applyViewToConnectorDefinition(def, req.GetView())
		pageDefs = append(pageDefs, def)
	}

	return &pb.ListConnectorDefinitionsResponse{
		ConnectorDefinitions: pageDefs,
		NextPageToken:        nextPageToken,
		TotalSize:            int32(len(defs)),
	}, nil
}

func (s *service) GetConnectorDefinitionByID(ctx context.Context, id string) (*pb.ConnectorDefinition, error) {
	compDef, err := s.getComponentDefinitionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	switch compDef.Type {
	case pb.ComponentType_COMPONENT_TYPE_AI,
		pb.ComponentType_COMPONENT_TYPE_DATA,
		pb.ComponentType_COMPONENT_TYPE_APPLICATION,
		pb.ComponentType_COMPONENT_TYPE_GENERIC:

		return convertComponentDefToConnectorDef(compDef), nil
	default:
		return nil, errdomain.ErrNotFound
	}

}

func convertComponentDefToConnectorDef(compDef *pb.ComponentDefinition) *pb.ConnectorDefinition {

	return &pb.ConnectorDefinition{
		Id:               compDef.Id,
		Uid:              compDef.Uid,
		Name:             fmt.Sprintf("connector-definitions/%s", strings.Split(compDef.Name, "/")[1]),
		Title:            compDef.Title,
		DocumentationUrl: compDef.DocumentationUrl,
		Icon:             compDef.Icon,
		Spec:             (*pb.ConnectorSpec)(compDef.Spec),
		Type: func(c *pb.ComponentDefinition) pb.ConnectorType {
			switch c.Type {
			case pb.ComponentType_COMPONENT_TYPE_AI:
				return pb.ConnectorType_CONNECTOR_TYPE_AI
			case pb.ComponentType_COMPONENT_TYPE_DATA:
				return pb.ConnectorType_CONNECTOR_TYPE_DATA
			case pb.ComponentType_COMPONENT_TYPE_APPLICATION:
				return pb.ConnectorType_CONNECTOR_TYPE_APPLICATION
			case pb.ComponentType_COMPONENT_TYPE_GENERIC:
				return pb.ConnectorType_CONNECTOR_TYPE_GENERIC
			}
			return pb.ConnectorType_CONNECTOR_TYPE_UNSPECIFIED

		}(compDef),
		Tombstone:        compDef.Tombstone,
		Public:           compDef.Public,
		Custom:           compDef.Custom,
		SourceUrl:        compDef.SourceUrl,
		Version:          compDef.Version,
		Tasks:            compDef.Tasks,
		Description:      compDef.Description,
		ReleaseStage:     compDef.ReleaseStage,
		Vendor:           compDef.Vendor,
		VendorAttributes: compDef.VendorAttributes,
	}

}

func convertComponentDefToOperatorDef(compDef *pb.ComponentDefinition) *pb.OperatorDefinition {

	return &pb.OperatorDefinition{
		Id:               compDef.Id,
		Uid:              compDef.Uid,
		Name:             fmt.Sprintf("operator-definitions/%s", strings.Split(compDef.Name, "/")[1]),
		Title:            compDef.Title,
		DocumentationUrl: compDef.DocumentationUrl,
		Icon:             compDef.Icon,
		Spec:             (*pb.OperatorSpec)(compDef.Spec),
		Tombstone:        compDef.Tombstone,
		Public:           compDef.Public,
		Custom:           compDef.Custom,
		SourceUrl:        compDef.SourceUrl,
		Version:          compDef.Version,
		Tasks:            compDef.Tasks,
		Description:      compDef.Description,
		ReleaseStage:     compDef.ReleaseStage,
	}

}

func (s *service) getComponentDefinitionByID(ctx context.Context, id string) (*pb.ComponentDefinition, error) {
	vars, err := recipe.GenerateSystemVariables(ctx, recipe.SystemVariables{})
	if err != nil {
		return nil, err
	}

	cd, err := s.component.GetDefinitionByID(id, vars, nil)
	if err != nil {
		if errors.Is(err, component.ErrComponentDefinitionNotFound) {
			err = fmt.Errorf("fetching component definition: %w", errdomain.ErrNotFound)
		}
		return nil, err
	}

	return cd, nil
}

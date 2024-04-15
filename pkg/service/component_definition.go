package service

import (
	"context"
	"fmt"
	"time"

	"go.einride.tech/aip/filtering"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/paginate"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func (s *service) GetOperatorDefinitionByID(ctx context.Context, defID string) (*pipelinePB.OperatorDefinition, error) {
	return s.operator.GetOperatorDefinitionByID(defID, nil)
}

func (s *service) implementedOperatorDefinitions() []*pipelinePB.OperatorDefinition {
	allDefs := s.operator.ListOperatorDefinitions()

	implemented := make([]*pipelinePB.OperatorDefinition, 0, len(allDefs))
	for _, def := range allDefs {
		if implementedReleaseStages[def.GetReleaseStage()] {
			implemented = append(implemented, def)
		}
	}

	return implemented
}

func (s *service) ListOperatorDefinitions(ctx context.Context, req *pipelinePB.ListOperatorDefinitionsRequest) (*pipelinePB.ListOperatorDefinitionsResponse, error) {
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
	defs := s.implementedOperatorDefinitions()

	startIdx := 0
	lastUID := ""
	for idx, def := range defs {
		if def.Uid == prevLastUID {
			startIdx = idx + 1
			break
		}
	}
	page := make([]*pipelinePB.OperatorDefinition, 0, pageSize)
	for i := 0; i < pageSize && startIdx+i < len(defs); i++ {
		def := proto.Clone(defs[startIdx+i]).(*pipelinePB.OperatorDefinition)
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

	resp := &pipelinePB.ListOperatorDefinitionsResponse{
		NextPageToken:       nextPageToken,
		TotalSize:           int32(len(page)),
		OperatorDefinitions: page,
	}

	return resp, nil
}

func (s *service) RemoveCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct) {
	utils.RemoveCredentialFieldsWithMaskString(s.connector, dbConnDefID, config)
}

func (s *service) KeepCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct) {
	utils.KeepCredentialFieldsWithMaskString(s.connector, dbConnDefID, config)
}

func (s *service) filterConnectorDefinitions(defs []*pipelinePB.ConnectorDefinition, filter filtering.Filter) []*pipelinePB.ConnectorDefinition {
	if filter.CheckedExpr == nil {
		return defs
	}

	filtered := make([]*pipelinePB.ConnectorDefinition, 0, len(defs))
	trans := repository.NewTranspiler(filter)
	expr, _ := trans.Transpile()
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
		return "", repository.ErrPageTokenDecode
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

func (s *service) applyViewToConnectorDefinition(cd *pipelinePB.ConnectorDefinition, v pipelinePB.ComponentDefinition_View) {
	cd.VendorAttributes = nil
	if v == pipelinePB.ComponentDefinition_VIEW_BASIC {
		cd.Spec = nil
	}
}

func (s *service) applyViewToOperatorDefinition(od *pipelinePB.OperatorDefinition, v pipelinePB.ComponentDefinition_View) {
	od.Name = fmt.Sprintf("operator-definitions/%s", od.Id)
	if v == pipelinePB.ComponentDefinition_VIEW_BASIC {
		od.Spec = nil
	}
}

// ListComponentDefinitions returns a paginated list of components.
func (s *service) ListComponentDefinitions(ctx context.Context, req *pipelinePB.ListComponentDefinitionsRequest) (*pipelinePB.ListComponentDefinitionsResponse, error) {
	pageSize := s.pageSizeInRange(req.GetPageSize())
	page := s.pageInRange(req.GetPage())

	var compType pipelinePB.ComponentType
	var releaseStage pipelinePB.ComponentDefinition_ReleaseStage
	declarations, err := filtering.NewDeclarations(
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("q_title", filtering.TypeString),
		filtering.DeclareEnumIdent("release_stage", releaseStage.Type()),
		filtering.DeclareEnumIdent("component_type", compType.Type()),
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

	defs := make([]*pipelinePB.ComponentDefinition, len(uids))

	for i, uid := range uids {
		d := &pipelinePB.ComponentDefinition{
			Type: pipelinePB.ComponentType(uid.ComponentType),
		}

		switch d.Type {
		case pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI,
			pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_APPLICATION,
			pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA:

			cd, err := s.connector.GetConnectorDefinitionByUID(uid.UID, nil, nil)
			if err != nil {
				return nil, err
			}

			cd = proto.Clone(cd).(*pipelinePB.ConnectorDefinition)
			s.applyViewToConnectorDefinition(cd, *req.View)
			d.Definition = &pipelinePB.ComponentDefinition_ConnectorDefinition{
				ConnectorDefinition: cd,
			}
		case pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR:
			od, err := s.operator.GetOperatorDefinitionByUID(uid.UID, nil)
			if err != nil {
				return nil, err
			}

			od = proto.Clone(od).(*pipelinePB.OperatorDefinition)
			s.applyViewToOperatorDefinition(od, *req.View)
			d.Definition = &pipelinePB.ComponentDefinition_OperatorDefinition{
				OperatorDefinition: od,
			}
		default:
			return nil, fmt.Errorf("invalid component definition type in database")
		}

		defs[i] = d
	}

	resp := &pipelinePB.ListComponentDefinitionsResponse{
		PageSize:             int32(pageSize),
		Page:                 int32(page),
		TotalSize:            int32(totalSize),
		ComponentDefinitions: defs,
	}

	return resp, nil
}

var implementedReleaseStages = map[pipelinePB.ComponentDefinition_ReleaseStage]bool{
	pipelinePB.ComponentDefinition_RELEASE_STAGE_ALPHA: true,
	pipelinePB.ComponentDefinition_RELEASE_STAGE_BETA:  true,
	pipelinePB.ComponentDefinition_RELEASE_STAGE_GA:    true,
}

func (s *service) implementedConnectorDefinitions() []*pipelinePB.ConnectorDefinition {
	allDefs := s.connector.ListConnectorDefinitions()

	implemented := make([]*pipelinePB.ConnectorDefinition, 0, len(allDefs))
	for _, def := range allDefs {
		if implementedReleaseStages[def.GetReleaseStage()] {
			implemented = append(implemented, def)
		}
	}

	return implemented
}

func (s *service) ListConnectorDefinitions(ctx context.Context, req *pipelinePB.ListConnectorDefinitionsRequest) (*pipelinePB.ListConnectorDefinitionsResponse, error) {
	pageSize := s.pageSizeInRange(req.GetPageSize())
	prevLastUID, err := s.lastUIDFromToken(req.GetPageToken())
	if err != nil {
		return nil, err
	}

	var connType pipelinePB.ConnectorType
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareEnumIdent("connector_type", connType.Type()),
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
	defs := s.filterConnectorDefinitions(s.implementedConnectorDefinitions(), filter)

	startIdx := 0
	lastUID := ""
	for idx, def := range defs {
		if def.Uid == prevLastUID {
			startIdx = idx + 1
			break
		}
	}

	page := make([]*pipelinePB.ConnectorDefinition, 0, pageSize)
	for i := 0; i < pageSize && startIdx+i < len(defs); i++ {
		def := proto.Clone(defs[startIdx+i]).(*pipelinePB.ConnectorDefinition)
		page = append(page, def)
		lastUID = def.Uid
	}

	nextPageToken := ""

	if startIdx+len(page) < len(defs) {
		nextPageToken = paginate.EncodeToken(time.Time{}, lastUID)
	}

	pageDefs := make([]*pipelinePB.ConnectorDefinition, 0, len(page))
	for _, def := range page {
		s.applyViewToConnectorDefinition(def, *req.View)
		pageDefs = append(pageDefs, def)
	}

	return &pipelinePB.ListConnectorDefinitionsResponse{
		ConnectorDefinitions: pageDefs,
		NextPageToken:        nextPageToken,
		TotalSize:            int32(len(defs)),
	}, nil
}

func (s *service) GetConnectorDefinitionByID(ctx context.Context, id string, view pipelinePB.ComponentDefinition_View) (*pipelinePB.ConnectorDefinition, error) {

	def, err := s.connector.GetConnectorDefinitionByID(id, nil, nil)
	if err != nil {
		return nil, err
	}
	def = proto.Clone(def).(*pipelinePB.ConnectorDefinition)
	if view == pipelinePB.ComponentDefinition_VIEW_BASIC {
		def.Spec = nil
	}
	def.VendorAttributes = nil

	return def, nil
}

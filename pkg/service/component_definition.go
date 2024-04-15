package service

import (
	"context"
	"fmt"
	"time"

	"go.einride.tech/aip/filtering"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/x/paginate"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func (s *service) GetOperatorDefinitionByID(ctx context.Context, defID string) (*pb.OperatorDefinition, error) {
	return s.operator.GetOperatorDefinitionByID(defID, nil)
}

func (s *service) implementedOperatorDefinitions() []*pb.OperatorDefinition {
	allDefs := s.operator.ListOperatorDefinitions()

	implemented := make([]*pb.OperatorDefinition, 0, len(allDefs))
	for _, def := range allDefs {
		if implementedReleaseStages[def.GetReleaseStage()] {
			implemented = append(implemented, def)
		}
	}

	return implemented
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
	defs := s.implementedOperatorDefinitions()

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

func (s *service) applyViewToConnectorDefinition(cd *pb.ConnectorDefinition, v pb.ComponentDefinition_View) {
	cd.VendorAttributes = nil
	if v == pb.ComponentDefinition_VIEW_BASIC {
		cd.Spec = nil
	}
}

func (s *service) applyViewToOperatorDefinition(od *pb.OperatorDefinition, v pb.ComponentDefinition_View) {
	od.Name = fmt.Sprintf("operator-definitions/%s", od.Id)
	if v == pb.ComponentDefinition_VIEW_BASIC {
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

	defs := make([]*pb.ComponentDefinition, len(uids))

	for i, uid := range uids {
		d := &pb.ComponentDefinition{
			Type: pb.ComponentType(uid.ComponentType),
		}

		switch d.Type {
		case pb.ComponentType_COMPONENT_TYPE_CONNECTOR_AI,
			pb.ComponentType_COMPONENT_TYPE_CONNECTOR_APPLICATION,
			pb.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA:

			cd, err := s.connector.GetConnectorDefinitionByUID(uid.UID, nil)
			if err != nil {
				return nil, err
			}

			cd = proto.Clone(cd).(*pb.ConnectorDefinition)
			s.applyViewToConnectorDefinition(cd, req.GetView())
			d.Definition = &pb.ComponentDefinition_ConnectorDefinition{
				ConnectorDefinition: cd,
			}
		case pb.ComponentType_COMPONENT_TYPE_OPERATOR:
			od, err := s.operator.GetOperatorDefinitionByUID(uid.UID, nil)
			if err != nil {
				return nil, err
			}

			od = proto.Clone(od).(*pb.OperatorDefinition)
			s.applyViewToOperatorDefinition(od, req.GetView())
			d.Definition = &pb.ComponentDefinition_OperatorDefinition{
				OperatorDefinition: od,
			}
		default:
			return nil, fmt.Errorf("invalid component definition type in database")
		}

		defs[i] = d
	}

	resp := &pb.ListComponentDefinitionsResponse{
		PageSize:             int32(pageSize),
		Page:                 int32(page),
		TotalSize:            int32(totalSize),
		ComponentDefinitions: defs,
	}

	return resp, nil
}

var implementedReleaseStages = map[pb.ComponentDefinition_ReleaseStage]bool{
	pb.ComponentDefinition_RELEASE_STAGE_ALPHA: true,
	pb.ComponentDefinition_RELEASE_STAGE_BETA:  true,
	pb.ComponentDefinition_RELEASE_STAGE_GA:    true,
}

func (s *service) implementedConnectorDefinitions() []*pb.ConnectorDefinition {
	allDefs := s.connector.ListConnectorDefinitions()

	implemented := make([]*pb.ConnectorDefinition, 0, len(allDefs))
	for _, def := range allDefs {
		if implementedReleaseStages[def.GetReleaseStage()] {
			implemented = append(implemented, def)
		}
	}

	return implemented
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
	return s.connector.GetConnectorDefinitionByID(id, nil)
}

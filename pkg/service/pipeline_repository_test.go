//go:build dbtest
// +build dbtest

package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/gofrs/uuid"
	"github.com/gojuno/minimock/v3"
	"go.einride.tech/aip/filtering"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc/metadata"
	"gopkg.in/guregu/null.v4"
	"gorm.io/gorm"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/mock"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	if err := config.Init("../../config/config.yaml"); err != nil {
		panic(err)
	}

	db = database.GetSharedConnection()
	defer database.Close(db)

	os.Exit(m.Run())
}

func TestService_ListPipelineRuns(t *testing.T) {
	c := qt.New(t)
	mc := minimock.NewController(t)

	mockUIDs := make([]uuid.UUID, 4)
	for i := range len(mockUIDs) {
		mockUIDs[i] = uuid.Must(uuid.NewV4())
	}
	ownerUID := mockUIDs[0].String()
	user2 := mockUIDs[1].String()
	namespace1 := mockUIDs[2].String()
	pipelineUID := mockUIDs[3]

	t0 := time.Now()
	ownerPermalink := "users/" + ownerUID
	pipelineID := "pipelineID-test"

	testCases := []struct {
		description   string
		runner        string
		runNamespace  string
		viewer        string
		viewNamespace string
		canView       bool
	}{
		{
			description:   "can view logs when view ns is resource owner ns or credit owner ns",
			runner:        ownerUID,
			runNamespace:  ownerUID,
			viewer:        ownerUID,
			viewNamespace: ownerUID,
			canView:       true,
		},
		{
			description:   "cannot view logs when view ns is neither resource owner ns nor credit owner",
			runner:        ownerUID,
			runNamespace:  ownerUID,
			viewer:        ownerUID,
			viewNamespace: namespace1,
			canView:       false,
		},
		{
			description:   "cannot view logs when view ns is neither resource owner ns nor credit owner",
			runner:        ownerUID,
			runNamespace:  ownerUID,
			viewer:        user2,
			viewNamespace: user2,
			canView:       false,
		},
		{
			description:   "cannot view logs when view ns is neither resource owner ns nor credit owner",
			runner:        ownerUID,
			runNamespace:  ownerUID,
			viewer:        user2,
			viewNamespace: namespace1,
			canView:       false,
		},
		{
			description:   "can view logs when view ns is resource owner ns",
			runner:        ownerUID,
			runNamespace:  namespace1,
			viewer:        ownerUID,
			viewNamespace: ownerUID,
			canView:       true,
		},
		{
			description:   "can view logs when view ns is credit owner",
			runner:        ownerUID,
			runNamespace:  namespace1,
			viewer:        ownerUID,
			viewNamespace: namespace1,
			canView:       true,
		},
		{
			description:   "cannot view logs when view ns is neither resource owner ns nor credit owner",
			runner:        ownerUID,
			runNamespace:  namespace1,
			viewer:        user2,
			viewNamespace: user2,
			canView:       false,
		},
		{
			description:   "can view logs when view ns is credit owner",
			runner:        ownerUID,
			runNamespace:  namespace1,
			viewer:        user2,
			viewNamespace: namespace1,
			canView:       true,
		},
		{
			description:   "can view logs when view ns is resource owner ns",
			runner:        user2,
			runNamespace:  user2,
			viewer:        ownerUID,
			viewNamespace: ownerUID,
			canView:       true,
		},
		{
			description:   "cannot view logs when view ns is neither resource owner ns nor credit owner",
			runner:        user2,
			runNamespace:  user2,
			viewer:        ownerUID,
			viewNamespace: namespace1,
			canView:       false,
		},
	}

	redisClient, _ := redismock.NewClientMock()

	var temporalClient client.Client

	aclClient := mock.NewACLClientInterfaceMock(mc)

	mockConverter := mock.NewConverterMock(mc)
	mgmtPrivateClient := mock.NewMgmtPrivateServiceClientMock(mc)
	mgmtPrivateClient.CheckNamespaceAdminMock.Return(&mgmtpb.CheckNamespaceAdminResponse{
		Type: mgmtpb.CheckNamespaceAdminResponse_NAMESPACE_USER,
		Uid:  ownerUID,
	}, nil)
	mgmtPrivateClient.CheckNamespaceByUIDAdminMock.Return(&mgmtpb.CheckNamespaceByUIDAdminResponse{
		Type:  0,
		Id:    "test-user",
		Owner: nil,
	}, nil)

	mockMinio := mock.NewMinioIMock(mc)
	mockMinio.GetFilesByPathsMock.Return(nil, fmt.Errorf("some errors"))

	for i, testCase := range testCases {
		c.Run(fmt.Sprintf("get pipeline run with permissions test case %d %s", i+1, testCase.description), func(c *qt.C) {

			tx := db.Begin()
			c.Cleanup(func() { tx.Rollback() })

			repo := repository.NewRepository(tx, redisClient)

			svc := NewService(
				repo,
				redisClient,
				temporalClient,
				aclClient,
				mockConverter,
				mgmtPrivateClient,
				mockMinio,
				nil,
				nil,
				uuid.UUID{},
			)

			ctx := context.Background()

			p := &datamodel.Pipeline{
				Owner: ownerPermalink,
				ID:    pipelineID,
				BaseDynamic: datamodel.BaseDynamic{
					UID:        pipelineUID,
					CreateTime: t0,
					UpdateTime: t0,
				},
			}
			err := repo.CreateNamespacePipeline(ctx, p)
			c.Assert(err, qt.IsNil)

			got, err := repo.GetNamespacePipelineByID(ctx, ownerPermalink, pipelineID, true, false)
			c.Assert(err, qt.IsNil)
			c.Check(got.NumberOfRuns, qt.Equals, 0)
			c.Check(got.LastRunTime.IsZero(), qt.IsTrue)
			c.Check(got.OwnerUID().String(), qt.Equals, ownerUID)

			pipelineRun := &datamodel.PipelineRun{
				PipelineTriggerUID: uuid.Must(uuid.NewV4()),
				PipelineUID:        got.UID,
				Status:             datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
				Source:             datamodel.RunSource(runpb.RunSource_RUN_SOURCE_API),
				TriggeredBy:        testCase.runner,
				Namespace:          testCase.runNamespace,
				StartedTime:        time.Now(),
				TotalDuration:      null.IntFrom(42),
				Components:         nil,
			}

			err = repo.UpsertPipelineRun(ctx, pipelineRun)
			c.Assert(err, qt.IsNil)

			m := make(map[string]string)
			m[constant.HeaderRequesterUIDKey] = testCase.viewNamespace
			m[constant.HeaderUserUIDKey] = testCase.viewer

			ctxWithHeader := metadata.NewIncomingContext(context.Background(), metadata.New(m))
			req := &pb.ListPipelineRunsRequest{
				NamespaceId: ownerUID,
				PipelineId:  pipelineID,
				Page:        0,
				PageSize:    10,
			}
			runs, err := svc.ListPipelineRuns(ctxWithHeader, req, filtering.Filter{})
			c.Assert(err, qt.IsNil)
			if testCase.canView {
				c.Check(len(runs.PipelineRuns), qt.Equals, 1)
			} else {
				c.Check(len(runs.PipelineRuns), qt.Equals, 0)
			}
		})
	}
}

func TestService_ListPipelineRuns_OrgResource(t *testing.T) {
	c := qt.New(t)
	mc := minimock.NewController(t)

	mockUIDs := make([]uuid.UUID, 6)
	for i := range len(mockUIDs) {
		mockUIDs[i] = uuid.Must(uuid.NewV4())
	}
	orgUID := mockUIDs[0].String()
	user1 := mockUIDs[1].String()
	namespace1 := mockUIDs[2].String()
	pipelineUID := mockUIDs[3]
	user2 := mockUIDs[4].String()
	user3 := mockUIDs[5].String()

	t0 := time.Now()
	ownerPermalink := "organizations/" + orgUID
	pipelineID := "pipelineID-test"

	testCases := []struct {
		description   string
		runner        string
		runNamespace  string
		viewer        string
		viewNamespace string
		canView       bool
	}{
		{
			description:   "can view logs when view ns is credit owner",
			runner:        user1,
			runNamespace:  user1,
			viewer:        user1,
			viewNamespace: user1,
			canView:       true,
		},
		{
			description:   "can view logs when view ns is resource owner ns",
			runner:        user1,
			runNamespace:  user1,
			viewer:        user1,
			viewNamespace: orgUID,
			canView:       true,
		},
		{
			description:   "can view logs when view ns is resource owner ns or credit owner ns",
			runner:        user1,
			runNamespace:  orgUID,
			viewer:        user1,
			viewNamespace: orgUID,
			canView:       true,
		},
		{
			description:   "cannot view logs when view ns is neither resource owner ns nor credit owner",
			runner:        user1,
			runNamespace:  orgUID,
			viewer:        user1,
			viewNamespace: user1,
			canView:       false,
		},
		{
			description:   "can view logs when view ns is resource owner ns or credit owner ns",
			runner:        user1,
			runNamespace:  orgUID,
			viewer:        user2,
			viewNamespace: orgUID,
			canView:       true,
		},
		{
			description:   "can view logs when view ns is credit owner",
			runner:        user2,
			runNamespace:  user2,
			viewer:        user2,
			viewNamespace: user2,
			canView:       true,
		},
		{
			description:   "cannot view logs when view ns is neither resource owner ns nor credit owner",
			runner:        user2,
			runNamespace:  user2,
			viewer:        user1,
			viewNamespace: user1,
			canView:       false,
		},
		{
			description:   "can view logs when view ns is resource owner ns",
			runner:        user2,
			runNamespace:  user2,
			viewer:        user1,
			viewNamespace: orgUID,
			canView:       true,
		},
		{
			description:   "cannot view logs when view ns is neither resource owner ns nor credit owner",
			runner:        user2,
			runNamespace:  orgUID,
			viewer:        user1,
			viewNamespace: user1,
			canView:       false,
		},
		{
			description:   "can view logs when view ns is resource owner ns or credit owner ns",
			runner:        user2,
			runNamespace:  orgUID,
			viewer:        user1,
			viewNamespace: orgUID,
			canView:       true,
		},
		{
			description:   "cannot view logs when view ns is neither resource owner ns nor credit owner",
			runner:        user1,
			runNamespace:  user1,
			viewer:        user3,
			viewNamespace: namespace1,
			canView:       false,
		},
		{
			description:   "cannot view logs when view ns is neither resource owner ns nor credit owner",
			runner:        user1,
			runNamespace:  orgUID,
			viewer:        user3,
			viewNamespace: namespace1,
			canView:       false,
		},
	}

	redisClient, _ := redismock.NewClientMock()

	var temporalClient client.Client

	aclClient := mock.NewACLClientInterfaceMock(mc)

	mockConverter := mock.NewConverterMock(mc)
	mgmtPrivateClient := mock.NewMgmtPrivateServiceClientMock(mc)
	mgmtPrivateClient.CheckNamespaceAdminMock.Return(&mgmtpb.CheckNamespaceAdminResponse{
		Type: mgmtpb.CheckNamespaceAdminResponse_NAMESPACE_ORGANIZATION,
		Uid:  orgUID,
	}, nil)
	mgmtPrivateClient.CheckNamespaceByUIDAdminMock.Return(&mgmtpb.CheckNamespaceByUIDAdminResponse{
		Type:  0,
		Id:    "test-user",
		Owner: nil,
	}, nil)

	mockMinio := mock.NewMinioIMock(mc)
	mockMinio.GetFilesByPathsMock.Return(nil, fmt.Errorf("some error happens"))

	for i, testCase := range testCases {
		c.Run(fmt.Sprintf("get pipeline run with permissions test case %d %s", i+1, testCase.description), func(c *qt.C) {

			tx := db.Begin()
			c.Cleanup(func() { tx.Rollback() })

			repo := repository.NewRepository(tx, redisClient)

			svc := NewService(
				repo,
				redisClient,
				temporalClient,
				aclClient,
				mockConverter,
				mgmtPrivateClient,
				mockMinio,
				nil,
				nil,
				uuid.UUID{},
			)

			ctx := context.Background()

			p := &datamodel.Pipeline{
				Owner: ownerPermalink,
				ID:    pipelineID,
				BaseDynamic: datamodel.BaseDynamic{
					UID:        pipelineUID,
					CreateTime: t0,
					UpdateTime: t0,
				},
				Sharing: &datamodel.Sharing{
					Users:     map[string]*datamodel.SharingUser{},
					ShareCode: nil,
				},
			}
			err := repo.CreateNamespacePipeline(ctx, p)
			c.Assert(err, qt.IsNil)

			got, err := repo.GetNamespacePipelineByID(ctx, ownerPermalink, pipelineID, true, false)
			c.Assert(err, qt.IsNil)
			c.Check(got.NumberOfRuns, qt.Equals, 0)
			c.Check(got.LastRunTime.IsZero(), qt.IsTrue)
			c.Check(got.OwnerUID().String(), qt.Equals, orgUID)

			pipelineRun := &datamodel.PipelineRun{
				PipelineTriggerUID: uuid.Must(uuid.NewV4()),
				PipelineUID:        got.UID,
				Status:             datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
				Source:             datamodel.RunSource(runpb.RunSource_RUN_SOURCE_API),
				TriggeredBy:        testCase.runner,
				Namespace:          testCase.runNamespace,
				StartedTime:        time.Now(),
				TotalDuration:      null.IntFrom(42),
				Components:         nil,
			}

			err = repo.UpsertPipelineRun(ctx, pipelineRun)
			c.Assert(err, qt.IsNil)

			m := make(map[string]string)
			m[constant.HeaderRequesterUIDKey] = testCase.viewNamespace
			m[constant.HeaderUserUIDKey] = testCase.viewer

			ctxWithHeader := metadata.NewIncomingContext(context.Background(), metadata.New(m))
			req := &pb.ListPipelineRunsRequest{
				NamespaceId: "org1",
				PipelineId:  pipelineID,
				Page:        0,
				PageSize:    10,
			}
			runs, err := svc.ListPipelineRuns(ctxWithHeader, req, filtering.Filter{})
			c.Assert(err, qt.IsNil)
			if testCase.canView {
				c.Check(len(runs.PipelineRuns), qt.Equals, 1)
			} else {
				c.Check(len(runs.PipelineRuns), qt.Equals, 0)
			}
		})
	}
}

package presetdownloader

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/protojson"

	openfga "github.com/openfga/api/proto/openfga/v1"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/acl"
	"github.com/instill-ai/pipeline-backend/pkg/component/store"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/x/temporal"
	"github.com/instill-ai/x/zapadapter"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

func DownloadPresetPipelines(ctx context.Context, repo repository.Repository) error {
	// In Instill Cloud, we have a special organization called `preset`, which
	// stores all the preset pipelines that users can use or clone. We also want
	// to provide these preset pipelines in Instill Core. Thus, we have
	// implemented this download logic here.
	// Note: The implementation here is temporary. We need to refactor it to
	// have a better structure and handle use cases such as when the preset is
	// deleted.

	logger, _ := logger.GetZapLogger(ctx)

	db := database.GetSharedConnection()
	defer database.Close(db)

	var temporalClientOptions client.Options
	var err error
	if config.Config.Temporal.Ca != "" && config.Config.Temporal.Cert != "" && config.Config.Temporal.Key != "" {
		if temporalClientOptions, err = temporal.GetTLSClientOption(
			config.Config.Temporal.HostPort,
			config.Config.Temporal.Namespace,
			zapadapter.NewZapAdapter(logger),
			config.Config.Temporal.Ca,
			config.Config.Temporal.Cert,
			config.Config.Temporal.Key,
			config.Config.Temporal.ServerName,
			true,
		); err != nil {
			logger.Fatal(fmt.Sprintf("Unable to get Temporal client options: %s", err))
		}
	} else {
		if temporalClientOptions, err = temporal.GetClientOption(
			config.Config.Temporal.HostPort,
			config.Config.Temporal.Namespace,
			zapadapter.NewZapAdapter(logger)); err != nil {
			logger.Fatal(fmt.Sprintf("Unable to get Temporal client options: %s", err))
		}
	}

	temporalClient, err := client.Dial(temporalClientOptions)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create client: %s", err))
	}
	defer temporalClient.Close()

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	fgaClient, fgaClientConn := acl.InitOpenFGAClient(ctx, config.Config.OpenFGA.Host, config.Config.OpenFGA.Port)
	if fgaClientConn != nil {
		defer fgaClientConn.Close()
	}
	var fgaReplicaClient openfga.OpenFGAServiceClient
	var fgaReplicaClientConn *grpc.ClientConn
	if config.Config.OpenFGA.Replica.Host != "" {

		fgaReplicaClient, fgaReplicaClientConn = acl.InitOpenFGAClient(ctx, config.Config.OpenFGA.Replica.Host, config.Config.OpenFGA.Replica.Port)
		if fgaReplicaClientConn != nil {
			defer fgaReplicaClientConn.Close()
		}
	}

	aclClient := acl.NewACLClient(fgaClient, fgaReplicaClient, redisClient)

	mgmtPrivateServiceClient, mgmtPrivateServiceClientConn := external.InitMgmtPrivateServiceClient(ctx)
	if mgmtPrivateServiceClientConn != nil {
		defer mgmtPrivateServiceClientConn.Close()
	}
	presetOrgResp, err := mgmtPrivateServiceClient.GetOrganizationAdmin(ctx, &mgmtpb.GetOrganizationAdminRequest{OrganizationId: constant.PresetNamespaceID})
	if err != nil {
		return err
	}
	componentStore := store.Init(store.InitParams{
		Logger:  logger,
		Secrets: config.Config.Component.Secrets,
	})

	converter := service.NewConverter(service.ConverterConfig{
		MgmtClient:      mgmtPrivateServiceClient,
		RedisClient:     redisClient,
		ACLClient:       &aclClient,
		Repository:      repo,
		InstillCoreHost: "",
		ComponentStore:  componentStore,
	})

	if config.Config.InstillCloud.Host == "" {
		// Skip the download process if the Instill Cloud host is not set.
		return nil
	}

	clientConn, err := grpc.NewClient(fmt.Sprintf("%s:%d", config.Config.InstillCloud.Host, config.Config.InstillCloud.Port),
		grpc.WithTransportCredentials(credentials.NewTLS((&tls.Config{MinVersion: tls.VersionTLS12}))),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(constant.MaxPayloadSize), grpc.MaxCallSendMsgSize(constant.MaxPayloadSize)))
	if err != nil {
		// Skip the download process if Instill Cloud is unreachable.
		return nil
	}
	defer clientConn.Close()

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}

	ns := resource.Namespace{
		NsType: resource.Organization,
		NsID:   constant.PresetNamespaceID,
		NsUID:  uuid.FromStringOrNil(presetOrgResp.Organization.Uid),
	}

	pageToken := ""
	for {
		url := fmt.Sprintf("https://%s/v1beta/namespaces/%s/pipelines?view=VIEW_RECIPE",
			config.Config.InstillCloud.Host,
			constant.PresetNamespaceID,
		)
		if pageToken != "" {
			url += "&pageToken=" + pageToken
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			// If the preset organization does not exist, we skip the download process.
			return nil
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var listResp pipelinepb.ListNamespacePipelinesResponse
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if err := protojson.Unmarshal(b, &listResp); err != nil {
			return err
		}

		for _, pipeline := range listResp.Pipelines {
			dbPipeline, err := converter.ConvertPipelineToDB(ctx, ns, pipeline)
			if err != nil {
				return err
			}
			dbPipeline.Sharing = &datamodel.Sharing{
				Users: map[string]*datamodel.SharingUser{
					"*/*": &datamodel.SharingUser{
						Role:    "ROLE_EXECUTOR",
						Enabled: true,
					},
				},
			}

			p, err := repo.GetNamespacePipelineByID(ctx, ns.Permalink(), dbPipeline.ID, true, false)
			if err == nil {
				if err := repo.UpdateNamespacePipelineByUID(ctx, p.UID, dbPipeline); err != nil {
					return err
				}
				err = aclClient.SetOwner(ctx, "pipeline", dbPipeline.UID, "organization", ns.NsUID)
				if err != nil {
					return err
				}
				// TODO: use OpenFGA as single source of truth
				err = aclClient.SetPipelinePermissionMap(ctx, dbPipeline)
				if err != nil {
					return err
				}
			} else {
				if err := repo.CreateNamespacePipeline(ctx, dbPipeline); err != nil {
					return err
				}
				err = aclClient.SetOwner(ctx, "pipeline", dbPipeline.UID, "organization", ns.NsUID)
				if err != nil {
					return err
				}
				// TODO: use OpenFGA as single source of truth
				err = aclClient.SetPipelinePermissionMap(ctx, dbPipeline)
				if err != nil {
					return err
				}
			}

			releasePageToken := ""
			for {
				releaseURL := fmt.Sprintf("https://%s/v1beta/organizations/%s/pipelines/%s/releases?view=VIEW_RECIPE",
					config.Config.InstillCloud.Host,
					constant.PresetNamespaceID,
					dbPipeline.ID,
				)
				if releasePageToken != "" {
					releaseURL += "&pageToken=" + releasePageToken
				}

				releaseReq, err := http.NewRequestWithContext(ctx, "GET", releaseURL, nil)
				if err != nil {
					return err
				}

				releaseResp, err := httpClient.Do(releaseReq)
				if err != nil {
					return err
				}
				defer releaseResp.Body.Close()

				if releaseResp.StatusCode != http.StatusOK {
					return fmt.Errorf("unexpected status code: %d", releaseResp.StatusCode)
				}

				var listReleaseResp pipelinepb.ListNamespacePipelineReleasesResponse
				b, err = io.ReadAll(releaseResp.Body)
				if err != nil {
					return err
				}
				if err := protojson.Unmarshal(b, &listReleaseResp); err != nil {
					return err
				}

				for _, release := range listReleaseResp.Releases {
					dbRelease, err := converter.ConvertPipelineReleaseToDB(ctx, dbPipeline.UID, release)
					if err != nil {
						return err
					}
					_, err = repo.GetNamespacePipelineReleaseByID(ctx, ns.Permalink(), dbPipeline.UID, dbRelease.ID, true)
					if err != nil {
						if err := repo.CreateNamespacePipelineRelease(ctx, ns.Permalink(), dbPipeline.UID, dbRelease); err != nil {
							return err
						}
					}
				}

				if listReleaseResp.NextPageToken == "" {
					break
				}
				releasePageToken = listReleaseResp.NextPageToken
			}
		}

		if listResp.NextPageToken == "" {
			break
		}
		pageToken = listResp.NextPageToken
	}

	return nil
}

import grpc from "k6/net/grpc";
import { check, group } from "k6";

import * as constant from "./const.js";
import * as helper from "./helper.js";

const publicClient = new grpc.Client();
const privateClient = new grpc.Client();

publicClient.load(["proto"], "pipeline/v1beta/pipeline_public_service.proto");
privateClient.load(["proto"], "pipeline/v1beta/pipeline_private_service.proto");

// ============================================================================
// Private Service API Tests
// These tests cover admin/private service APIs (PipelinePrivateService)
// that are only accessible internally (port 3081, not exposed externally)
//
// For service-to-service communication, we pass internal headers:
// - Instill-User-Uid: The authenticated user's UUID
// - Instill-Requester-Uid: The requester's UUID (namespace making the request)
// ============================================================================

/**
 * Get internal service metadata with user/requester UIDs.
 * This simulates what the API Gateway would inject after authentication.
 */
function getInternalServiceMetadata() {
  // Get the admin user's UID from the database
  var userUid = helper.getNamespaceUidFromId(constant.defaultUsername);
  if (!userUid) {
    console.log(`[WARN] Could not get user UID for ${constant.defaultUsername}, using fallback`);
    return null;
  }

  return {
    metadata: {
      "Instill-User-Uid": userUid,
      "Instill-Requester-Uid": userUid, // For admin user, requester is the same
    },
  };
}

/**
 * Test LookUpPipelineAdmin (private service)
 * This endpoint is admin-only and requires the internal UID
 * NOTE: Private service port (3081) is not exposed outside container,
 * so this test only runs when executing from within the container.
 */
export function CheckLookUpPipelineAdmin(data) {
  // Skip if running from host - private service port not exposed
  if (constant.isHostMode) {
    group("Pipelines API: Look up a pipeline by UID (Admin) [SKIPPED - host mode]", () => {
      console.log("SKIPPED: Private service tests require running inside container");
    });
    return;
  }

  group("Pipelines API: Look up a pipeline by UID (Admin)", () => {
    publicClient.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });
    privateClient.connect(constant.pipelineGRPCPrivateHost, {
      plaintext: true,
    });

    // Get internal service metadata (simulates what API Gateway injects)
    var internalMeta = getInternalServiceMetadata();
    if (!internalMeta) {
      console.log("[SKIP] Could not get internal service metadata, skipping LookUpPipelineAdmin");
      publicClient.close();
      privateClient.close();
      return;
    }

    var reqBody = Object.assign(
      {},
      constant.simplePipelineWithYAMLRecipe
    );

    // Create a pipeline via public API
    var createRes = publicClient.invoke(
      "pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline",
      {
        parent: constant.namespace,
        pipeline: reqBody,
      },
      data.metadata
    );
    check(createRes, {
      "CreateNamespacePipeline response StatusOK": (r) => r.status === grpc.StatusOK,
    });

    if (createRes.status !== grpc.StatusOK || !createRes.message || !createRes.message.pipeline) {
      console.log("Failed to create pipeline in CheckLookUpPipelineAdmin, skipping remaining tests");
      publicClient.close();
      privateClient.close();
      return;
    }

    var pipelineId = createRes.message.pipeline.id;

    // Get the internal UID from database
    var pipelineUid = helper.getPipelineUidFromId(pipelineId);
    check(pipelineUid, {
      "Got pipeline UID from database": (uid) => uid !== null && uid !== undefined,
    });

    if (!pipelineUid) {
      console.log(`Failed to get pipeline UID for id=${pipelineId}, skipping LookUpAdmin test`);
      // Cleanup
      publicClient.invoke(
        "pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline",
        { name: `${constant.namespace}/pipelines/${pipelineId}` },
        data.metadata
      );
      publicClient.close();
      privateClient.close();
      return;
    }

    // LookUpPipelineAdmin by UID permalink (private service)
    // Use internal service headers for service-to-service communication
    var lookupRes = privateClient.invoke(
      "pipeline.v1beta.PipelinePrivateService/LookUpPipelineAdmin",
      {
        permalink: `pipelines/${pipelineUid}`,
      },
      internalMeta // Use internal service-to-service headers
    );
    check(lookupRes, {
      "LookUpPipelineAdmin response StatusOK": (r) => r.status === grpc.StatusOK,
      "LookUpPipelineAdmin response has pipeline": (r) =>
        r.message && r.message.pipeline,
      "LookUpPipelineAdmin response pipeline has correct id": (r) =>
        r.message && r.message.pipeline && r.message.pipeline.id === pipelineId,
      "LookUpPipelineAdmin response pipeline has name": (r) =>
        r.message && r.message.pipeline && r.message.pipeline.name &&
        r.message.pipeline.name.includes("/pipelines/"),
    });

    // Delete the pipeline
    publicClient.invoke(
      "pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline",
      {
        name: `${constant.namespace}/pipelines/${pipelineId}`,
      },
      data.metadata
    );

    publicClient.close();
    privateClient.close();
  });
}

/**
 * Test ListPipelinesAdmin (private service)
 * This endpoint lists all pipelines across all namespaces (admin only)
 * NOTE: Private service port (3081) is not exposed outside container,
 * so this test only runs when executing from within the container.
 */
export function CheckListPipelinesAdmin(data) {
  // Skip if running from host - private service port not exposed
  if (constant.isHostMode) {
    group("Pipelines API: List all pipelines (Admin) [SKIPPED - host mode]", () => {
      console.log("SKIPPED: Private service tests require running inside container");
    });
    return;
  }

  group("Pipelines API: List all pipelines (Admin)", () => {
    publicClient.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });
    privateClient.connect(constant.pipelineGRPCPrivateHost, {
      plaintext: true,
    });

    // Get internal service metadata (simulates what API Gateway injects)
    var internalMeta = getInternalServiceMetadata();
    if (!internalMeta) {
      console.log("[SKIP] Could not get internal service metadata, skipping ListPipelinesAdmin");
      publicClient.close();
      privateClient.close();
      return;
    }

    var reqBody = Object.assign(
      {},
      constant.simplePipelineWithYAMLRecipe
    );

    // Create a pipeline via public API
    var createRes = publicClient.invoke(
      "pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline",
      {
        parent: constant.namespace,
        pipeline: reqBody,
      },
      data.metadata
    );
    check(createRes, {
      "CreateNamespacePipeline response StatusOK": (r) => r.status === grpc.StatusOK,
    });

    if (createRes.status !== grpc.StatusOK || !createRes.message || !createRes.message.pipeline) {
      console.log("Failed to create pipeline in CheckListPipelinesAdmin, skipping remaining tests");
      publicClient.close();
      privateClient.close();
      return;
    }

    var pipelineId = createRes.message.pipeline.id;

    // ListPipelinesAdmin (private service)
    // Use internal service headers for service-to-service communication
    var listRes = privateClient.invoke(
      "pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin",
      {
        pageSize: 100,
      },
      internalMeta // Use internal service-to-service headers
    );
    check(listRes, {
      "ListPipelinesAdmin response StatusOK": (r) => r.status === grpc.StatusOK,
      "ListPipelinesAdmin response has pipelines": (r) =>
        r.message && r.message.pipelines,
      "ListPipelinesAdmin response contains created pipeline": (r) =>
        r.message && r.message.pipelines &&
        r.message.pipelines.some(p => p.id === pipelineId),
    });

    // Delete the pipeline
    publicClient.invoke(
      "pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline",
      {
        name: `${constant.namespace}/pipelines/${pipelineId}`,
      },
      data.metadata
    );

    publicClient.close();
    privateClient.close();
  });
}

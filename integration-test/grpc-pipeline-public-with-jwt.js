import grpc from "k6/net/grpc";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";

const client = new grpc.Client();
client.load(["proto"], "pipeline/v1beta/pipeline_public_service.proto");

export function CheckCreate(data) {
  group(
    `Pipelines API: Create a pipeline [with random "Instill-User-Uid" header]`,
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      // Note: id is now OUTPUT_ONLY (server-generated), so we don't send it
      var reqBody = Object.assign(
        {
          description: randomString(50),
        },
        constant.simplePipelineWithYAMLRecipe
      );

      // Cannot create a pipeline of a non-exist user
      var createRes = client.invoke(
        "pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        constant.paramsGRPCWithJwt
      );
      console.log(`[DEBUG] CreateNamespacePipeline with random JWT - status: ${createRes.status}, error: ${createRes.error}`);
      check(createRes,
        {
          [`[with random "Instill-User-Uid" header] pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline response StatusUnauthenticated`]:
            (r) => r.status === grpc.StatusUnauthenticated,
        }
      );

      client.close();
    }
  );
}

export function CheckList(data) {
  group(`Pipelines API: List pipelines [with random "Instill-User-Uid" header]`, () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    // Cannot list pipelines of a non-exist user
    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/ListNamespacePipelines",
        {
          parent: `${constant.namespace}`,
        },
        constant.paramsGRPCWithJwt
      ),
      {
        [`[with random "Instill-User-Uid" header] pipeline.v1beta.PipelinePublicService/ListNamespacePipelines response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    client.close();
  });
}

export function CheckGet(data) {
  group(`Pipelines API: Get a pipeline [with random "Instill-User-Uid" header]`, () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    // Note: id is now OUTPUT_ONLY (server-generated), so we don't send it
    var reqBody = Object.assign(
      {
        description: randomString(50),
      },
      constant.simplePipelineWithYAMLRecipe
    );

    var createRes = client.invoke(
      "pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline",
      {
        parent: `${constant.namespace}`,
        pipeline: reqBody,
      },
      data.metadata
    );

    console.log(`[DEBUG] CheckGet CreateNamespacePipeline - host: ${constant.pipelineGRPCPublicHost}, status: ${createRes.status}, error: ${createRes.error}`);

    check(createRes, {
      [`pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
    });

    // Get the server-generated pipeline ID
    if (createRes.status !== grpc.StatusOK || !createRes.message || !createRes.message.pipeline) {
      console.log(`Failed to create pipeline in CheckGet - status: ${createRes.status}, error: ${createRes.error}, skipping remaining tests`);
      client.close();
      return;
    }
    var pipelineId = createRes.message.pipeline.id;

    // Cannot get a pipeline of a non-exist user
    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/GetNamespacePipeline",
        {
          name: `${constant.namespace}/pipelines/${pipelineId}`,
        },
        constant.paramsGRPCWithJwt
      ),
      {
        [`[with random "Instill-User-Uid" header] pipeline.v1beta.PipelinePublicService/GetNamespacePipeline response StatusNotFound`]:
          (r) => r.status === grpc.StatusNotFound,
      }
    );

    // Delete the pipeline
    check(
      client.invoke(
        `pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline`,
        {
          name: `${constant.namespace}/pipelines/${pipelineId}`,
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    client.close();
  });
}

export function CheckUpdate(data) {
  group(
    `Pipelines API: Update a pipeline [with random "Instill-User-Uid" header]`,
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      // Note: id is now OUTPUT_ONLY (server-generated), so we don't send it
      var reqBody = Object.assign(
        {},
        constant.simplePipelineWithYAMLRecipe
      );

      // Create a pipeline
      var resOrigin = client.invoke(
        "pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        data.metadata
      );

      check(resOrigin, {
        [`pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      });

      // Get the server-generated pipeline ID
      if (resOrigin.status !== grpc.StatusOK || !resOrigin.message || !resOrigin.message.pipeline) {
        console.log("Failed to create pipeline in CheckUpdate, skipping remaining tests");
        client.close();
        return;
      }
      var pipelineId = resOrigin.message.pipeline.id;

      // Note: uid no longer exists in the proto, removed from update request
      var reqBodyUpdate = Object.assign({
        name: `${constant.namespace}/pipelines/${pipelineId}`,
        description: randomString(50),
      });

      // Cannot update a pipeline of a non-exist user
      check(
        client.invoke(
          "pipeline.v1beta.PipelinePublicService/UpdateNamespacePipeline",
          {
            pipeline: reqBodyUpdate,
            update_mask: "description",
          },
          constant.paramsGRPCWithJwt
        ),
        {
          [`[with random "Instill-User-Uid" header] pipeline.v1beta.PipelinePublicService/UpdateNamespacePipeline response StatusUnauthenticated`]:
            (r) => r.status === grpc.StatusUnauthenticated,
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
          `pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline`,
          {
            name: `${constant.namespace}/pipelines/${pipelineId}`,
          },
          data.metadata
        ),
        {
          [`pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}

export function CheckRename(data) {
  group(
    `Pipelines API: Rename a pipeline [with random "Instill-User-Uid" header]`,
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      // Note: id is now OUTPUT_ONLY (server-generated), so we don't send it
      var reqBody = Object.assign(
        {},
        constant.simplePipelineWithYAMLRecipe
      );

      // Create a pipeline
      var res = client.invoke(
        "pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        data.metadata
      );

      check(res, {
        [`pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      });

      // Get the server-generated pipeline ID
      if (res.status !== grpc.StatusOK || !res.message || !res.message.pipeline) {
        console.log("Failed to create pipeline in CheckRename, skipping remaining tests");
        client.close();
        return;
      }
      var pipelineId = res.message.pipeline.id;

      check(res, {
        // Note: Backend may return either users/admin or namespaces/admin format during transition
        [`pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline response pipeline name`]:
          (r) => r.message.pipeline.name && r.message.pipeline.name.endsWith(`/pipelines/${pipelineId}`),
      });


      var new_pipeline_id = randomString(10);

      // Cannot rename a pipeline of a non-exist user
      check(
        client.invoke(
          "pipeline.v1beta.PipelinePublicService/RenameNamespacePipeline",
          {
            namespace_id: "admin",
            pipeline_id: pipelineId,
            new_pipeline_id: new_pipeline_id,
          },
          constant.paramsGRPCWithJwt
        ),
        {
          [`[with random "Instill-User-Uid" header] pipeline.v1beta.PipelinePublicService/RenameNamespacePipeline response StatusUnauthenticated`]:
            (r) => r.status === grpc.StatusUnauthenticated,
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
          `pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline`,
          {
            name: `${constant.namespace}/pipelines/${pipelineId}`,
          },
          data.metadata
        ),
        {
          [`pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}

export function CheckLookUp(data) {
  group(
    `Pipelines API: Look up a pipeline by id [with random "Instill-User-Uid" header]`,
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      // Note: id is now OUTPUT_ONLY (server-generated), so we don't send it
      var reqBody = Object.assign(
        {},
        constant.simplePipelineWithYAMLRecipe
      );

      // Create a pipeline
      var res = client.invoke(
        "pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        data.metadata
      );

      check(res, {
        [`pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      });

      // Get the server-generated pipeline ID
      if (res.status !== grpc.StatusOK || !res.message || !res.message.pipeline) {
        console.log("Failed to create pipeline in CheckLookUp, skipping remaining tests");
        client.close();
        return;
      }
      var pipelineId = res.message.pipeline.id;

      // Note: LookUpPipeline now uses id instead of uid (uid no longer exists in proto)
      // Cannot look up a pipeline of a non-exist user
      check(
        client.invoke(
          "pipeline.v1beta.PipelinePublicService/LookUpPipeline",
          {
            permalink: `pipelines/${pipelineId}`,
          },
          constant.paramsGRPCWithJwt
        ),
        {
          [`[with random "Instill-User-Uid" header] pipeline.v1beta.PipelinePublicService/LookUpPipeline response StatusUnauthenticated`]:
            (r) => r.status === grpc.StatusUnauthenticated,
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
          `pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline`,
          {
            name: `${constant.namespace}/pipelines/${pipelineId}`,
          },
          data.metadata
        ),
        {
          [`pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}

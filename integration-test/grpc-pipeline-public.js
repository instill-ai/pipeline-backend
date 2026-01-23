import http from "k6/http";
import grpc from "k6/net/grpc";
import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";
import * as helper from "./helper.js";

const client = new grpc.Client();
client.load(["proto"], "pipeline/v1beta/pipeline_public_service.proto");

export function CheckCreate(data) {
  group("Pipelines API: Create a pipeline", () => {
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

    // Create a pipeline
    var resOrigin = client.invoke(
      "pipeline.v1beta.PipelinePublicService/CreatePipeline",
      {
        parent: `${constant.namespace}`,
        pipeline: reqBody,
      },
      data.metadata
    );

    check(resOrigin, {
      "pipeline.v1beta.PipelinePublicService/CreatePipeline response StatusOK":
        (r) => r.status === grpc.StatusOK,
    });

    // Get the server-generated pipeline ID
    if (resOrigin.status !== grpc.StatusOK || !resOrigin.message || !resOrigin.message.pipeline) {
      console.log("Failed to create pipeline in CheckCreate, skipping remaining tests");
      client.close();
      return;
    }
    var pipelineId = resOrigin.message.pipeline.id;

    check(resOrigin, {
      // Note: Backend may return either users/admin or namespaces/admin format during transition
      "pipeline.v1beta.PipelinePublicService/CreatePipeline response pipeline name":
        (r) => r.message.pipeline.name && r.message.pipeline.name.endsWith(`/pipelines/${pipelineId}`),
      // Note: uid no longer exists in the proto
      "pipeline.v1beta.PipelinePublicService/CreatePipeline response pipeline id exists":
        (r) => r.message.pipeline.id && r.message.pipeline.id.length > 0,
      "pipeline.v1beta.PipelinePublicService/CreatePipeline response pipeline displayName":
        (r) => r.message.pipeline.displayName === reqBody.displayName,
      "pipeline.v1beta.PipelinePublicService/CreatePipeline response pipeline slug derived from displayName":
        (r) => r.message.pipeline.slug === "integration-test-pipeline",
      "pipeline.v1beta.PipelinePublicService/CreatePipeline response pipeline description":
        (r) => r.message.pipeline.description === reqBody.description,
      "pipeline.v1beta.PipelinePublicService/CreatePipeline response pipeline recipe is valid":
        (r) => helper.validateRecipeGRPC(r.message.pipeline.recipe, false),
      "pipeline.v1beta.PipelinePublicService/CreatePipeline response pipeline owner is valid":
        (r) => helper.isValidOwner(r.message.pipeline.owner, data.expectedOwner),
      "pipeline.v1beta.PipelinePublicService/CreatePipeline response pipeline createTime":
        (r) =>
          new Date(r.message.pipeline.createTime).getTime() >
          new Date().setTime(0),
      "pipeline.v1beta.PipelinePublicService/CreatePipeline response pipeline updateTime":
        (r) =>
          new Date(r.message.pipeline.updateTime).getTime() >
          new Date().setTime(0),
    });


    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/CreatePipeline",
        {
          parent: `${constant.namespace}`,
        },
        data.metadata
      ),
      {
        "pipeline.v1beta.PipelinePublicService/CreatePipeline response StatusInvalidArgument":
          (r) => r.status === grpc.StatusInvalidArgument,
      }
    );

    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/CreatePipeline",
        {
          parent: `${constant.namespace}`,
        },
        data.metadata
      ),
      {
        "pipeline.v1beta.PipelinePublicService/CreatePipeline response StatusInvalidArgument":
          (r) => r.status === grpc.StatusInvalidArgument,
      }
    );

    // Note: The duplicate creation test no longer applies since ID is server-generated
    // Each CreatePipeline call generates a unique ID, so StatusAlreadyExists won't happen

    // NOTE: ID validation tests removed - id is now OUTPUT_ONLY (server-generated)
    // Invalid ID tests (null, non-RFC-1034, >32 char, non-ASCII) are no longer applicable.

    // Delete the pipeline
    check(
      client.invoke(
        `pipeline.v1beta.PipelinePublicService/DeletePipeline`,
        {
          name: `${constant.namespace}/pipelines/${resOrigin.message.pipeline.id}`,
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/DeletePipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    client.close();
  });
}

export function CheckList(data) {
  group("Pipelines API: List pipelines", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    // Record initial pipeline count (database might not be clean)
    var initialRes = client.invoke(
      "pipeline.v1beta.PipelinePublicService/ListPipelines",
      {
        parent: `${constant.namespace}`,
      },
      data.metadata
    );

    check(initialRes, {
      [`pipeline.v1beta.PipelinePublicService/ListPipelines initial response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
    });

    var initialCount = initialRes.message ? initialRes.message.totalSize : 0;

    const numPipelines = 200;
    var createdPipelineIds = [];

    // Create pipelines and capture server-generated IDs
    for (var i = 0; i < numPipelines; i++) {
      var reqBody = Object.assign(
        {
          description: randomString(50),
        },
        constant.simplePipelineWithYAMLRecipe
      );

      var createRes = client.invoke(
        "pipeline.v1beta.PipelinePublicService/CreatePipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        data.metadata
      );

      check(createRes, {
        [`pipeline.v1beta.PipelinePublicService/CreatePipeline x${numPipelines} response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      });

      if (createRes.status === grpc.StatusOK && createRes.message && createRes.message.pipeline) {
        createdPipelineIds.push(createRes.message.pipeline.id);
      }
    }


    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/ListPipelines",
        {
          parent: `${constant.namespace}`,
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/ListPipelines response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`pipeline.v1beta.PipelinePublicService/ListPipelines response pipelines.length == 10`]:
          (r) => r.message.pipelines.length === 10,
        [`pipeline.v1beta.PipelinePublicService/ListPipelines response pipelines[0].recipe is null`]:
          (r) => r.message.pipelines[0].recipe === null,
        // totalSize should be initial + 200 created pipelines
        [`pipeline.v1beta.PipelinePublicService/ListPipelines response totalSize >= 200`]:
          (r) => r.message.totalSize >= numPipelines,
      }
    );

    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/ListPipelines",
        {
          parent: `${constant.namespace}`,
          view: "VIEW_FULL",
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/ListPipelines view=VIEW_FULL response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`pipeline.v1beta.PipelinePublicService/ListPipelines view=VIEW_FULL response pipelines[0].recipe is valid`]:
          (r) =>
            helper.validateRecipeGRPC(r.message.pipelines[0].recipe, false),
      }
    );

    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/ListPipelines",
        {
          parent: `${constant.namespace}`,
          view: "VIEW_BASIC",
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/ListPipelines view=VIEW_BASIC response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`pipeline.v1beta.PipelinePublicService/ListPipelines view=VIEW_BASIC response pipelines[0].recipe is null`]:
          (r) => r.message.pipelines[0].recipe === null,
      }
    );

    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/ListPipelines",
        {
          parent: `${constant.namespace}`,
          pageSize: 3,
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/ListPipelines response pipelines.length == 3`]:
          (r) => r.message.pipelines.length === 3,
      }
    );

    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/ListPipelines",
        {
          parent: `${constant.namespace}`,
          pageSize: 101,
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/ListPipelines response pipelines.length == 100`]:
          (r) => r.message.pipelines.length === 100,
      }
    );

    var resFirst100 = client.invoke(
      "pipeline.v1beta.PipelinePublicService/ListPipelines",
      {
        parent: `${constant.namespace}`,
        pageSize: 100,
      },
      data.metadata
    );

    check(resFirst100, {
      [`pipeline.v1beta.PipelinePublicService/ListPipelines pageSize=100 response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
      [`pipeline.v1beta.PipelinePublicService/ListPipelines pageSize=100 response has results`]:
        (r) => r.message.pipelines.length > 0,
    });

    if (resFirst100.message && resFirst100.message.nextPageToken) {
      var resSecond100 = client.invoke(
        "pipeline.v1beta.PipelinePublicService/ListPipelines",
        {
          parent: `${constant.namespace}`,
          pageSize: 100,
          pageToken: resFirst100.message.nextPageToken,
        },
        data.metadata
      );
      check(resSecond100, {
        [`pipeline.v1beta.PipelinePublicService/ListPipelines pageSize=100 page 2 response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`pipeline.v1beta.PipelinePublicService/ListPipelines pageSize=100 page 2 response has results`]:
          (r) => r.message.pipelines.length > 0,
      });
    }

    // Filtering

    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/ListPipelines",
        {
          parent: `${constant.namespace}`,
          filter:
            'createTime>timestamp("2000-06-19T23:31:08.657Z")',
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/ListPipelines filter: state=createTime>timestamp("2000-06-19T23:31:08.657Z") response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`pipeline.v1beta.PipelinePublicService/ListPipelines filter: state=createTime>timestamp("2000-06-19T23:31:08.657Z") response pipelines.length`]:
          (r) => r.message.pipelines.length > 0,
      }
    );

    // Delete the pipelines
    for (const pipelineId of createdPipelineIds) {
      check(
        client.invoke(
          `pipeline.v1beta.PipelinePublicService/DeletePipeline`,
          {
            name: `${constant.namespace}/pipelines/${pipelineId}`,
          },
          data.metadata
        ),
        {
          [`pipeline.v1beta.PipelinePublicService/DeletePipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );
    }

    client.close();
  });
}

export function CheckGet(data) {
  group("Pipelines API: Get a pipeline", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    var reqBody = Object.assign(
      {
        description: randomString(50),
      },
      constant.simplePipelineWithYAMLRecipe
    );

    var createRes = client.invoke(
      "pipeline.v1beta.PipelinePublicService/CreatePipeline",
      {
        parent: `${constant.namespace}`,
        pipeline: reqBody,
      },
      data.metadata
    );

    check(createRes, {
      [`pipeline.v1beta.PipelinePublicService/CreatePipeline response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
    });

    if (createRes.status !== grpc.StatusOK || !createRes.message || !createRes.message.pipeline) {
      console.log("Failed to create pipeline in CheckGet, skipping remaining tests");
      client.close();
      return;
    }
    var pipelineId = createRes.message.pipeline.id;

    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/GetPipeline",
        {
          name: `${constant.namespace}/pipelines/${pipelineId}`,
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/GetPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        // Note: Backend may return either users/admin or namespaces/admin format during transition
        [`pipeline.v1beta.PipelinePublicService/GetPipeline response pipeline name`]:
          (r) => r.message.pipeline.name && r.message.pipeline.name.endsWith(`/pipelines/${pipelineId}`),
        // Note: uid is no longer exposed in the API
        [`pipeline.v1beta.PipelinePublicService/GetPipeline response pipeline id`]:
          (r) => r.message.pipeline.id === pipelineId,
        [`pipeline.v1beta.PipelinePublicService/GetPipeline response pipeline description`]:
          (r) => r.message.pipeline.description === reqBody.description,
        [`pipeline.v1beta.PipelinePublicService/GetPipeline response pipeline recipe is null`]:
          (r) => r.message.pipeline.recipe === null,
      }
    );


    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/GetPipeline",
        {
          name: `${constant.namespace}/pipelines/${pipelineId}`,
          view: "VIEW_FULL",
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/GetPipeline view: "VIEW_FULL" response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`pipeline.v1beta.PipelinePublicService/GetPipeline view: "VIEW_FULL" response pipeline recipe is not null`]:
          (r) => r.message.pipeline.recipe !== null,
        [`pipeline.v1beta.PipelinePublicService/GetPipeline view: "VIEW_FULL" response pipeline owner is valid`]:
          (r) => helper.isValidOwner(r.message.pipeline.owner, data.expectedOwner),
      }
    );

    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/GetPipeline",
        {
          name: `${constant.namespace}/pipelines/this-id-does-not-exist`,
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/GetPipeline this-id-does-not-exist response StatusNotFound`]:
          (r) => r.status === grpc.StatusNotFound,
      }
    );

    // Delete the pipeline
    check(
      client.invoke(
        `pipeline.v1beta.PipelinePublicService/DeletePipeline`,
        {
          name: `${constant.namespace}/pipelines/${pipelineId}`,
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/DeletePipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    client.close();
  });
}

export function CheckUpdate(data) {
  group("Pipelines API: Update a pipeline", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    var reqBody = Object.assign(
      {},
      constant.simplePipelineWithYAMLRecipe
    );

    // Create a pipeline
    var resOrigin = client.invoke(
      "pipeline.v1beta.PipelinePublicService/CreatePipeline",
      {
        parent: `${constant.namespace}`,
        pipeline: reqBody,
      },
      data.metadata
    );

    check(resOrigin, {
      [`pipeline.v1beta.PipelinePublicService/CreatePipeline response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
    });

    if (resOrigin.status !== grpc.StatusOK || !resOrigin.message || !resOrigin.message.pipeline) {
      console.log("Failed to create pipeline in CheckUpdate, skipping remaining tests");
      client.close();
      return;
    }
    var pipelineId = resOrigin.message.pipeline.id;

    var reqBodyUpdate = Object.assign({
      name: `${constant.namespace}/pipelines/${pipelineId}`,
      description: randomString(50),
    });

    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/UpdatePipeline",
        {
          pipeline: reqBodyUpdate,
          update_mask: "description",
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/UpdatePipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`pipeline.v1beta.PipelinePublicService/UpdatePipeline response pipeline name (OUTPUT_ONLY)`]:
          (r) =>
            r.message.pipeline.name ===
            `${constant.namespace}/pipelines/${pipelineId}`,
        // Note: uid is no longer exposed in the API
        [`pipeline.v1beta.PipelinePublicService/UpdatePipeline response pipeline id (OUTPUT_ONLY)`]:
          (r) => r.message.pipeline.id === pipelineId,
        [`pipeline.v1beta.PipelinePublicService/UpdatePipeline response pipeline description (OPTIONAL)`]:
          (r) => r.message.pipeline.description === reqBodyUpdate.description,
        [`pipeline.v1beta.PipelinePublicService/UpdatePipeline response pipeline recipe (IMMUTABLE)`]:
          (r) => r.message.pipeline.recipe !== null,
        [`pipeline.v1beta.PipelinePublicService/UpdatePipeline response pipeline createTime (OUTPUT_ONLY)`]:
          (r) =>
            new Date(r.message.pipeline.createTime).getTime() >
            new Date().setTime(0),
        [`pipeline.v1beta.PipelinePublicService/UpdatePipeline response pipeline updateTime (OUTPUT_ONLY)`]:
          (r) =>
            new Date(r.message.pipeline.updateTime).getTime() >
            new Date().setTime(0),
        [`pipeline.v1beta.PipelinePublicService/UpdatePipeline response pipeline updateTime > createTime`]:
          (r) =>
            new Date(r.message.pipeline.updateTime).getTime() >
            new Date(r.message.pipeline.createTime).getTime(),
      }
    );

    reqBodyUpdate.description = "";
    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/UpdatePipeline",
        {
          pipeline: reqBodyUpdate,
          update_mask: "description",
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/UpdatePipeline response pipeline description empty`]:
          (r) => r.message.pipeline.description === "",
      }
    );

    reqBodyUpdate.description = randomString(10);
    check(
      client.invoke(
        "pipeline.v1beta.PipelinePublicService/UpdatePipeline",
        {
          pipeline: reqBodyUpdate,
          update_mask: "description",
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/UpdatePipeline response pipeline description non-empty`]:
          (r) => r.message.pipeline.description === reqBodyUpdate.description,
      }
    );

    // Note: id is now OUTPUT_ONLY, so these IMMUTABLE field tests are no longer applicable
    // The server ignores the id field in update requests

    // Delete the pipeline
    check(
      client.invoke(
        `pipeline.v1beta.PipelinePublicService/DeletePipeline`,
        {
          name: `${constant.namespace}/pipelines/${pipelineId}`,
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/DeletePipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    client.close();
  });
}


export function CheckRename(data) {
  group("Pipelines API: Rename a pipeline", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    var reqBody = Object.assign(
      {},
      constant.simplePipelineWithYAMLRecipe
    );

    // Create a pipeline
    var res = client.invoke(
      "pipeline.v1beta.PipelinePublicService/CreatePipeline",
      {
        parent: `${constant.namespace}`,
        pipeline: reqBody,
      },
      data.metadata
    );

    check(res, {
      [`pipeline.v1beta.PipelinePublicService/CreatePipeline response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
    });

    if (res.status !== grpc.StatusOK || !res.message || !res.message.pipeline) {
      console.log("Failed to create pipeline in CheckRename, skipping remaining tests");
      client.close();
      return;
    }
    var pipelineId = res.message.pipeline.id;

    var new_pipeline_id = randomString(10);

    var renameRes = client.invoke(
      "pipeline.v1beta.PipelinePublicService/RenamePipeline",
      {
        name: `${constant.namespace}/pipelines/${pipelineId}`,
        new_pipeline_id: new_pipeline_id,
      },
      data.metadata
    );
    check(
      renameRes,
      {
        [`pipeline.v1beta.PipelinePublicService/RenamePipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        // Note: Backend may return either users/admin or namespaces/admin format during transition
        [`pipeline.v1beta.PipelinePublicService/RenamePipeline response pipeline new name`]:
          (r) =>
            r.message.pipeline.name && r.message.pipeline.name.endsWith(`/pipelines/${new_pipeline_id}`),
        [`pipeline.v1beta.PipelinePublicService/RenamePipeline response pipeline new id`]:
          (r) => r.message.pipeline.id === new_pipeline_id,
      }
    );

    // Delete the pipeline
    check(
      client.invoke(
        `pipeline.v1beta.PipelinePublicService/DeletePipeline`,
        {
          name: `${constant.namespace}/pipelines/${new_pipeline_id}`,
        },
        data.metadata
      ),
      {
        [`pipeline.v1beta.PipelinePublicService/DeletePipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    client.close();
  });
}

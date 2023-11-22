import http from "k6/http";
import grpc from "k6/net/grpc";
import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";
import * as helper from "./helper.js";

const client = new grpc.Client();
client.load(["../proto/vdp/pipeline/v1alpha"], "pipeline_public_service.proto");

export function CheckCreate(metadata) {
  group("Pipelines API: Create a pipeline", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    var reqBody = Object.assign(
      {
        id: randomString(63),
        description: randomString(50),
      },
      constant.simpleRecipe
    );

    // Create a pipeline
    var resOrigin = client.invoke(
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
      {
        parent: `${constant.namespace}`,
        pipeline: reqBody,
      },
      metadata
    );

    check(resOrigin, {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusOK":
        (r) => r.status === grpc.StatusOK,
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response pipeline name":
        (r) => r.message.pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response pipeline uid":
        (r) => helper.isUUID(r.message.pipeline.uid),
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response pipeline id":
        (r) => r.message.pipeline.id === reqBody.id,
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response pipeline description":
        (r) => r.message.pipeline.description === reqBody.description,
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response pipeline recipe is valid":
        (r) => helper.validateRecipeGRPC(r.message.pipeline.recipe, false),
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response pipeline owner is UUID":
        (r) => helper.isValidOwner(r.message.pipeline.user),
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response pipeline create_time":
        (r) =>
          new Date(r.message.pipeline.createTime).getTime() >
          new Date().setTime(0),
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response pipeline update_time":
        (r) =>
          new Date(r.message.pipeline.updateTime).getTime() >
          new Date().setTime(0),
    });


    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
        {
          parent: `${constant.namespace}`,
        },
        metadata
      ),
      {
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusInvalidArgument":
          (r) => r.status === grpc.StatusInvalidArgument,
      }
    );

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
        {
          parent: `${constant.namespace}`,
        },
        metadata
      ),
      {
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusInvalidArgument":
          (r) => r.status === grpc.StatusInvalidArgument,
      }
    );

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        metadata
      ),
      {
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusAlreadyExists":
          (r) => r.status === grpc.StatusAlreadyExists,
      }
    );

    check(
      client.invoke(
        `vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline`,
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline ${reqBody.id} response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        metadata
      ),
      {
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusOK":
          (r) => r.status === grpc.StatusOK,
      }
    );

    reqBody.id = null;
    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        metadata
      ),
      {
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline with null id response StatusInvalidArgument":
          (r) => r.status === grpc.StatusInvalidArgument,
      }
    );

    reqBody.id = "abcd?*&efg!";
    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        metadata
      ),
      {
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline with non-RFC-1034 naming id response StatusInvalidArgument":
          (r) => r.status === grpc.StatusInvalidArgument,
      }
    );

    reqBody.id = randomString(64);
    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        metadata
      ),
      {
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline with > 63-character id response StatusInvalidArgument":
          (r) => r.status === grpc.StatusInvalidArgument,
      }
    );

    reqBody.id = "🧡💜我愛潤物科技💚💙";
    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        metadata
      ),
      {
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline with non-ASCII id response StatusInvalidArgument":
          (r) => r.status === grpc.StatusInvalidArgument,
      }
    );

    // Delete the pipeline
    check(
      client.invoke(
        `vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline`,
        {
          name: `${constant.namespace}/pipelines/${resOrigin.message.pipeline.id}`,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    client.close();
  });
}

export function CheckList(metadata) {
  group("Pipelines API: List pipelines", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines",
        {
          parent: `${constant.namespace}`,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines response nextPageToken is empty`]:
          (r) => r.message.nextPageToken === "",
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines response totalSize is 0`]:
          (r) => r.message.totalSize == 0,
      }
    );

    const numPipelines = 200;
    var reqBodies = [];
    for (var i = 0; i < numPipelines; i++) {
      reqBodies[i] = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.simpleRecipeWithoutCSV
      );
    }

    // Create pipelines
    for (const reqBody of reqBodies) {
      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
          {
            parent: `${constant.namespace}`,
            pipeline: reqBody,
          },
          metadata
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline x${reqBodies.length} response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

    }


    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines",
        {
          parent: `${constant.namespace}`,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines response pipelines.length == 10`]:
          (r) => r.message.pipelines.length === 10,
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines response pipelines[0].recipe is null`]:
          (r) => r.message.pipelines[0].recipe === null,
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines response totalSize == 200`]:
          (r) => r.message.totalSize == 200,
      }
    );

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines",
        {
          parent: `${constant.namespace}`,
          view: "VIEW_FULL",
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines view=VIEW_FULL response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines view=VIEW_FULL response pipelines[0].recipe is valid`]:
          (r) =>
            helper.validateRecipeGRPC(r.message.pipelines[0].recipe, false),
      }
    );

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines",
        {
          parent: `${constant.namespace}`,
          view: "VIEW_BASIC",
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines view=VIEW_BASIC response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines view=VIEW_BASIC response pipelines[0].recipe is null`]:
          (r) => r.message.pipelines[0].recipe === null,
      }
    );

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines",
        {
          parent: `${constant.namespace}`,
          pageSize: 3,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines response pipelines.length == 3`]:
          (r) => r.message.pipelines.length === 3,
      }
    );

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines",
        {
          parent: `${constant.namespace}`,
          pageSize: 101,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines response pipelines.length == 100`]:
          (r) => r.message.pipelines.length === 100,
      }
    );

    var resFirst100 = client.invoke(
      "vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines",
      {
        parent: `${constant.namespace}`,
        pageSize: 100,
      },
      metadata
    );
    var resSecond100 = client.invoke(
      "vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines",
      {
        parent: `${constant.namespace}`,
        pageSize: 100,
        pageToken: resFirst100.message.nextPageToken,
      },
      metadata
    );
    check(resSecond100, {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines pageSize=100 pageToken=${resFirst100.message.nextPageToken} response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines pageSize=100 pageToken=${resFirst100.message.nextPageToken} response 100 results`]:
        (r) => r.message.pipelines.length === 100,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines pageSize=100 pageToken=${resFirst100.message.nextPageToken} nextPageToken is empty`]:
        (r) => r.message.nextPageToken === "",
    });

    // Filtering

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines",
        {
          parent: `${constant.namespace}`,
          filter:
            'create_time>timestamp("2000-06-19T23:31:08.657Z")',
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines filter: state=create_time>timestamp("2000-06-19T23:31:08.657Z") response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines filter: state=create_time>timestamp("2000-06-19T23:31:08.657Z") response pipelines.length`]:
          (r) => r.message.pipelines.length > 0,
      }
    );

    // Get UUID for foreign resources
    var srcConnPermalink = "operator-definitions/2ac8be70-0f7a-4b61-a33d-098b8acfa6f3"

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines",
        {
          parent: `${constant.namespace}`,
          filter: `recipe.components.definition_name:"${srcConnPermalink}"`,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines filter: recipe.components.definition_name:"${srcConnPermalink}" response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines filter: recipe.components.definition_name:"${srcConnPermalink}" response pipelines.length`]:
          (r) => r.message.pipelines.length > 0,
      }
    );

    // Delete the pipelines
    for (const reqBody of reqBodies) {
      check(
        client.invoke(
          `vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${reqBody.id}`,
          },
          metadata
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );
    }

    client.close();
  });
}

export function CheckGet(metadata) {
  group("Pipelines API: Get a pipeline", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    var reqBody = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.simpleRecipe
    );

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline",
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline name: pipelines/${reqBody.id} response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline name: pipelines/${reqBody.id} response pipeline name`]:
          (r) => r.message.pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
        [`vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline name: pipelines/${reqBody.id} response pipeline uid`]:
          (r) => helper.isUUID(r.message.pipeline.uid),
        [`vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline name: pipelines/${reqBody.id} response pipeline id`]:
          (r) => r.message.pipeline.id === reqBody.id,
        [`vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline name: pipelines/${reqBody.id} response pipeline description`]:
          (r) => r.message.pipeline.description === reqBody.description,
        [`vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline name: pipelines/${reqBody.id} response pipeline recipe is null`]:
          (r) => r.message.pipeline.recipe === null,
        [`vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline name: pipelines/${reqBody.id} response pipeline owner is UUID`]:
          (r) => helper.isValidOwner(r.message.pipeline.user),
      }
    );

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline",
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
          view: "VIEW_FULL",
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline name: pipelines/${reqBody.id} view: "VIEW_FULL" response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline name: pipelines/${reqBody.id} view: "VIEW_FULL" response pipeline recipe is null`]:
          (r) => r.message.pipeline.recipe !== null,
        [`vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline name: pipelines/${reqBody.id} view: "VIEW_FULL" response pipeline owner is UUID`]:
          (r) => helper.isValidOwner(r.message.pipeline.user),
      }
    );

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline",
        {
          name: `${constant.namespace}/pipelines/this-id-does-not-exist`,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline name: this-id-does-not-exist response StatusNotFound`]:
          (r) => r.status === grpc.StatusNotFound,
      }
    );

    // Delete the pipeline
    check(
      client.invoke(
        `vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline`,
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    client.close();
  });
}

export function CheckUpdate(metadata) {
  group("Pipelines API: Update a pipeline", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.simpleRecipe
    );

    // Create a pipeline
    var resOrigin = client.invoke(
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
      {
        parent: `${constant.namespace}`,
        pipeline: reqBody,
      },
      metadata
    );

    check(resOrigin, {
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
    });

    var reqBodyUpdate = Object.assign({
      id: reqBody.id,
      name: `${constant.namespace}/pipelines/${reqBody.id}`,
      uid: "output-only-to-be-ignored",
      description: randomString(50),
    });

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline",
        {
          pipeline: reqBodyUpdate,
          update_mask: "description",
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline response pipeline name (OUTPUT_ONLY)`]:
          (r) =>
            r.message.pipeline.name ===
            `${constant.namespace}/pipelines/${resOrigin.message.pipeline.id}`,
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline response pipeline uid (OUTPUT_ONLY)`]:
          (r) => r.message.pipeline.uid === resOrigin.message.pipeline.uid,
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline response pipeline id (IMMUTABLE)`]:
          (r) => r.message.pipeline.id === resOrigin.message.pipeline.id,
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline response pipeline state (OUTPUT_ONLY)`]:
          (r) => r.message.pipeline.state === resOrigin.message.pipeline.state,
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline response pipeline description (OPTIONAL)`]:
          (r) => r.message.pipeline.description === reqBodyUpdate.description,
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline response pipeline recipe (IMMUTABLE)`]:
          (r) => r.message.pipeline.recipe !== null,
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline response pipeline createTime (OUTPUT_ONLY)`]:
          (r) =>
            new Date(r.message.pipeline.createTime).getTime() >
            new Date().setTime(0),
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline response pipeline updateTime (OUTPUT_ONLY)`]:
          (r) =>
            new Date(r.message.pipeline.updateTime).getTime() >
            new Date().setTime(0),
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline response pipeline updateTime > create_time`]:
          (r) =>
            new Date(r.message.pipeline.updateTime).getTime() >
            new Date(r.message.pipeline.createTime).getTime(),
      }
    );

    reqBodyUpdate.description = "";
    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline",
        {
          pipeline: reqBodyUpdate,
          update_mask: "description",
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline response pipeline description empty`]:
          (r) => r.message.pipeline.description === "",
      }
    );

    reqBodyUpdate.description = randomString(10);
    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline",
        {
          pipeline: reqBodyUpdate,
          update_mask: "description",
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline response pipeline description non-empty`]:
          (r) => r.message.pipeline.description === reqBodyUpdate.description,
      }
    );

    reqBodyUpdate.id = randomString(10);
    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline",
        {
          pipeline: reqBodyUpdate,
          update_mask: "id",
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline updating IMMUTABLE field with different id response StatusInvalidArgument`]:
          (r) => r.status === grpc.StatusInvalidArgument,
      }
    );

    reqBodyUpdate.id = reqBody.id;
    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline",
        {
          pipeline: reqBodyUpdate,
          update_mask: "id",
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline updating IMMUTABLE field with the same id response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    // Delete the pipeline
    check(
      client.invoke(
        `vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline`,
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    client.close();
  });
}


export function CheckRename(metadata) {
  group("Pipelines API: Rename a pipeline", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.simpleRecipe
    );

    // Create a pipeline
    var res = client.invoke(
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
      {
        parent: `${constant.namespace}`,
        pipeline: reqBody,
      },
      metadata
    );

    check(res, {
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response pipeline name`]:
        (r) => r.message.pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
    });

    reqBody.new_pipeline_id = randomString(10);

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/RenameUserPipeline",
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
          new_pipeline_id: reqBody.new_pipeline_id,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/RenameUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePublicService/RenameUserPipeline response pipeline new name`]:
          (r) =>
            r.message.pipeline.name === `${constant.namespace}/pipelines/${reqBody.new_pipeline_id}`,
        [`vdp.pipeline.v1alpha.PipelinePublicService/RenameUserPipeline response pipeline new id`]:
          (r) => r.message.pipeline.id === reqBody.new_pipeline_id,
      }
    );

    // Delete the pipeline
    check(
      client.invoke(
        `vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline`,
        {
          name: `${constant.namespace}/pipelines/${reqBody.new_pipeline_id}`,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    client.close();
  });
}

export function CheckLookUp(metadata) {
  group("Pipelines API: Look up a pipeline by uid", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.simpleRecipe
    );

    // Create a pipeline
    var res = client.invoke(
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
      {
        parent: `${constant.namespace}`,
        pipeline: reqBody,
      },
      metadata
    );

    check(res, {
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
    });

    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/LookUpPipeline",
        {
          permalink: `pipelines/${res.message.pipeline.uid}`,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/LookUpPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePublicService/LookUpPipeline response pipeline new name`]:
          (r) => r.message.pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
      }
    );

    // Delete the pipeline
    check(
      client.invoke(
        `vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline`,
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
        },
        metadata
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    client.close();
  });
}

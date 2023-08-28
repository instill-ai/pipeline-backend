import http from "k6/http";
import grpc from "k6/net/grpc";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";
import * as helper from "./helper.js";

const clientPrivate = new grpc.Client();
const clientPublic = new grpc.Client();
clientPrivate.load(
  ["proto/vdp/pipeline/v1alpha"],
  "pipeline_private_service.proto"
);
clientPublic.load(
  ["proto/vdp/pipeline/v1alpha"],
  "pipeline_public_service.proto"
);

export function CheckList() {
  group("Pipelines API: List pipelines by admin", () => {
    clientPrivate.connect(constant.pipelineGRPCPrivateHost, {
      plaintext: true,
    });

    clientPublic.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    check(
      clientPrivate.invoke(
        "vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin",
        {},
        {}
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response nextPageToken is empty`]:
          (r) => r.message.nextPageToken === "",
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response totalSize is 0`]:
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
        clientPublic.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
          {
            parent: `${constant.namespace}`,
            pipeline: reqBody,
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline x${reqBodies.length} response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );
    }

    check(
      clientPrivate.invoke(
        "vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin",
        {},
        {}
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response pipelines.length == 10`]:
          (r) => r.message.pipelines.length === 10,
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response pipelines[0].recipe is null`]:
          (r) => r.message.pipelines[0].recipe === null,
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response totalSize == 200`]:
          (r) => r.message.totalSize == 200,
      }
    );

    check(
      clientPrivate.invoke(
        "vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin",
        {
          view: "VIEW_FULL",
        },
        {}
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin view=VIEW_FULL response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin view=VIEW_FULL response pipelines[0].recipe is valid`]:
          (r) => helper.validateRecipeGRPC(r.message.pipelines[0].recipe, true),
      }
    );

    check(
      clientPrivate.invoke(
        "vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin",
        {
          view: "VIEW_BASIC",
        },
        {}
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin view=VIEW_BASIC response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin view=VIEW_BASIC response pipelines[0].recipe is null`]:
          (r) => r.message.pipelines[0].recipe === null,
      }
    );

    check(
      clientPrivate.invoke(
        "vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin",
        {
          pageSize: 3,
        },
        {}
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response pipelines.length == 3`]:
          (r) => r.message.pipelines.length === 3,
      }
    );

    check(
      clientPrivate.invoke(
        "vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin",
        {
          pageSize: 101,
        },
        {}
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response pipelines.length == 100`]:
          (r) => r.message.pipelines.length === 100,
      }
    );

    var resFirst100 = clientPrivate.invoke(
      "vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin",
      {
        pageSize: 100,
      },
      {}
    );
    var resSecond100 = clientPrivate.invoke(
      "vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin",
      {
        pageSize: 100,
        pageToken: resFirst100.message.nextPageToken,
      },
      {}
    );
    check(resSecond100, {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin pageSize=100 pageToken=${resFirst100.message.nextPageToken} response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin pageSize=100 pageToken=${resFirst100.message.nextPageToken} response 100 results`]:
        (r) => r.message.pipelines.length === 100,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin pageSize=100 pageToken=${resFirst100.message.nextPageToken} nextPageToken is empty`]:
        (r) => r.message.nextPageToken === "",
    });

    // Filtering

    check(
      clientPrivate.invoke(
        "vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin",
        {
          filter:
            'create_time>timestamp("2000-06-19T23:31:08.657Z")',
        },
        {}
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: create_time>timestamp("2000-06-19T23:31:08.657Z") response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: create_time>timestamp("2000-06-19T23:31:08.657Z") response pipelines.length`]:
          (r) => r.message.pipelines.length > 0,
      }
    );

    var srcConnPermalink = "operator-definitions/2ac8be70-0f7a-4b61-a33d-098b8acfa6f3"

    check(
      clientPrivate.invoke(
        "vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin",
        {
          filter: `recipe.components.definition_name:"${srcConnPermalink}"`,
        },
        {}
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: recipe.components.definition_name:"${srcConnPermalink}" response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: recipe.components.definition_name:"${srcConnPermalink}" response pipelines.length`]:
          (r) => r.message.pipelines.length > 0,
      }
    );

    // Delete the pipelines
    for (const reqBody of reqBodies) {
      check(
        clientPublic.invoke(
          `vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${reqBody.id}`,
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );
    }

    clientPrivate.close();
    clientPublic.close();
  });
}

export function CheckLookUp() {
  group("Pipelines API: Look up a pipeline by uid by admin", () => {
    clientPrivate.connect(constant.pipelineGRPCPrivateHost, {
      plaintext: true,
    });

    clientPublic.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.simpleRecipe
    );

    // Create a pipeline
    var res = clientPublic.invoke(
      "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
      {
        parent: `${constant.namespace}`,
        pipeline: reqBody,
      }
    );

    check(res, {
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
    });

    check(
      clientPrivate.invoke(
        "vdp.pipeline.v1alpha.PipelinePrivateService/LookUpPipelineAdmin",
        {
          permalink: `pipelines/${res.message.pipeline.uid}`,
        }
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePrivateService/LookUpPipelineAdmin response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePrivateService/LookUpPipelineAdmin response pipeline new name`]:
          (r) => r.message.pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
      }
    );

    // Delete the pipeline
    check(
      clientPublic.invoke(
        `vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline`,
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
        }
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    clientPrivate.close();
    clientPublic.close();
  });
}

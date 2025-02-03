import http from "k6/http";
import grpc from "k6/net/grpc";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";
import * as helper from "./helper.js";

const clientPrivate = new grpc.Client();
const clientPublic = new grpc.Client();
clientPrivate.load(
  ["../proto/pipeline/pipeline/v1beta"],
  "pipeline_private_service.proto"
);
clientPublic.load(
  ["../proto/pipeline/pipeline/v1beta"],
  "pipeline_public_service.proto"
);

export function CheckList(data) {
  group("Pipelines API: List pipelines by admin", () => {
    clientPrivate.connect(constant.pipelineGRPCPrivateHost, {
      plaintext: true,
    });

    clientPublic.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    check(
      clientPrivate.invoke(
        "pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin",
        {},
        {}
      ),
      {
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin response nextPageToken is empty`]:
          (r) => r.message.nextPageToken === "",
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin response totalSize is 0`]:
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
        constant.simplePipelineWithYAMLRecipe
      );
    }

    // Create pipelines
    for (const reqBody of reqBodies) {
      check(
        clientPublic.invoke(
          "pipeline.pipeline.v1beta.PipelinePublicService/CreateUserPipeline",
          {
            parent: `${constant.namespace}`,
            pipeline: reqBody,
          },
          data.metadata
        ),
        {
          [`pipeline.pipeline.v1beta.PipelinePublicService/CreateUserPipeline x${reqBodies.length} response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );
    }

    check(
      clientPrivate.invoke(
        "pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin",
        {},
        {}
      ),
      {
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin response pipelines.length == 10`]:
          (r) => r.message.pipelines.length === 10,
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin response pipelines[0].recipe is null`]:
          (r) => r.message.pipelines[0].recipe === null,
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin response totalSize == 200`]:
          (r) => r.message.totalSize == 200,
      }
    );

    check(
      clientPrivate.invoke(
        "pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin",
        {
          view: "VIEW_FULL",
        },
        {}
      ),
      {
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin view=VIEW_FULL response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin view=VIEW_FULL response pipelines[0].recipe is valid`]:
          (r) => helper.validateRecipeGRPC(r.message.pipelines[0].recipe, true),
      }
    );

    check(
      clientPrivate.invoke(
        "pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin",
        {
          view: "VIEW_BASIC",
        },
        {}
      ),
      {
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin view=VIEW_BASIC response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin view=VIEW_BASIC response pipelines[0].recipe is null`]:
          (r) => r.message.pipelines[0].recipe === null,
      }
    );

    check(
      clientPrivate.invoke(
        "pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin",
        {
          pageSize: 3,
        },
        {}
      ),
      {
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin response pipelines.length == 3`]:
          (r) => r.message.pipelines.length === 3,
      }
    );

    check(
      clientPrivate.invoke(
        "pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin",
        {
          pageSize: 101,
        },
        {}
      ),
      {
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin response pipelines.length == 100`]:
          (r) => r.message.pipelines.length === 100,
      }
    );

    var resFirst100 = clientPrivate.invoke(
      "pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin",
      {
        pageSize: 100,
      },
      {}
    );
    var resSecond100 = clientPrivate.invoke(
      "pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin",
      {
        pageSize: 100,
        pageToken: resFirst100.message.nextPageToken,
      },
      {}
    );
    check(resSecond100, {
      [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin pageSize=100 pageToken=${resFirst100.message.nextPageToken} response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
      [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin pageSize=100 pageToken=${resFirst100.message.nextPageToken} response 100 results`]:
        (r) => r.message.pipelines.length === 100,
      [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin pageSize=100 pageToken=${resFirst100.message.nextPageToken} nextPageToken is empty`]:
        (r) => r.message.nextPageToken === "",
    });

    // Filtering

    check(
      clientPrivate.invoke(
        "pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin",
        {
          filter:
            'createTime>timestamp("2000-06-19T23:31:08.657Z")',
        },
        {}
      ),
      {
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin filter: createTime>timestamp("2000-06-19T23:31:08.657Z") response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`pipeline.pipeline.v1beta.PipelinePrivateService/ListPipelinesAdmin filter: createTime>timestamp("2000-06-19T23:31:08.657Z") response pipelines.length`]:
          (r) => r.message.pipelines.length > 0,
      }
    );


    // Delete the pipelines
    for (const reqBody of reqBodies) {
      check(
        clientPublic.invoke(
          `pipeline.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${reqBody.id}`,
          },
          data.metadata
        ),
        {
          [`pipeline.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );
    }

    clientPrivate.close();
    clientPublic.close();
  });
}

export function CheckLookUp(data) {
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
      constant.simplePipelineWithYAMLRecipe
    );

    // Create a pipeline
    var res = clientPublic.invoke(
      "pipeline.pipeline.v1beta.PipelinePublicService/CreateUserPipeline",
      {
        parent: `${constant.namespace}`,
        pipeline: reqBody,
      },
      data.metadata
    );

    check(res, {
      [`pipeline.pipeline.v1beta.PipelinePublicService/CreateUserPipeline response StatusOK`]:
        (r) => r.status === grpc.StatusOK,
    });

    check(
      clientPrivate.invoke(
        "pipeline.pipeline.v1beta.PipelinePrivateService/LookUpPipelineAdmin",
        {
          permalink: `pipelines/${res.message.pipeline.uid}`,
        }
      ),
      {
        [`pipeline.pipeline.v1beta.PipelinePrivateService/LookUpPipelineAdmin response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`pipeline.pipeline.v1beta.PipelinePrivateService/LookUpPipelineAdmin response pipeline new name`]:
          (r) => r.message.pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
      }
    );

    // Delete the pipeline
    check(
      clientPublic.invoke(
        `pipeline.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline`,
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
        },
        data.metadata
      ),
      {
        [`pipeline.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );

    clientPrivate.close();
    clientPublic.close();
  });
}

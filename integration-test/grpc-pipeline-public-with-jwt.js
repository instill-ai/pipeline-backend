import grpc from "k6/net/grpc";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";

const client = new grpc.Client();
client.load(["proto/vdp/pipeline/v1alpha"], "pipeline_public_service.proto");

export function CheckCreate() {
  group(
    `Pipelines API: Create a pipeline [with random "jwt-sub" header]`,
    () => {
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

      // Cannot create a pipeline of a non-exist user
      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline",
          {
            parent: `${constant.namespace}`,
            pipeline: reqBody,
          },
          constant.paramsGRPCWithJwt
        ),
        {
          [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusUnauthenticated`]:
            (r) => r.status === grpc.StatusUnauthenticated,
        }
      );

      client.close();
    }
  );
}

export function CheckList() {
  group(`Pipelines API: List pipelines [with random "jwt-sub" header]`, () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    // Cannot list pipelines of a non-exist user
    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines",
        {
          parent: `${constant.namespace}`,
        },
        constant.paramsGRPCWithJwt
      ),
      {
        [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines response StatusUnauthenticated`]:
          (r) => r.status === grpc.StatusUnauthenticated,
      }
    );

    client.close();
  });
}

export function CheckGet() {
  group(`Pipelines API: Get a pipeline [with random "jwt-sub" header]`, () => {
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
        }
      ),
      {
        [`vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );


    // Cannot get a pipeline of a non-exist user
    check(
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline",
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
        },
        constant.paramsGRPCWithJwt
      ),
      {
        [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/GetUserPipeline response StatusUnauthenticated`]:
          (r) => r.status === grpc.StatusUnauthenticated,
      }
    );

    // Delete the pipeline
    check(
      client.invoke(
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

    client.close();
  });
}

export function CheckUpdate() {
  group(
    `Pipelines API: Update a pipeline [with random "jwt-sub" header]`,
    () => {
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
        }
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

      // Cannot update a pipeline of a non-exist user
      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline",
          {
            pipeline: reqBodyUpdate,
            update_mask: "description",
          },
          constant.paramsGRPCWithJwt
        ),
        {
          [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserPipeline response StatusUnauthenticated`]:
            (r) => r.status === grpc.StatusUnauthenticated,
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
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

      client.close();
    }
  );
}

export function CheckRename() {
  group(
    `Pipelines API: Rename a pipeline [with random "jwt-sub" header]`,
    () => {
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
        }
      );

      check(res, {
        [`vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response pipeline name`]:
          (r) => r.message.pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
      });


      var new_pipeline_id = randomString(10);

      // Cannot rename a pipeline of a non-exist user
      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/RenameUserPipeline",
          {
            name: `${constant.namespace}/pipelines/${reqBody.id}`,
            new_pipeline_id: new_pipeline_id,
          },
          constant.paramsGRPCWithJwt
        ),
        {
          [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/RenameUserPipeline response StatusUnauthenticated`]:
            (r) => r.status === grpc.StatusUnauthenticated,
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
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

      client.close();
    }
  );
}

export function CheckLookUp() {
  group(
    `Pipelines API: Look up a pipeline by uid [with random "jwt-sub" header]`,
    () => {
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
        }
      );

      check(res, {
        [`vdp.pipeline.v1alpha.PipelinePublicService/CreateUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      });


      // Cannot look up a pipeline of a non-exist user
      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/LookUpUserPipeline",
          {
            permalink: `${constant.namespace}/pipelines/${res.message.pipeline.uid}`,
          },
          constant.paramsGRPCWithJwt
        ),
        {
          [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/LookUpUserPipeline response StatusUnauthenticated`]:
            (r) => r.status === grpc.StatusUnauthenticated,
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
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

      client.close();
    }
  );
}

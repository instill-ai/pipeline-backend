import grpc from "k6/net/grpc";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";

const client = new grpc.Client();
client.load(["../proto/vdp/pipeline/v1beta"], "pipeline_public_service.proto");

export function CheckCreate(data) {
  group(
    `Pipelines API: Create a pipeline [with random "Instill-User-Uid" header]`,
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqBody = Object.assign(
        {
          id: randomString(32),
          description: randomString(50),
        },
        constant.simpleRecipe
      );

      // Cannot create a pipeline of a non-exist user
      check(
        client.invoke(
          "vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline",
          {
            parent: `${constant.namespace}`,
            pipeline: reqBody,
          },
          constant.paramsGRPCWithJwt
        ),
        {
          [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline response StatusUnauthenticated`]:
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
        "vdp.pipeline.v1beta.PipelinePublicService/ListUserPipelines",
        {
          parent: `${constant.namespace}`,
        },
        constant.paramsGRPCWithJwt
      ),
      {
        [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/ListUserPipelines response StatusOK`]:
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

    var reqBody = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.simpleRecipe
    );

    check(
      client.invoke(
        "vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        data.metadata
      ),
      {
        [`vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );


    // Cannot get a pipeline of a non-exist user
    check(
      client.invoke(
        "vdp.pipeline.v1beta.PipelinePublicService/GetUserPipeline",
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
        },
        constant.paramsGRPCWithJwt
      ),
      {
        [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/GetUserPipeline response StatusNotFound`]:
          (r) => r.status === grpc.StatusNotFound,
      }
    );

    // Delete the pipeline
    check(
      client.invoke(
        `vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline`,
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
        },
        data.metadata
      ),
      {
        [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
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

      var reqBody = Object.assign(
        {
          id: randomString(10),
        },
        constant.simpleRecipe
      );

      // Create a pipeline
      var resOrigin = client.invoke(
        "vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        data.metadata
      );

      check(resOrigin, {
        [`vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline response StatusOK`]:
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
          "vdp.pipeline.v1beta.PipelinePublicService/UpdateUserPipeline",
          {
            pipeline: reqBodyUpdate,
            update_mask: "description",
          },
          constant.paramsGRPCWithJwt
        ),
        {
          [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/UpdateUserPipeline response StatusUnauthenticated`]:
            (r) => r.status === grpc.StatusUnauthenticated,
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
          `vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${reqBody.id}`,
          },
          data.metadata
        ),
        {
          [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
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

      var reqBody = Object.assign(
        {
          id: randomString(10),
        },
        constant.simpleRecipe
      );

      // Create a pipeline
      var res = client.invoke(
        "vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        data.metadata
      );

      check(res, {
        [`vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
        [`vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline response pipeline name`]:
          (r) => r.message.pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
      });


      var new_pipeline_id = randomString(10);

      // Cannot rename a pipeline of a non-exist user
      check(
        client.invoke(
          "vdp.pipeline.v1beta.PipelinePublicService/RenameUserPipeline",
          {
            name: `${constant.namespace}/pipelines/${reqBody.id}`,
            new_pipeline_id: new_pipeline_id,
          },
          constant.paramsGRPCWithJwt
        ),
        {
          [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/RenameUserPipeline response StatusUnauthenticated`]:
            (r) => r.status === grpc.StatusUnauthenticated,
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
          `vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${reqBody.id}`,
          },
          data.metadata
        ),
        {
          [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}

export function CheckLookUp(data) {
  group(
    `Pipelines API: Look up a pipeline by uid [with random "Instill-User-Uid" header]`,
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
        "vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        data.metadata
      );

      check(res, {
        [`vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      });


      // Cannot look up a pipeline of a non-exist user
      check(
        client.invoke(
          "vdp.pipeline.v1beta.PipelinePublicService/LookUpPipeline",
          {
            permalink: `pipelines/${res.message.pipeline.uid}`,
          },
          constant.paramsGRPCWithJwt
        ),
        {
          [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/LookUpPipeline response StatusUnauthenticated`]:
            (r) => r.status === grpc.StatusUnauthenticated,
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
          `vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${reqBody.id}`,
          },
          data.metadata
        ),
        {
          [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}

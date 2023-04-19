import http from "k6/http";
import grpc from 'k6/net/grpc';
import {
  check,
  group
} from "k6";
import {
  randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"
import * as helper from "./helper.js"

const clientPrivate = new grpc.Client();
const clientPublic = new grpc.Client();
clientPrivate.load(['proto/vdp/pipeline/v1alpha'], 'pipeline_private_service.proto');
clientPublic.load(['proto/vdp/pipeline/v1alpha'], 'pipeline_public_service.proto');

export function CheckList() {

  group("Pipelines API: List pipelines by admin", () => {

    clientPrivate.connect(constant.pipelineGRPCPrivateHost, {
      plaintext: true
    });

    clientPublic.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true
    });

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin', {}, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response totalSize is 0`]: (r) => r.message.totalSize == 0,
    });

    const numPipelines = 200
    var reqBodies = [];
    for (var i = 0; i < numPipelines; i++) {
      reqBodies[i] = Object.assign({
        id: randomString(10),
        description: randomString(50),
      },
        constant.detSyncHTTPSingleModelRecipe
      )
    }

    // Create pipelines
    for (const reqBody of reqBodies)
      check(clientPublic.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
        pipeline: reqBody
      }), {
        [`vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
      });

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin', {}, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response pipelines.length == 10`]: (r) => r.message.pipelines.length === 10,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response pipelines[0].recipe is null`]: (r) => r.message.pipelines[0].recipe === null,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response totalSize == 200`]: (r) => r.message.totalSize == 200,
    });

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin', {
      view: "VIEW_FULL"
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin view=VIEW_FULL response pipelines[0].recipe is valid`]: (r) => helper.validateRecipeGRPC(r.message.pipelines[0].recipe),
    });

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin', {
      view: "VIEW_BASIC"
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin view=VIEW_BASIC response pipelines[0].recipe is null`]: (r) => r.message.pipelines[0].recipe === null,
    });

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin', {
      pageSize: 3
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response pipelines.length == 3`]: (r) => r.message.pipelines.length === 3,
    });

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin', {
      pageSize: 101
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin response pipelines.length == 100`]: (r) => r.message.pipelines.length === 100,
    });


    var resFirst100 = clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin', {
      pageSize: 100
    }, {})
    var resSecond100 = clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin', {
      pageSize: 100,
      pageToken: resFirst100.message.nextPageToken
    }, {})
    check(resSecond100, {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin pageSize=100 pageToken=${resFirst100.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin pageSize=100 pageToken=${resFirst100.message.nextPageToken} response 100 results`]: (r) => r.message.pipelines.length === 100,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin pageSize=100 pageToken=${resFirst100.message.nextPageToken} nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
    });

    // Filtering
    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin', {
      filter: "mode=MODE_SYNC"
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: "mode=MODE_SYNC" response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: "mode=MODE_SYNC" response pipelines.length`]: (r) => r.message.pipelines.length > 0,
    });

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin', {
      filter: 'mode=MODE_SYNC AND state=STATE_ACTIVE'
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: mode=MODE_SYNC AND state=STATE_ACTIVE response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: mode=MODE_SYNC AND state=STATE_ACTIVE response pipelines.length`]: (r) => r.message.pipelines.length > 0,
    });

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin', {
      filter: 'state=STATE_ACTIVE AND create_time>timestamp("2000-06-19T23:31:08.657Z")'
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: state=STATE_ACTIVE AND create_time>timestamp("2000-06-19T23:31:08.657Z") response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: state=STATE_ACTIVE AND create_time>timestamp("2000-06-19T23:31:08.657Z") response pipelines.length`]: (r) => r.message.pipelines.length > 0,
    });

    // Get UUID for foreign resources
    var srcConnUid = http.get(`${constant.connectorPublicHost}/v1alpha/source-connectors/source-http`, {}, constant.params).json().source_connector.uid
    var srcConnPermalink = `source-connectors/${srcConnUid}`

    var dstConnUid = http.get(`${constant.connectorPublicHost}/v1alpha/destination-connectors/destination-http`, {}, constant.params).json().destination_connector.uid
    var dstConnPermalink = `destination-connectors/${dstConnUid}`

    var modelUid = http.get(`${constant.modelPublicHost}/v1alpha/models/${constant.model_id}`, {}, constant.params).json().model.uid
    var modelPermalink = `models/${modelUid}`

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin', {
      filter: `mode=MODE_SYNC AND recipe.source="${srcConnPermalink}"`
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: mode=MODE_SYNC AND recipe.source="${srcConnPermalink}" response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: mode=MODE_SYNC AND recipe.source="${srcConnPermalink}" response pipelines.length`]: (r) => r.message.pipelines.length > 0,
    });

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin', {
      filter: `mode=MODE_SYNC AND recipe.destination="${dstConnPermalink}" AND recipe.models:"${modelPermalink}"`
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: mode=MODE_SYNC AND recipe.destination="${dstConnPermalink}" AND recipe.models:"${modelPermalink}" response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/ListPipelinesAdmin filter: mode=MODE_SYNC AND recipe.destination="${dstConnPermalink}" AND recipe.models:"${modelPermalink}" response pipelines.length`]: (r) => r.message.pipelines.length > 0,
    });

    // Delete the pipelines
    for (const reqBody of reqBodies) {
      check(clientPublic.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
        name: `pipelines/${reqBody.id}`
      }), {
        [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
      });
    }

    clientPrivate.close()
    clientPublic.close();
  });
}

export function CheckGet() {

  group("Pipelines API: Get a pipeline by admin", () => {

    clientPrivate.connect(constant.pipelineGRPCPrivateHost, {
      plaintext: true
    });


    clientPublic.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true
    });

    var reqBody = Object.assign({
      id: randomString(10),
      description: randomString(50),
    },
      constant.detSyncHTTPSingleModelRecipe
    )

    check(clientPublic.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin', {
      name: `pipelines/${reqBody.id}`
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin name: pipelines/${reqBody.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin name: pipelines/${reqBody.id} response pipeline name`]: (r) => r.message.pipeline.name === `pipelines/${reqBody.id}`,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin name: pipelines/${reqBody.id} response pipeline uid`]: (r) => helper.isUUID(r.message.pipeline.uid),
      [`vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin name: pipelines/${reqBody.id} response pipeline id`]: (r) => r.message.pipeline.id === reqBody.id,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin name: pipelines/${reqBody.id} response pipeline description`]: (r) => r.message.pipeline.description === reqBody.description,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin name: pipelines/${reqBody.id} response pipeline recipe is null`]: (r) => r.message.pipeline.recipe === null,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin name: pipelines/${reqBody.id} response pipeline owner is UUID`]: (r) => helper.isValidOwner(r.message.pipeline.user),
    });

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin', {
      name: `pipelines/${reqBody.id}`,
      view: "VIEW_FULL"
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin name: pipelines/${reqBody.id} view: "VIEW_FULL" response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin name: pipelines/${reqBody.id} view: "VIEW_FULL" response pipeline recipe is null`]: (r) => r.message.pipeline.recipe !== null,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin name: pipelines/${reqBody.id} view: "VIEW_FULL" response pipeline owner is UUID`]: (r) => helper.isValidOwner(r.message.pipeline.user),
    });

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin', {
      name: `this-id-does-not-exist`,
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/GetPipelineAdmin name: this-id-does-not-exist response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
    });

    // Delete the pipeline
    check(clientPublic.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqBody.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    clientPrivate.close();
    clientPublic.close();
  });
}

export function CheckLookUp() {

  group("Pipelines API: Look up a pipeline by uid by admin", () => {

    clientPrivate.connect(constant.pipelineGRPCPrivateHost, {
      plaintext: true
    });

    clientPublic.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true
    });

    var reqBody = Object.assign({
      id: randomString(10),
    },
      constant.detSyncHTTPSingleModelRecipe
    )

    // Create a pipeline
    var res = clientPublic.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    })

    check(res, {
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    check(clientPrivate.invoke('vdp.pipeline.v1alpha.PipelinePrivateService/LookUpPipelineAdmin', {
      permalink: `pipelines/${res.message.pipeline.uid}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePrivateService/LookUpPipelineAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePrivateService/LookUpPipelineAdmin response pipeline new name`]: (r) => r.message.pipeline.name === `pipelines/${reqBody.id}`,
    });

    // Delete the pipeline
    check(clientPublic.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqBody.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    clientPrivate.close()
    clientPublic.close()
  });

}

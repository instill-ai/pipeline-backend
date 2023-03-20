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

const client = new grpc.Client();
client.load(['proto/vdp/pipeline/v1alpha'], 'pipeline_public_service.proto');

export function CheckCreate() {

  group("Pipelines API: Create a pipeline", () => {

    client.connect(constant.pipelineGRPCHost, {
      plaintext: true
    });

    var reqBody = Object.assign({
        id: randomString(63),
        description: randomString(50),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    )

    // Create a pipeline
    var resOrigin = client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    })
    check(resOrigin, {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response StatusOK": (r) => r.status === grpc.StatusOK,
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response pipeline name": (r) => r.message.pipeline.name === `pipelines/${reqBody.id}`,
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response pipeline name": (r) => r.message.pipeline.name === `pipelines/${reqBody.id}`,
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response pipeline uid": (r) => helper.isUUID(r.message.pipeline.uid),
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response pipeline id": (r) => r.message.pipeline.id === reqBody.id,
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response pipeline description": (r) => r.message.pipeline.description === reqBody.description,
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response pipeline recipe is valid": (r) => helper.validateRecipeGRPC(r.message.pipeline.recipe),
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response pipeline state ACTIVE": (r) => r.message.pipeline.state === "STATE_ACTIVE",
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response pipeline mode": (r) => r.message.pipeline.mode == "MODE_SYNC",
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response pipeline create_time": (r) => new Date(r.message.pipeline.createTime).getTime() > new Date().setTime(0),
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response pipeline update_time": (r) => new Date(r.message.pipeline.updateTime).getTime() > new Date().setTime(0)
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {}), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response StatusInvalidArgument": (r) => r.status === grpc.StatusInvalidArgument,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {}), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response StatusInvalidArgument": (r) => r.status === grpc.StatusInvalidArgument,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response StatusAlreadyExists": (r) => r.status === grpc.StatusAlreadyExists,
    });

    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqBody.id}`
    }), {
      [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${reqBody.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response StatusOK": (r) => r.status === grpc.StatusOK,
    });

    reqBody.id = null
    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline with null id response StatusInvalidArgument": (r) => r.status === grpc.StatusInvalidArgument,
    });

    reqBody.id = "abcd?*&efg!"
    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline with non-RFC-1034 naming id response StatusInvalidArgument": (r) => r.status === grpc.StatusInvalidArgument,
    });

    reqBody.id = randomString(64)
    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline with > 63-character id response StatusInvalidArgument": (r) => r.status === grpc.StatusInvalidArgument,
    });

    reqBody.id = "🧡💜我愛潤物科技💚💙"
    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline with non-ASCII id response StatusInvalidArgument": (r) => r.status === grpc.StatusInvalidArgument,
    });

    // Delete the pipeline
    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${resOrigin.message.pipeline.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    client.close();
  });
}

export function CheckList() {

  group("Pipelines API: List pipelines", () => {

    client.connect(constant.pipelineGRPCHost, {
      plaintext: true
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {}, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines response totalSize is 0`]: (r) => r.message.totalSize == 0,
    });

    const numPipelines = 200
    var reqBodies = [];
    for (var i = 0; i < numPipelines; i++) {
      reqBodies[i] = Object.assign({
          id: randomString(10),
          description: randomString(50),
        },
        constant.detSyncHTTPSingleModelInstRecipe
      )
    }

    // Create pipelines
    for (const reqBody of reqBodies)
      check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
        pipeline: reqBody
      }), {
        [`vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
      });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {}, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines response pipelines.length == 10`]: (r) => r.message.pipelines.length === 10,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines response pipelines[0].recipe is null`]: (r) => r.message.pipelines[0].recipe === null,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines response totalSize == 200`]: (r) => r.message.totalSize == 200,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
      view: "VIEW_FULL"
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines view=VIEW_FULL response pipelines[0].recipe is valid`]: (r) => helper.validateRecipeGRPC(r.message.pipelines[0].recipe),
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
      view: "VIEW_BASIC"
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines view=VIEW_BASIC response pipelines[0].recipe is null`]: (r) => r.message.pipelines[0].recipe === null,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
      pageSize: 3
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines response pipelines.length == 3`]: (r) => r.message.pipelines.length === 3,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
      pageSize: 101
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines response pipelines.length == 100`]: (r) => r.message.pipelines.length === 100,
    });


    var resFirst100 = client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
      pageSize: 100
    }, {})
    var resSecond100 = client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
      pageSize: 100,
      pageToken: resFirst100.message.nextPageToken
    }, {})
    check(resSecond100, {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines pageSize=100 pageToken=${resFirst100.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines pageSize=100 pageToken=${resFirst100.message.nextPageToken} response 100 results`]: (r) => r.message.pipelines.length === 100,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines pageSize=100 pageToken=${resFirst100.message.nextPageToken} nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
    });

    // Filtering
    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
      filter: "mode=MODE_SYNC"
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines filter: "mode=MODE_SYNC" response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines filter: "mode=MODE_SYNC" response pipelines.length`]: (r) => r.message.pipelines.length > 0,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
      filter: 'mode=MODE_SYNC AND state=STATE_ACTIVE'
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines filter: mode=MODE_SYNC AND state=STATE_ACTIVE response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines filter: mode=MODE_SYNC AND state=STATE_ACTIVE response pipelines.length`]: (r) => r.message.pipelines.length > 0,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
      filter: 'state=STATE_ACTIVE AND create_time>timestamp("2000-06-19T23:31:08.657Z")'
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines filter: state=STATE_ACTIVE AND create_time>timestamp("2000-06-19T23:31:08.657Z") response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines filter: state=STATE_ACTIVE AND create_time>timestamp("2000-06-19T23:31:08.657Z") response pipelines.length`]: (r) => r.message.pipelines.length > 0,
    });

    // Get UUID for foreign resources
    var srcConnUid = http.get(`${constant.connectorHost}/v1alpha/source-connectors/source-http`, {}, {
      headers: {
        "Content-Type": "application/json"
      },
    }).json().source_connector.uid
    var srcConnPermalink = `source-connectors/${srcConnUid}`

    var dstConnUid = http.get(`${constant.connectorHost}/v1alpha/destination-connectors/destination-http`, {}, {
      headers: {
        "Content-Type": "application/json"
      },
    }).json().destination_connector.uid
    var dstConnPermalink = `destination-connectors/${dstConnUid}`

    var modelUid = http.get(`${constant.modelHost}/v1alpha/models/${constant.model_id}`, {}, {
      headers: {
        "Content-Type": "application/json"
      },
    }).json().model.uid
    var modelInstUid = http.get(`${constant.modelHost}/v1alpha/models/${constant.model_id}/instances/latest`, {}, {
      headers: {
        "Content-Type": "application/json"
      },
    }).json().instance.uid
    var modelInstPermalink = `models/${modelUid}/instances/${modelInstUid}`

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
      filter: `mode=MODE_SYNC AND recipe.source="${srcConnPermalink}"`
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines filter: mode=MODE_SYNC AND recipe.source="${srcConnPermalink}" response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines filter: mode=MODE_SYNC AND recipe.source="${srcConnPermalink}" response pipelines.length`]: (r) => r.message.pipelines.length > 0,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
      filter: `mode=MODE_SYNC AND recipe.destination="${dstConnPermalink}" AND recipe.model_instances:"${modelInstPermalink}"`
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines filter: mode=MODE_SYNC AND recipe.destination="${dstConnPermalink}" AND recipe.model_instances:"${modelInstPermalink}" response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines filter: mode=MODE_SYNC AND recipe.destination="${dstConnPermalink}" AND recipe.model_instances:"${modelInstPermalink}" response pipelines.length`]: (r) => r.message.pipelines.length > 0,
    });

    // Delete the pipelines
    for (const reqBody of reqBodies) {
      check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
        name: `pipelines/${reqBody.id}`
      }), {
        [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
      });
    }

    client.close();
  });
}

export function CheckGet() {

  group("Pipelines API: Get a pipeline", () => {

    client.connect(constant.pipelineGRPCHost, {
      plaintext: true
    });

    var reqBody = Object.assign({
        id: randomString(10),
        description: randomString(50),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    )

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/GetPipeline', {
      name: `pipelines/${reqBody.id}`
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/GetPipeline name: pipelines/${reqBody.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/GetPipeline name: pipelines/${reqBody.id} response pipeline name`]: (r) => r.message.pipeline.name === `pipelines/${reqBody.id}`,
      [`vdp.pipeline.v1alpha.PipelinePublicService/GetPipeline name: pipelines/${reqBody.id} response pipeline uid`]: (r) => helper.isUUID(r.message.pipeline.uid),
      [`vdp.pipeline.v1alpha.PipelinePublicService/GetPipeline name: pipelines/${reqBody.id} response pipeline id`]: (r) => r.message.pipeline.id === reqBody.id,
      [`vdp.pipeline.v1alpha.PipelinePublicService/GetPipeline name: pipelines/${reqBody.id} response pipeline description`]: (r) => r.message.pipeline.description === reqBody.description,
      [`vdp.pipeline.v1alpha.PipelinePublicService/GetPipeline name: pipelines/${reqBody.id} response pipeline recipe is null`]: (r) => r.message.pipeline.recipe === null,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/GetPipeline', {
      name: `pipelines/${reqBody.id}`,
      view: "VIEW_FULL"
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/GetPipeline name: pipelines/${reqBody.id} view: "VIEW_FULL" response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/GetPipeline name: pipelines/${reqBody.id} view: "VIEW_FULL" response pipeline recipe is null`]: (r) => r.message.pipeline.recipe !== null,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/GetPipeline', {
      name: `this-id-does-not-exist`,
    }, {}), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/GetPipeline name: this-id-does-not-exist response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
    });

    // Delete the pipeline
    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqBody.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    client.close();
  });
}

export function CheckUpdate() {

  group("Pipelines API: Update a pipeline", () => {

    client.connect(constant.pipelineGRPCHost, {
      plaintext: true
    });

    var reqBody = Object.assign({
        id: randomString(10),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    )

    // Create a pipeline
    var resOrigin = client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    })

    check(resOrigin, {
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    var reqBodyUpdate = Object.assign({
      id: reqBody.id,
      name: `pipelines/${reqBody.id}`,
      uid: "output-only-to-be-ignored",
      mode: "MODE_ASYNC",
      description: randomString(50),
    }, )

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline', {
      pipeline: reqBodyUpdate,
      update_mask: "description"
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline response pipeline name (OUTPUT_ONLY)`]: (r) => r.message.pipeline.name === `pipelines/${resOrigin.message.pipeline.id}`,
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline response pipeline uid (OUTPUT_ONLY)`]: (r) => r.message.pipeline.uid === resOrigin.message.pipeline.uid,
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline response pipeline id (IMMUTABLE)`]: (r) => r.message.pipeline.id === resOrigin.message.pipeline.id,
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline response pipeline mode (OUTPUT_ONLY)`]: (r) => r.message.pipeline.mode === resOrigin.message.pipeline.mode,
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline response pipeline state (OUTPUT_ONLY)`]: (r) => r.message.pipeline.state === resOrigin.message.pipeline.state,
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline response pipeline description (OPTIONAL)`]: (r) => r.message.pipeline.description === reqBodyUpdate.description,
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline response pipeline recipe (IMMUTABLE)`]: (r) => r.message.pipeline.recipe !== null,
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline response pipeline createTime (OUTPUT_ONLY)`]: (r) => new Date(r.message.pipeline.createTime).getTime() > new Date().setTime(0),
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline response pipeline updateTime (OUTPUT_ONLY)`]: (r) => new Date(r.message.pipeline.updateTime).getTime() > new Date().setTime(0),
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline response pipeline updateTime > create_time`]: (r) => new Date(r.message.pipeline.updateTime).getTime() > new Date(r.message.pipeline.createTime).getTime()
    });

    reqBodyUpdate.description = ""
    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline', {
      pipeline: reqBodyUpdate,
      update_mask: "description"
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline response pipeline description empty`]: (r) => r.message.pipeline.description === "",
    });

    reqBodyUpdate.description = randomString(10)
    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline', {
      pipeline: reqBodyUpdate,
      update_mask: "description"
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline response pipeline description non-empty`]: (r) => r.message.pipeline.description === reqBodyUpdate.description,
    });

    reqBodyUpdate.id = randomString(10)
    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline', {
      pipeline: reqBodyUpdate,
      update_mask: "id"
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline updating IMMUTABLE field with different id response StatusInvalidArgument`]: (r) => r.status === grpc.StatusInvalidArgument,
    });

    reqBodyUpdate.id = reqBody.id
    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline', {
      pipeline: reqBodyUpdate,
      update_mask: "id"
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/UpdatePipeline updating IMMUTABLE field with the same id response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    // Delete the pipeline
    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqBody.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    client.close()
  });
}

export function CheckUpdateState() {

  group("Pipelines API: Update a pipeline state", () => {

    client.connect(constant.pipelineGRPCHost, {
      plaintext: true
    });

    var reqBodySync = Object.assign({
        id: randomString(10),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    )

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBodySync
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline Sync response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline Sync response pipeline state ACTIVE`]: (r) => r.message.pipeline.state === "STATE_ACTIVE",
    })

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/DeactivatePipeline', {
      name: `pipelines/${reqBodySync.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeactivatePipeline ${reqBodySync.id} response status is StatusInvalidArgument for sync pipeline`]: (r) => r.status === grpc.StatusInvalidArgument,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline', {
      name: `pipelines/${reqBodySync.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline ${reqBodySync.id} response status is StatusOK for sync pipeline`]: (r) => r.status === grpc.StatusOK,
    });

    var reqBodyAsync = Object.assign({
        id: randomString(10),
      },
      constant.detAsyncSingleModelInstRecipe
    )


    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBodyAsync
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline async response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline async response pipeline state ACTIVE`]: (r) => r.message.pipeline.state === "STATE_ACTIVE",
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline', {
      name: `pipelines/${reqBodyAsync.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline ${reqBodyAsync.id} response status is StatusOK for async pipeline`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline ${reqBodyAsync.id} response pipeline state ACTIVE`]: (r) => r.message.pipeline.state === "STATE_ACTIVE",
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/DeactivatePipeline', {
      name: `pipelines/${reqBodyAsync.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeactivatePipeline ${reqBodyAsync.id} response status is StatusOK for async pipeline`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeactivatePipeline ${reqBodyAsync.id} response pipeline state INACTIVE`]: (r) => r.message.pipeline.state === "STATE_INACTIVE",
    });

    // Delete the pipeline
    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqBodySync.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqBodyAsync.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    client.close()
  });
}

export function CheckRename() {

  group("Pipelines API: Rename a pipeline", () => {

    client.connect(constant.pipelineGRPCHost, {
      plaintext: true
    });

    var reqBody = Object.assign({
        id: randomString(10),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    )

    // Create a pipeline
    var res = client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    })

    check(res, {
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response pipeline name`]: (r) => r.message.pipeline.name === `pipelines/${reqBody.id}`,
    });

    reqBody.new_pipeline_id = randomString(10)
    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/RenamePipeline', {
      name: `pipelines/${reqBody.id}`,
      new_pipeline_id: reqBody.new_pipeline_id
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/RenamePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/RenamePipeline response pipeline new name`]: (r) => r.message.pipeline.name === `pipelines/${reqBody.new_pipeline_id}`,
      [`vdp.pipeline.v1alpha.PipelinePublicService/RenamePipeline response pipeline new id`]: (r) => r.message.pipeline.id === reqBody.new_pipeline_id,
    });

    // Delete the pipeline
    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqBody.new_pipeline_id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    client.close()
  });

}

export function CheckLookUp() {

  group("Pipelines API: Look up a pipeline by uid", () => {

    client.connect(constant.pipelineGRPCHost, {
      plaintext: true
    });

    var reqBody = Object.assign({
        id: randomString(10),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    )

    // Create a pipeline
    var res = client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    })

    check(res, {
      [`vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/LookUpPipeline', {
      permalink: `pipelines/${res.message.pipeline.uid}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/LookUpPipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/LookUpPipeline response pipeline new name`]: (r) => r.message.pipeline.name === `pipelines/${reqBody.id}`,
    });

    // Delete the pipeline
    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqBody.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    client.close()
  });

}
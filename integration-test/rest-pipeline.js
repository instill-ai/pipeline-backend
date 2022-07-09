import http from "k6/http";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

  group("Pipelines API: Create a pipeline", () => {

    var reqBody = Object.assign(
      {
        id: randomString(63),
        description: randomString(50),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    )

    // Create a pipeline
    var resOrigin = http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    })
    check(resOrigin, {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
      "POST /v1alpha/pipelines response pipeline name": (r) => r.json().pipeline.name === `pipelines/${reqBody.id}`,
      "POST /v1alpha/pipelines response pipeline uid": (r) => helper.isUUID(r.json().pipeline.uid),
      "POST /v1alpha/pipelines response pipeline id": (r) => r.json().pipeline.id === reqBody.id,
      "POST /v1alpha/pipelines response pipeline description": (r) => r.json().pipeline.description === reqBody.description,
      "POST /v1alpha/pipelines response pipeline recipe is valid": (r) => helper.validateRecipe(r.json().pipeline.recipe),
      "POST /v1alpha/pipelines response pipeline state ACTIVE": (r) => r.json().pipeline.state === "STATE_ACTIVE",
      "POST /v1alpha/pipelines response pipeline mode": (r) => r.json().pipeline.mode === "MODE_SYNC",
      "POST /v1alpha/pipelines response pipeline create_time": (r) => new Date(r.json().pipeline.create_time).getTime() > new Date().setTime(0),
      "POST /v1alpha/pipelines response pipeline update_time": (r) => new Date(r.json().pipeline.update_time).getTime() > new Date().setTime(0)
    });

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify({}), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines request body JSON Schema failed status 400": (r) => r.status === 400,
    });

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines re-create the same id response status is 409": (r) => r.status === 409
    });

    check(http.request("DELETE", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines re-create the same id after deletion response status is 201": (r) => r.status === 201
    });

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify({}), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines with empty body response status is 400": (r) => r.status === 400,
    });

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines with null body response status is 400": (r) => r.status === 400,
    });

    reqBody.id = null
    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines with null id response status is 400": (r) => r.status === 400,
    });

    reqBody.id = "abcd?*&efg!"
    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines with non-RFC-1034 naming id response status is 400": (r) => r.status === 400,
    });

    reqBody.id = randomString(64)
    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines with > 63-character id response status is 400": (r) => r.status === 400,
    });

    reqBody.id = "🧡💜我愛潤物科技💚💙"
    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines with non-ASCII id response status is 400": (r) => r.status === 400,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelineHost}/v1alpha/pipelines/${resOrigin.json().pipeline.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /v1alpha/pipelines/${resOrigin.json().pipeline.id} response status 204`]: (r) => r.status === 204,
    });

  });
}

export function CheckList() {

  group("Pipelines API: List pipelines", () => {

    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines`), {
      [`GET /v1alpha/pipelines response status is 200`]: (r) => r.status === 200,
      [`GET /v1alpha/pipelines response has pipelines array`]: (r) => Array.isArray(r.json().pipelines),
      [`GET /v1alpha/pipelines response has total_size 0`]: (r) => r.json().total_size == 0,
      [`GET /v1alpha/pipelines response has empty next_page_token`]: (r) => r.json().next_page_token == "",
    });

    const numPipelines = 200
    var reqBodies = [];
    for (var i = 0; i < numPipelines; i++) {
      reqBodies[i] = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.detSyncHTTPSingleModelInstRecipe
      )
    }

    // Create pipelines
    for (const reqBody of reqBodies) {
      check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
        headers: {
          "Content-Type": "application/json",
        },
      }), {
        [`POST /v1alpha/pipelines x${reqBodies.length} response status is 201`]: (r) => r.status === 201
      });
    }

    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /v1alpha/pipelines response status is 200`]: (r) => r.status === 200,
      [`GET /v1alpha/pipelines response pipelines.length == 10`]: (r) => r.json().pipelines.length == 10,
      [`GET /v1alpha/pipelines response pipelines[0] no recipe`]: (r) => r.json().pipelines[0].recipe === null,
      [`GET /v1alpha/pipelines response total_size == 200`]: (r) => r.json().total_size == 200,
    });

    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines?view=VIEW_FULL`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /v1alpha/pipelines?view=VIEW_FULL response pipelines[0] has recipe`]: (r) => r.json().pipelines[0].recipe !== null,
      [`GET /v1alpha/pipelines?view=VIEW_FULL response pipelines[0] recipe is valid`]: (r) => helper.validateRecipe(r.json().pipelines[0].recipe),
    });

    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines?view=VIEW_BASIC`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /v1alpha/pipelines?view=VIEW_BASIC response pipelines[0] has no recipe`]: (r) => r.json().pipelines[0].recipe === null,
    });

    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines?page_size=3`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /v1alpha/pipelines?page_size=3 response pipelines.length == 3`]: (r) => r.json().pipelines.length == 3,
    });

    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines?page_size=101`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /v1alpha/pipelines?page_size=101 response pipelines.length == 100`]: (r) => r.json().pipelines.length == 100,
    });

    var resFirst100 = http.request("GET", `${pipelineHost}/v1alpha/pipelines?page_size=100`)
    var resSecond100 = http.request("GET", `${pipelineHost}/v1alpha/pipelines?page_size=100&page_token=${resFirst100.json().next_page_token}`)
    check(resSecond100, {
      [`GET /v1alpha/pipelines?page_size=100&page_token=${resFirst100.json().next_page_token} response status 200`]: (r) => r.status == 200,
      [`GET /v1alpha/pipelines?page_size=100&page_token=${resFirst100.json().next_page_token} response return 100 results`]: (r) => r.json().pipelines.length == 100,
      [`GET /v1alpha/pipelines?page_size=100&page_token=${resFirst100.json().next_page_token} response next_page_token is empty`]: (r) => r.json().next_page_token == "",
    });

    // Filtering
    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines?filter=mode=MODE_SYNC`, null, {headers: {"Content-Type": "application/json",}}), {
      [`GET /v1alpha/pipelines?filter=mode=MODE_SYNC response 200`]: (r) => r.status == 200,
      [`GET /v1alpha/pipelines?filter=mode=MODE_SYNC response pipelines.length > 0`]: (r) => r.json().pipelines.length > 0,
    });

    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines?filter=mode=MODE_SYNC%20AND%20state=STATE_ACTIVE`, null, {headers: {"Content-Type": "application/json",}}), {
      [`GET /v1alpha/pipelines?filter=mode=MODE_SYNC%20AND%20state=STATE_ACTIVE response 200`]: (r) => r.status == 200,
      [`GET /v1alpha/pipelines?filter=mode=MODE_SYNC%20AND%20state=STATE_ACTIVE response pipelines.length > 0`]: (r) => r.json().pipelines.length > 0,
    });

    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines?filter=state=STATE_ACTIVE%20AND%20create_time>timestamp%28%222000-06-19T23:31:08.657Z%22%29`, null, {headers: {"Content-Type": "application/json",}}), {
      [`GET /v1alpha/pipelines?filter=state=STATE_ACTIVE%20AND%20create_time%20>%20timestamp%28%222000-06-19T23:31:08.657Z%22%29 response 200`]: (r) => r.status == 200,
      [`GET /v1alpha/pipelines?filter=state=STATE_ACTIVE%20AND%20create_time%20>%20timestamp%28%222000-06-19T23:31:08.657Z%22%29 response pipelines.length > 0`]: (r) => r.json().pipelines.length > 0,
    });

    // Get UUID for foreign resources
    var srcConnUid = http.get(`${connectorHost}/v1alpha/source-connectors/source-http`, {}, {headers: {"Content-Type": "application/json"},}).json().source_connector.uid
    var srcConnPermalink = `source-connectors/${srcConnUid}`

    var dstConnUid = http.get(`${connectorHost}/v1alpha/destination-connectors/destination-http`, {}, {headers: {"Content-Type": "application/json"},}).json().destination_connector.uid
    var dstConnPermalink = `destination-connectors/${dstConnUid}`

    var modelUid = http.get(`${modelHost}/v1alpha/models/${constant.model_id}`, {}, {headers: {"Content-Type": "application/json"},}).json().model.uid
    var modelInstUid = http.get(`${modelHost}/v1alpha/models/${constant.model_id}/instances/latest`, {}, {headers: {"Content-Type": "application/json"},}).json().instance.uid
    var modelInstPermalink = `models/${modelUid}/instances/${modelInstUid}`

    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines?filter=mode=MODE_SYNC%20AND%20recipe.source=%22${srcConnPermalink}%22`, null, {headers: {"Content-Type": "application/json",}}), {
      [`GET /v1alpha/pipelines?filter=mode=MODE_SYNC%20AND%20recipe.source=%22${srcConnPermalink}%22 response 200`]: (r) => r.status == 200,
      [`GET /v1alpha/pipelines?filter=mode=MODE_SYNC%20AND%20recipe.source=%22${srcConnPermalink}%22 response pipelines.length > 0`]: (r) => r.json().pipelines.length > 0,
    });

    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines?filter=mode=MODE_SYNC%20AND%20recipe.destination=%22${dstConnPermalink}%22%20AND%20recipe.model_instances:%22${modelInstPermalink}%22`, null, {headers: {"Content-Type": "application/json",}}), {
      [`GET /v1alpha/pipelines?filter=mode=MODE_SYNC%20AND%20recipe.source=%22${dstConnPermalink}%22%20AND%20recipe.model_instances:%22${modelInstPermalink}%22 response 200`]: (r) => r.status == 200,
      [`GET /v1alpha/pipelines?filter=mode=MODE_SYNC%20AND%20recipe.source=%22${dstConnPermalink}%22%20AND%20recipe.model_instances:%22${modelInstPermalink}%22 response pipelines.length > 0`]: (r) => r.json().pipelines.length > 0,
    });

    // Delete the pipelines
    for (const reqBody of reqBodies) {
      check(http.request(
        "DELETE",
        `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`,
        JSON.stringify(reqBody), {
        headers: {
          "Content-Type": "application/json",
        },
      }), {
        [`DELETE /v1alpha/pipelines x${reqBodies.length} response status is 204`]: (r) => r.status === 204,
      });
    }
  });
}

export function CheckGet() {

  group("Pipelines API: Get a pipeline", () => {

    var reqBody = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    )

    // Create a pipeline
    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`GET /v1alpha/pipelines/${reqBody.id} response status is 200`]: (r) => r.status === 200,
      [`GET /v1alpha/pipelines/${reqBody.id} response pipeline name`]: (r) => r.json().pipeline.name === `pipelines/${reqBody.id}`,
      [`GET /v1alpha/pipelines/${reqBody.id} response pipeline uid`]: (r) => helper.isUUID(r.json().pipeline.uid),
      [`GET /v1alpha/pipelines/${reqBody.id} response pipeline id`]: (r) => r.json().pipeline.id === reqBody.id,
      [`GET /v1alpha/pipelines/${reqBody.id} response pipeline description`]: (r) => r.json().pipeline.description === reqBody.description,
      [`GET /v1alpha/pipelines/${reqBody.id} response pipeline recipe`]: (r) => r.json().pipeline.recipe !== undefined,
    });

    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines/this-id-does-not-exist`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "GET /v1alpha/pipelines/this-id-does-not-exist response status is 404": (r) => r.status === 404,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /v1alpha/pipelines/${pipeline.id} response status 204`]: (r) => r.status === 204,
    });

  });
}

export function CheckUpdate() {

  group("Pipelines API: Update a pipeline", () => {

    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    )

    // Create a pipeline
    var resOrigin = http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    })

    check(resOrigin, {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    var reqBodyUpdate = Object.assign(
      {
        uid: "output-only-to-be-ignored",
        mode: "MODE_ASYNC",
        name: "pipelines/some-string-to-be-ignored",
        description: randomString(50),
      },
    )

    check(http.request("PATCH", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`, JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`PATCH /v1alpha/pipelines/${reqBody.id} response status is 200`]: (r) => r.status === 200,
      [`PATCH /v1alpha/pipelines/${reqBody.id} response pipeline name (OUTPUT_ONLY)`]: (r) => r.json().pipeline.name === `pipelines/${resOrigin.json().pipeline.id}`,
      [`PATCH /v1alpha/pipelines/${reqBody.id} response pipeline uid (OUTPUT_ONLY)`]: (r) => r.json().pipeline.uid === resOrigin.json().pipeline.uid,
      [`PATCH /v1alpha/pipelines/${reqBody.id} response pipeline id (IMMUTABLE)`]: (r) => r.json().pipeline.id === resOrigin.json().pipeline.id,
      [`PATCH /v1alpha/pipelines/${reqBody.id} response pipeline mode (OUTPUT_ONLY)`]: (r) => r.json().pipeline.mode === resOrigin.json().pipeline.mode,
      [`PATCH /v1alpha/pipelines/${reqBody.id} response pipeline state (OUTPUT_ONLY)`]: (r) => r.json().pipeline.state === resOrigin.json().pipeline.state,
      [`PATCH /v1alpha/pipelines/${reqBody.id} response pipeline description (OPTIONAL)`]: (r) => r.json().pipeline.description === reqBodyUpdate.description,
      [`PATCH /v1alpha/pipelines/${reqBody.id} response pipeline recipe (IMMUTABLE)`]: (r) => helper.deepEqual(r.json().pipeline.recipe, reqBody.recipe),
      [`PATCH /v1alpha/pipelines/${reqBody.id} response pipeline create_time (OUTPUT_ONLY)`]: (r) => new Date(r.json().pipeline.create_time).getTime() > new Date().setTime(0),
      [`PATCH /v1alpha/pipelines/${reqBody.id} response pipeline update_time (OUTPUT_ONLY)`]: (r) => new Date(r.json().pipeline.update_time).getTime() > new Date().setTime(0),
      [`PATCH /v1alpha/pipelines/${reqBody.id} response pipeline update_time > create_time`]: (r) => new Date(r.json().pipeline.update_time).getTime() > new Date(r.json().pipeline.create_time).getTime()
    });

    reqBodyUpdate.description = ""
    check(http.request("PATCH", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`,
      JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`PATCH /v1alpha/pipelines/${reqBody.id} response pipeline empty description`]: (r) => r.json().pipeline.description === reqBodyUpdate.description,
    });

    reqBodyUpdate.description = randomString(10)
    check(http.request("PATCH", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`,
      JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`PATCH /v1alpha/pipelines/${reqBody.id} response pipeline non-empty description`]: (r) => r.json().pipeline.description === reqBodyUpdate.description,
    });

    reqBodyUpdate.id = randomString(10)
    check(http.request("PATCH", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`, JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`PATCH /v1alpha/pipelines/${reqBody.id} response status for updating IMMUTABLE field with different id is 400`]: (r) => r.status === 400,
    });

    reqBodyUpdate.id = reqBody.id
    check(http.request("PATCH", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`, JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`PATCH /v1alpha/pipelines/${reqBody.id} response status for updating IMMUTABLE field with the same id is 200`]: (r) => r.status === 200,
    });

    check(http.request("PATCH", `${pipelineHost}/v1alpha/pipelines/this-id-does-not-exist`,
      JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "PATCH /v1alpha/pipelines/this-id-does-not-exist response status is 404": (r) => r.status === 404,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

  });
}

export function CheckUpdateState() {

  group("Pipelines API: Update a pipeline state", () => {

    var reqBodySync = Object.assign(
      {
        id: randomString(10),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    )

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBodySync), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines sync pipeline creation response status is 201": (r) => r.status === 201,
      "POST /v1alpha/pipelines sync pipeline creation response pipeline state ACTIVE": (r) => r.json().pipeline.state === "STATE_ACTIVE",
    });

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${reqBodySync.id}:deactivate`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBodySync.id}:deactivate response status is 400 for sync pipeline`]: (r) => r.status === 400,
    });

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${reqBodySync.id}:activate`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBodySync.id}:activate response status is 200 for sync pipeline`]: (r) => r.status === 200,
    });

    var reqBodyAsync = Object.assign(
      {
        id: randomString(10),
      },
      constant.detAsyncSingleModelInstRecipe
    )

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBodyAsync), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines async pipeline creation response status is 201": (r) => r.status === 201,
      "POST /v1alpha/pipelines async pipeline creation response pipeline state ACTIVE": (r) => r.json().pipeline.state === "STATE_ACTIVE",
    });

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${reqBodyAsync.id}:activate`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBodyAsync.id}:activate response status is 200 for async pipeline`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBodyAsync.id}:activate response pipeline state ACTIVE`]: (r) => r.json().pipeline.state === "STATE_ACTIVE",
    });

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${reqBodyAsync.id}:deactivate`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBodyAsync.id}:deactivate response status is 200 for async pipeline`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBodyAsync.id}:deactivate response pipeline state ACTIVE`]: (r) => r.json().pipeline.state === "STATE_INACTIVE",
    });

    // Delete the pipelines
    check(http.request("DELETE", `${pipelineHost}/v1alpha/pipelines/${reqBodySync.id}`, null, {
      headers: { "Content-Type": "application/json" },
    }), {
      [`DELETE /v1alpha/pipelines/${reqBodySync.id} response status 204`]: (r) => r.status === 204,
    });

    check(http.request("DELETE", `${pipelineHost}/v1alpha/pipelines/${reqBodyAsync.id}`, null, {
      headers: { "Content-Type": "application/json" },
    }), {
      [`DELETE /v1alpha/pipelines/${reqBodyAsync.id} response status 204`]: (r) => r.status === 204,
    });

  });
}

export function CheckRename() {

  group("Pipelines API: Rename a pipeline", () => {

    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    )

    // Create a pipeline
    var res = http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    })

    check(res, {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
      "POST /v1alpha/pipelines response pipeline name": (r) => r.json().pipeline.name === `pipelines/${reqBody.id}`,
    });

    reqBody.new_pipeline_id = randomString(10)
    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${res.json().pipeline.id}:rename`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${res.json().pipeline.id}:rename response status is 200"`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${res.json().pipeline.id}:rename response pipeline new name"`]: (r) => r.json().pipeline.name === `pipelines/${reqBody.new_pipeline_id}`,
      [`POST /v1alpha/pipelines/${res.json().pipeline.id}:rename response pipeline new id"`]: (r) => r.json().pipeline.id === reqBody.new_pipeline_id,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelineHost}/v1alpha/pipelines/${reqBody.new_pipeline_id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /v1alpha/pipelines/${reqBody.new_pipeline_id} response status 204`]: (r) => r.status === 204,
    });

  });

}

export function CheckLookUp() {

  group("Pipelines API: Look up a pipeline by uid", () => {

    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    )

    // Create a pipeline
    var res = http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    })

    check(res, {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    check(http.request("GET", `${pipelineHost}/v1alpha/pipelines/${res.json().pipeline.uid}:lookUp`), {
      [`GET /v1alpha/pipelines/${res.json().pipeline.uid}:lookUp response status is 200"`]: (r) => r.status === 200,
      [`GET /v1alpha/pipelines/${res.json().pipeline.uid}:lookUp response pipeline new name"`]: (r) => r.json().pipeline.name === `pipelines/${reqBody.id}`,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

  });

}

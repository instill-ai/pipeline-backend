import http from "k6/http";
import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js";
import * as helper from "./helper.js";

export function CheckCreate(data) {
  group("Pipelines API: Create a pipeline", () => {

    // Note: id is server-generated, so we don't include it in request body
    var reqBody = Object.assign(
      {
        description: randomString(50),
      },
      constant.simplePipelineWithYAMLRecipe
    );

    // Create a pipeline
    var resOrigin = http.request(
      "POST",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
      JSON.stringify(reqBody),
      data.header
    );
    check(resOrigin, {
      "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) => r.status === 201,
      "POST /v1beta/${constant.namespace}/pipelines response pipeline has id": (r) =>
        r.json().pipeline.id && r.json().pipeline.id.length > 0,
      "POST /v1beta/${constant.namespace}/pipelines response pipeline has name": (r) =>
        r.json().pipeline.name && r.json().pipeline.name.includes("/pipelines/"),
      "POST /v1beta/${constant.namespace}/pipelines response pipeline displayName": (r) =>
        r.json().pipeline.displayName === reqBody.displayName,
      "POST /v1beta/${constant.namespace}/pipelines response pipeline slug derived from displayName": (r) =>
        r.json().pipeline.slug === "integration-test-pipeline",
      "POST /v1beta/${constant.namespace}/pipelines response pipeline description": (r) =>
        r.json().pipeline.description === reqBody.description,
      "POST /v1beta/${constant.namespace}/pipelines response pipeline recipe is valid": (r) =>
        helper.validateRecipe(r.json().pipeline.recipe, false),
      "POST /v1beta/${constant.namespace}/pipelines response pipeline owner is valid": (r) =>
        helper.isValidOwner(r.json().pipeline.owner, data.expectedOwner),
      "POST /v1beta/${constant.namespace}/pipelines response pipeline creatorName is valid": (r) =>
        r.json().pipeline.creatorName && r.json().pipeline.creatorName.startsWith("users/"),
      "POST /v1beta/${constant.namespace}/pipelines response pipeline createTime": (r) =>
        new Date(r.json().pipeline.createTime).getTime() >
        new Date().setTime(0),
      "POST /v1beta/${constant.namespace}/pipelines response pipeline updateTime": (r) =>
        new Date(r.json().pipeline.updateTime).getTime() >
        new Date().setTime(0),
    });

    // Store the created pipeline id for later cleanup
    var createdPipelineId = resOrigin.json().pipeline.id;


    // Test empty body should fail
    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify({}),
        data.header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines with empty body response status is 400": (r) =>
          r.status === 400,
      }
    );

    // Test null body should fail
    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        null,
        data.header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines with null body response status is 400": (r) =>
          r.status === 400,
      }
    );

    // Delete the pipeline created at the start of this test
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${createdPipelineId}`,
        null,
        data.header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/${createdPipelineId} response status 204`]: (r) =>
          r.status === 204,
      }
    );
  });
}



export function CheckList(data) {
  group("Pipelines API: List pipelines", () => {
    // Record initial pipeline count (database might not be clean from previous runs)
    var initialRes = http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`, null, data.header);
    var initialCount = initialRes.status === 200 ? initialRes.json().totalSize : 0;
    check(initialRes, {
      [`GET /v1beta/${constant.namespace}/pipelines response status is 200`]: (r) =>
        r.status === 200,
      [`GET /v1beta/${constant.namespace}/pipelines response totalSize >= 0`]: (r) =>
        r.json().totalSize >= 0,
    });

    const numPipelines = 200;
    var createdPipelineIds = [];

    // Create pipelines and capture their IDs
    for (var i = 0; i < numPipelines; i++) {
      var reqBody = Object.assign(
        {
          description: randomString(50),
        },
        constant.simplePipelineWithYAMLRecipe
      );

      var createRes = http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        data.header
      );
      check(createRes, {
        [`POST /v1beta/${constant.namespace}/pipelines x${numPipelines} response status is 201`]:
          (r) => r.status === 201,
      }
      );
      if (createRes.status === 201) {
        createdPipelineIds.push(createRes.json().pipeline.id);
      }
    }

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        null,
        data.header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines response status is 200`]: (r) =>
          r.status === 200,
        [`GET /v1beta/${constant.namespace}/pipelines response pipelines.length == 10`]: (r) =>
          r.json().pipelines.length == 10,
        [`GET /v1beta/${constant.namespace}/pipelines response pipelines[0].recipe is null`]: (r) =>
          r.json().pipelines[0].recipe === null,
        [`GET /v1beta/${constant.namespace}/pipelines response totalSize >= 200`]: (r) =>
          r.json().totalSize >= 200,
        // Owner/Creator checks on LIST
        [`GET /v1beta/${constant.namespace}/pipelines response pipelines[0].owner is valid`]: (r) =>
          helper.isValidOwner(r.json().pipelines[0].owner, data.expectedOwner),
        [`GET /v1beta/${constant.namespace}/pipelines response pipelines[0].creatorName is valid`]: (r) =>
          r.json().pipelines[0].creatorName && r.json().pipelines[0].creatorName.startsWith("users/"),
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?view=VIEW_FULL`,
        null,
        data.header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines?view=VIEW_FULL response pipelines[0] has recipe`]:
          (r) => r.json().pipelines[0].recipe !== null,
        [`GET /v1beta/${constant.namespace}/pipelines?view=VIEW_FULL response pipelines[0] recipe is valid`]:
          (r) => helper.validateRecipe(r.json().pipelines[0].recipe, false),
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?view=VIEW_BASIC`,
        null,
        data.header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines?view=VIEW_BASIC response pipelines[0].recipe is null`]:
          (r) => r.json().pipelines[0].recipe === null,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?pageSize=3`,
        null,
        data.header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines?pageSize=3 response pipelines.length == 3`]: (
          r
        ) => r.json().pipelines.length == 3,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?pageSize=101`,
        null,
        data.header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines?pageSize=101 response pipelines.length == 100`]:
          (r) => r.json().pipelines.length == 100,
      }
    );

    var resFirst100 = http.request(
      "GET",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?pageSize=100`,
      null,
      data.header
    );
    var resSecond100 = http.request(
      "GET",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?pageSize=100&pageToken=${resFirst100.json().nextPageToken
      }`,
      null, data.header
    );
    check(resSecond100, {
      [`GET /v1beta/${constant.namespace}/pipelines?pageSize=100&pageToken=${resFirst100.json().nextPageToken
        } response status 200`]: (r) => r.status == 200,
      [`GET /v1beta/${constant.namespace}/pipelines?pageSize=100&pageToken=${resFirst100.json().nextPageToken
        } response return results`]: (r) => r.json().pipelines.length > 0,
    });

    // Filtering
    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        null,
        data.header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines response 200`]: (r) => r.status == 200,
        [`GET /v1beta/${constant.namespace}/pipelines response pipelines.length > 0`]: (r) =>
          r.json().pipelines.length > 0,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?filter=createTime>timestamp%28%222000-06-19T23:31:08.657Z%22%29`,
        null,
        data.header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines?filter=createTime%20>%20timestamp%28%222000-06-19T23:31:08.657Z%22%29 response 200`]:
          (r) => r.status == 200,
        [`GET /v1beta/${constant.namespace}/pipelines?filter=createTime%20>%20timestamp%28%222000-06-19T23:31:08.657Z%22%29 response pipelines.length > 0`]:
          (r) => r.json().pipelines.length > 0,
      }
    );

    // Delete the pipelines
    for (const pipelineId of createdPipelineIds) {
      check(
        http.request(
          "DELETE",
          `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
          null,
          data.header
        ),
        {
          [`DELETE /v1beta/${constant.namespace}/pipelines x${createdPipelineIds.length} response status is 204`]:
            (r) => r.status === 204,
        }
      );
    }
  });
}

export function CheckGet(data) {
  group("Pipelines API: Get a pipeline", () => {
    var reqBody = Object.assign(
      {
        description: randomString(50),
      },
      constant.simplePipelineWithYAMLRecipe
    );

    // Create a pipeline
    var createRes = http.request(
      "POST",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
      JSON.stringify(reqBody),
      data.header
    );
    check(createRes, {
      "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) =>
        r.status === 201,
    }
    );

    // Get the server-generated pipeline id
    var pipelineId = createRes.json().pipeline.id;

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
        null,
        data.header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines/{id} response status is 200`]: (r) =>
          r.status === 200,
        [`GET /v1beta/${constant.namespace}/pipelines/{id} response pipeline has name`]: (r) =>
          r.json().pipeline.name && r.json().pipeline.name.includes("/pipelines/"),
        [`GET /v1beta/${constant.namespace}/pipelines/{id} response pipeline has id`]: (r) =>
          r.json().pipeline.id && r.json().pipeline.id.length > 0,
        [`GET /v1beta/${constant.namespace}/pipelines/{id} response pipeline description`]:
          (r) => r.json().pipeline.description === reqBody.description,
        [`GET /v1beta/${constant.namespace}/pipelines/{id} response pipeline recipe is null`]:
          (r) => r.json().pipeline.recipe === null,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}?view=VIEW_FULL`,
        null,
        data.header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines/{id}?view=VIEW_FULL response status is 200`]: (r) =>
          r.status === 200,
        [`GET /v1beta/${constant.namespace}/pipelines/{id}?view=VIEW_FULL response pipeline recipe is not null`]:
          (r) => r.json().pipeline.recipe !== null,
        [`GET /v1beta/${constant.namespace}/pipelines/{id}?view=VIEW_FULL response pipeline owner is valid`]:
          (r) => helper.isValidOwner(r.json().pipeline.owner, data.expectedOwner),
        [`GET /v1beta/${constant.namespace}/pipelines/{id}?view=VIEW_FULL response pipeline creatorName is valid`]:
          (r) => r.json().pipeline.creatorName && r.json().pipeline.creatorName.startsWith("users/"),
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/this-id-does-not-exist`,
        null,
        data.header
      ),
      {
        "GET /v1beta/${constant.namespace}/pipelines/this-id-does-not-exist response status is 404":
          (r) => r.status === 404,
      }
    );

    // Delete the pipeline
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
        null,
        data.header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/{id} response status 204`]: (r) =>
          r.status === 204,
      }
    );
  });
}

export function CheckUpdate(data) {
  group("Pipelines API: Update a pipeline", () => {
    var reqBody = Object.assign(
      {},
      constant.simplePipelineWithYAMLRecipe
    );

    // Create a pipeline
    var resOrigin = http.request(
      "POST",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
      JSON.stringify(reqBody),
      data.header
    );

    check(resOrigin, {
      "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) => r.status === 201,
    });

    var pipelineId = resOrigin.json().pipeline.id;

    var reqBodyUpdate = Object.assign({
      name: "pipelines/some-string-to-be-ignored",
      description: randomString(50),
    });

    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
        JSON.stringify(reqBodyUpdate),
        data.header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/{id} response status is 200`]: (
          r
        ) => r.status === 200,
        // Note: Backend may return either users/admin or namespaces/admin format during transition
        [`PATCH /v1beta/${constant.namespace}/pipelines/{id} response pipeline name (OUTPUT_ONLY)`]:
          (r) =>
            r.json().pipeline.name &&
            r.json().pipeline.name.endsWith(`/pipelines/${resOrigin.json().pipeline.id}`),
        [`PATCH /v1beta/${constant.namespace}/pipelines/{id} response pipeline id (IMMUTABLE)`]:
          (r) => r.json().pipeline.id === resOrigin.json().pipeline.id,
        [`PATCH /v1beta/${constant.namespace}/pipelines/{id} response pipeline state (OUTPUT_ONLY)`]:
          (r) => r.json().pipeline.state === resOrigin.json().pipeline.state,
        [`PATCH /v1beta/${constant.namespace}/pipelines/{id} response pipeline description (OPTIONAL)`]:
          (r) => r.json().pipeline.description === reqBodyUpdate.description,
        [`PATCH /v1beta/${constant.namespace}/pipelines/{id} response pipeline owner isvalid`]:
          (r) => helper.isValidOwner(r.json().pipeline.owner, data.expectedOwner),
        [`PATCH /v1beta/${constant.namespace}/pipelines/{id} response pipeline createTime (OUTPUT_ONLY)`]:
          (r) =>
            new Date(r.json().pipeline.createTime).getTime() >
            new Date().setTime(0),
        [`PATCH /v1beta/${constant.namespace}/pipelines/{id} response pipeline updateTime (OUTPUT_ONLY)`]:
          (r) =>
            new Date(r.json().pipeline.updateTime).getTime() >
            new Date().setTime(0),
        [`PATCH /v1beta/${constant.namespace}/pipelines/{id} response pipeline updateTime > createTime`]:
          (r) =>
            new Date(r.json().pipeline.updateTime).getTime() >
            new Date(r.json().pipeline.createTime).getTime(),
      }
    );

    reqBodyUpdate.description = "";
    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
        JSON.stringify(reqBodyUpdate),
        data.header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/{id} response pipeline empty description`]:
          (r) => r.json().pipeline.description === reqBodyUpdate.description,
      }
    );

    reqBodyUpdate.description = randomString(10);
    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
        JSON.stringify(reqBodyUpdate),
        data.header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/{id} response pipeline non-empty description`]:
          (r) => r.json().pipeline.description === reqBodyUpdate.description,
      }
    );

    // Test updating with different id - since id is OUTPUT_ONLY, it should be ignored and return 200
    reqBodyUpdate.id = constant.dbIDPrefix + randomString(10);
    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
        JSON.stringify(reqBodyUpdate),
        data.header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/{id} response status when sending different id (OUTPUT_ONLY field ignored) is 200`]:
          (r) => r.status === 200,
      }
    );

    // Test updating with the same id should also succeed
    reqBodyUpdate.id = pipelineId;
    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
        JSON.stringify(reqBodyUpdate),
        data.header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/{id} response status for updating IMMUTABLE field with the same id is 200`]:
          (r) => r.status === 200,
      }
    );

    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/this-id-does-not-exist`,
        JSON.stringify(reqBodyUpdate),
        data.header
      ),
      {
        "PATCH /v1beta/${constant.namespace}/pipelines/this-id-does-not-exist response status is 404":
          (r) => r.status === 404,
      }
    );

    // Delete the pipeline
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
        null,
        data.header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/{id} response status 204`]: (r) =>
          r.status === 204,
      }
    );
  });
}


export function CheckRename(data) {
  group("Pipelines API: Rename a pipeline", () => {
    var reqBody = Object.assign(
      {},
      constant.simplePipelineWithYAMLRecipe
    );

    // Create a pipeline
    var res = http.request(
      "POST",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
      JSON.stringify(reqBody),
      data.header
    );

    check(res, {
      "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) => r.status === 201,
      "POST /v1beta/${constant.namespace}/pipelines response pipeline has name": (r) =>
        r.json().pipeline.name && r.json().pipeline.name.includes("/pipelines/"),
    });

    var pipelineId = res.json().pipeline.id;
    var newPipelineId = constant.dbIDPrefix + randomString(10);

    var renameBody = {
      new_pipeline_id: newPipelineId,
    };

    var renameRes = http.request(
      "POST",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}/rename`,
      JSON.stringify(renameBody),
      data.header
    );
    check(
      renameRes,
      {
        [`POST /v1beta/${constant.namespace}/pipelines/{id}/rename response status is 200`]: (r) => r.status === 200,
        // Note: Backend may return either users/admin or namespaces/admin format during transition
        [`POST /v1beta/${constant.namespace}/pipelines/{id}/rename response pipeline new name`]: (r) =>
          r.json().pipeline.name && r.json().pipeline.name.endsWith(`/pipelines/${newPipelineId}`),
        [`POST /v1beta/${constant.namespace}/pipelines/{id}/rename response pipeline new id`]: (r) =>
          r.json().pipeline.id === newPipelineId,
      }
    );

    // Delete the pipeline
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${newPipelineId}`,
        null,
        data.header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/{newId} response status 204`]:
          (r) => r.status === 204,
      }
    );
  });
}

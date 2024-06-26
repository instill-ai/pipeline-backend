import http from "k6/http";
import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js";
import * as helper from "./helper.js";

export function CheckCreate(data) {
  group("Pipelines API: Create a pipeline", () => {

    var reqBody = Object.assign(
      {
        id: randomString(32),
        description: randomString(50),
      },
      constant.simplePipelineWithJSONRecipe
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
      "POST /v1beta/${constant.namespace}/pipelines response pipeline name": (r) =>
        r.json().pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
      "POST /v1beta/${constant.namespace}/pipelines response pipeline uid": (r) =>
        helper.isUUID(r.json().pipeline.uid),
      "POST /v1beta/${constant.namespace}/pipelines response pipeline id": (r) =>
        r.json().pipeline.id === reqBody.id,
      "POST /v1beta/${constant.namespace}/pipelines response pipeline description": (r) =>
        r.json().pipeline.description === reqBody.description,
      "POST /v1beta/${constant.namespace}/pipelines response pipeline recipe is valid": (r) =>
        helper.validateRecipe(r.json().pipeline.recipe, false),
      "POST /v1beta/${constant.namespace}/pipelines response pipeline owner isinvalid": (r) =>
        helper.isValidOwner(r.json().pipeline.owner, data.expectedOwner),
      "POST /v1beta/${constant.namespace}/pipelines response pipeline createTime": (r) =>
        new Date(r.json().pipeline.createTime).getTime() >
        new Date().setTime(0),
      "POST /v1beta/${constant.namespace}/pipelines response pipeline updateTime": (r) =>
        new Date(r.json().pipeline.updateTime).getTime() >
        new Date().setTime(0),
    });


    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify({}),
        data.header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines request body JSON Schema failed status 400": (
          r
        ) => r.status === 400,
      }
    );

    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        data.header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines re-create the same id response status is 409":
          (r) => r.status === 409,
      }
    );

    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        null,
        data.header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status 204`]: (r) =>
          r.status === 204,
      }
    );

    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        data.header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines re-create the same id after deletion response status is 201":
          (r) => r.status === 201,
      }
    );

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

    reqBody.id = null;
    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        data.header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines with null id response status is 400": (r) =>
          r.status === 400,
      }
    );

    reqBody.id = "abcd?*&efg!";
    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        data.header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines with non-RFC-1034 naming id response status is 400":
          (r) => r.status === 400,
      }
    );

    reqBody.id = randomString(40);
    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        data.header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines with > 32-character id response status is 400":
          (r) => r.status === 400,
      }
    );

    reqBody.id = "🧡💜我愛潤物科技💚💙";
    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        data.header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines with non-ASCII id response status is 400": (
          r
        ) => r.status === 400,
      }
    );

    // Delete the pipeline
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${resOrigin.json().pipeline.id
        }`,
        null,
        data.header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/${resOrigin.json().pipeline.id
          } response status 204`]: (r) => r.status === 204,
      }
    );
  });
}



export function CheckList(data) {
  group("Pipelines API: List pipelines", () => {
    check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`, null, data.header), {
      [`GET /v1beta/${constant.namespace}/pipelines response status is 200`]: (r) =>
        r.status === 200,
      [`GET /v1beta/${constant.namespace}/pipelines response nextPageToken is empty`]: (r) =>
        r.json().nextPageToken === "",
      [`GET /v1beta/${constant.namespace}/pipelines response totalSize is 0`]: (r) =>
        r.json().totalSize == 0,
    });

    const numPipelines = 200;
    var reqBodies = [];
    for (var i = 0; i < numPipelines; i++) {
      reqBodies[i] = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.simplePipelineWithJSONRecipe
      );
    }

    // Create pipelines
    for (const reqBody of reqBodies) {
      check(
        http.request(
          "POST",
          `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
          JSON.stringify(reqBody),
          data.header
        ),
        {
          [`POST /v1beta/${constant.namespace}/pipelines x${reqBodies.length} response status is 201`]:
            (r) => r.status === 201,
        }
      );
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
        [`GET /v1beta/${constant.namespace}/pipelines response totalSize == 200`]: (r) =>
          r.json().totalSize == 200,
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
        } response return 100 results`]: (r) => r.json().pipelines.length == 100,
      [`GET /v1beta/${constant.namespace}/pipelines?pageSize=100&pageToken=${resFirst100.json().nextPageToken
        } response nextPageToken is empty`]: (r) =>
          r.json().nextPageToken === "",
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
    for (const reqBody of reqBodies) {
      check(
        http.request(
          "DELETE",
          `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
          JSON.stringify(reqBody),
          data.header
        ),
        {
          [`DELETE /v1beta/${constant.namespace}/pipelines x${reqBodies.length} response status is 204`]:
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
        id: randomString(10),
        description: randomString(50),
      },
      constant.simplePipelineWithJSONRecipe
    );

    // Create a pipeline
    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        data.header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) =>
          r.status === 201,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        null,
        data.header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status is 200`]: (r) =>
          r.status === 200,
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline name`]: (r) =>
          r.json().pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline uid`]: (r) =>
          helper.isUUID(r.json().pipeline.uid),
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline id`]: (r) =>
          r.json().pipeline.id === reqBody.id,
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline description`]:
          (r) => r.json().pipeline.description === reqBody.description,
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline recipe is null`]:
          (r) => r.json().pipeline.recipe === null,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}?view=VIEW_FULL`,
        null,
        data.header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status is 200`]: (r) =>
          r.status === 200,
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline recipe is not null`]:
          (r) => r.json().pipeline.recipe !== null,
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline owner isvalid`]:
          (r) => helper.isValidOwner(r.json().pipeline.owner, data.expectedOwner),
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
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        null,
        data.header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status 204`]: (r) =>
          r.status === 204,
      }
    );
  });
}

export function CheckUpdate(data) {
  group("Pipelines API: Update a pipeline", () => {
    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.simplePipelineWithJSONRecipe
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

    var reqBodyUpdate = Object.assign({
      uid: "output-only-to-be-ignored",
      name: "pipelines/some-string-to-be-ignored",
      description: randomString(50),
    });

    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        JSON.stringify(reqBodyUpdate),
        data.header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status is 200`]: (
          r
        ) => r.status === 200,
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline name (OUTPUT_ONLY)`]:
          (r) =>
            r.json().pipeline.name ===
            `${constant.namespace}/pipelines/${resOrigin.json().pipeline.id}`,
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline uid (OUTPUT_ONLY)`]:
          (r) => r.json().pipeline.uid === resOrigin.json().pipeline.uid,
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline id (IMMUTABLE)`]:
          (r) => r.json().pipeline.id === resOrigin.json().pipeline.id,
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline state (OUTPUT_ONLY)`]:
          (r) => r.json().pipeline.state === resOrigin.json().pipeline.state,
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline description (OPTIONAL)`]:
          (r) => r.json().pipeline.description === reqBodyUpdate.description,
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline owner isvalid`]:
          (r) => helper.isValidOwner(r.json().pipeline.owner, data.expectedOwner),
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline createTime (OUTPUT_ONLY)`]:
          (r) =>
            new Date(r.json().pipeline.createTime).getTime() >
            new Date().setTime(0),
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline updateTime (OUTPUT_ONLY)`]:
          (r) =>
            new Date(r.json().pipeline.updateTime).getTime() >
            new Date().setTime(0),
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline updateTime > createTime`]:
          (r) =>
            new Date(r.json().pipeline.updateTime).getTime() >
            new Date(r.json().pipeline.createTime).getTime(),
      }
    );

    reqBodyUpdate.description = "";
    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        JSON.stringify(reqBodyUpdate),
        data.header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline empty description`]:
          (r) => r.json().pipeline.description === reqBodyUpdate.description,
      }
    );

    reqBodyUpdate.description = randomString(10);
    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        JSON.stringify(reqBodyUpdate),
        data.header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline non-empty description`]:
          (r) => r.json().pipeline.description === reqBodyUpdate.description,
      }
    );

    reqBodyUpdate.id = randomString(10);
    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        JSON.stringify(reqBodyUpdate),
        data.header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status for updating IMMUTABLE field with different id is 400`]:
          (r) => r.status === 400,
      }
    );

    reqBodyUpdate.id = reqBody.id;
    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        JSON.stringify(reqBodyUpdate),
        data.header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status for updating IMMUTABLE field with the same id is 200`]:
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
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        null,
        data.header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status 204`]: (r) =>
          r.status === 204,
      }
    );
  });
}


export function CheckRename(data) {
  group("Pipelines API: Rename a pipeline", () => {
    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.simplePipelineWithJSONRecipe
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
      "POST /v1beta/${constant.namespace}/pipelines response pipeline name": (r) =>
        r.json().pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
    });

    reqBody.new_pipeline_id = randomString(10);

    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${res.json().pipeline.id
        }/rename`,
        JSON.stringify(reqBody),
        data.header
      ),
      {
        [`POST /v1beta/${constant.namespace}/pipelines/${res.json().pipeline.id
          }/rename response status is 200"`]: (r) => r.status === 200,
        [`POST /v1beta/${constant.namespace}/pipelines/${res.json().pipeline.id
          }/rename response pipeline new name"`]: (r) =>
            r.json().pipeline.name === `${constant.namespace}/pipelines/${reqBody.new_pipeline_id}`,
        [`POST /v1beta/${constant.namespace}/pipelines/${res.json().pipeline.id
          }/rename response pipeline new id"`]: (r) =>
            r.json().pipeline.id === reqBody.new_pipeline_id,
      }
    );

    // Delete the pipeline
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.new_pipeline_id}`,
        null,
        data.header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/${reqBody.new_pipeline_id} response status 204`]:
          (r) => r.status === 204,
      }
    );
  });
}

export function CheckLookUp(data) {
  group("Pipelines API: Look up a pipeline by uid", () => {
    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.simplePipelineWithJSONRecipe
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
    });

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/pipelines/${res.json().pipeline.uid
        }/lookUp`,
        null,
        data.header
      ),
      {
        [`GET /v1beta/pipelines/${res.json().pipeline.uid
          }/lookUp response status is 200"`]: (r) => r.status === 200,
        [`GET /v1beta/pipelines/${res.json().pipeline.uid
          }/lookUp response pipeline new name"`]: (r) =>
            r.json().pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
      }
    );

    // Delete the pipeline
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        null,
        data.header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status 204`]: (r) =>
          r.status === 204,
      }
    );
  });
}

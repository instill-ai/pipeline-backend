import http from "k6/http";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import {
  pipelinePrivateHost,
  pipelinePublicHost,
  connectorPublicHost,
} from "./const.js";

import * as constant from "./const.js";
import * as helper from "./helper.js";

export function CheckList() {
  group("Pipelines API: List pipelines by admin", () => {
    check(
      http.request("GET", `${pipelinePrivateHost}/v1alpha/admin/pipelines`),
      {
        [`GET /v1alpha/admin/pipelines response status is 200`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/admin/pipelines response next_page_token is empty`]: (
          r
        ) => r.json().next_page_token === "",
        [`GET /v1alpha/admin/pipelines response total_size is 0`]: (r) =>
          r.json().total_size == 0,
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
        constant.detSyncHTTPSimpleRecipe
      );
    }

    // Create pipelines
    for (const reqBody of reqBodies) {
      check(
        http.request(
          "POST",
          `${pipelinePublicHost}/v1alpha/pipelines`,
          JSON.stringify(reqBody),
          constant.params
        ),
        {
          [`POST /v1alpha/pipelines x${reqBodies.length} response status is 201`]:
            (r) => r.status === 201,
        }
      );
      http.request(
        "POST",
        `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/activate`,
        {},
        constant.params
      );
    }

    check(
      http.request(
        "GET",
        `${pipelinePrivateHost}/v1alpha/admin/pipelines`,
        null,
        constant.params
      ),
      {
        [`GET /v1alpha/admin/pipelines response status is 200`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/admin/pipelines response pipelines.length == 10`]: (r) =>
          r.json().pipelines.length == 10,
        [`GET /v1alpha/admin/pipelines response pipelines[0].recipe is null`]: (
          r
        ) => r.json().pipelines[0].recipe === null,
        [`GET /v1alpha/admin/pipelines response total_size == 200`]: (r) =>
          r.json().total_size == 200,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePrivateHost}/v1alpha/admin/pipelines?view=VIEW_FULL`,
        null,
        constant.params
      ),
      {
        [`GET /v1alpha/admin/pipelines?view=VIEW_FULL response pipelines[0] has recipe`]:
          (r) => r.json().pipelines[0].recipe !== null,
        [`GET /v1alpha/admin/pipelines?view=VIEW_FULL response pipelines[0] recipe is valid`]:
          (r) => helper.validateRecipe(r.json().pipelines[0].recipe, true),
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePrivateHost}/v1alpha/admin/pipelines?view=VIEW_BASIC`,
        null,
        constant.params
      ),
      {
        [`GET /v1alpha/admin/pipelines?view=VIEW_BASIC response pipelines[0].recipe is null`]:
          (r) => r.json().pipelines[0].recipe === null,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePrivateHost}/v1alpha/admin/pipelines?page_size=3`,
        null,
        constant.params
      ),
      {
        [`GET /v1alpha/admin/pipelines?page_size=3 response pipelines.length == 3`]:
          (r) => r.json().pipelines.length == 3,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePrivateHost}/v1alpha/admin/pipelines?page_size=101`,
        null,
        constant.params
      ),
      {
        [`GET /v1alpha/admin/pipelines?page_size=101 response pipelines.length == 100`]:
          (r) => r.json().pipelines.length == 100,
      }
    );

    var resFirst100 = http.request(
      "GET",
      `${pipelinePrivateHost}/v1alpha/admin/pipelines?page_size=100`
    );
    var resSecond100 = http.request(
      "GET",
      `${pipelinePrivateHost}/v1alpha/admin/pipelines?page_size=100&page_token=${resFirst100.json().next_page_token
      }`
    );
    check(resSecond100, {
      [`GET /v1alpha/admin/pipelines?page_size=100&page_token=${resFirst100.json().next_page_token
        } response status 200`]: (r) => r.status == 200,
      [`GET /v1alpha/admin/pipelines?page_size=100&page_token=${resFirst100.json().next_page_token
        } response return 100 results`]: (r) => r.json().pipelines.length == 100,
      [`GET /v1alpha/admin/pipelines?page_size=100&page_token=${resFirst100.json().next_page_token
        } response next_page_token is empty`]: (r) =>
          r.json().next_page_token === "",
    });

    // Filtering
    check(
      http.request(
        "GET",
        `${pipelinePrivateHost}/v1alpha/admin/pipelines`,
        null,
        constant.params
      ),
      {
        [`GET /v1alpha/admin/pipelines response 200`]: (r) => r.status == 200,
        [`GET /v1alpha/admin/pipelines response pipelines.length > 0`]: (r) =>
          r.json().pipelines.length > 0,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePrivateHost}/v1alpha/admin/pipelines?filter=state=STATE_ACTIVE`,
        null,
        constant.params
      ),
      {
        [`GET /v1alpha/admin/pipelines?filter=state=STATE_ACTIVE response 200`]:
          (r) => r.status == 200,
        [`GET /v1alpha/admin/pipelines?filter=state=STATE_ACTIVE response pipelines.length > 0`]:
          (r) => r.json().pipelines.length > 0,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePrivateHost}/v1alpha/admin/pipelines?filter=state=STATE_ACTIVE%20AND%20create_time>timestamp%28%222000-06-19T23:31:08.657Z%22%29`,
        null,
        constant.params
      ),
      {
        [`GET /v1alpha/admin/pipelines?filter=state=STATE_ACTIVE%20AND%20create_time%20>%20timestamp%28%222000-06-19T23:31:08.657Z%22%29 response 200`]:
          (r) => r.status == 200,
        [`GET /v1alpha/admin/pipelines?filter=state=STATE_ACTIVE%20AND%20create_time%20>%20timestamp%28%222000-06-19T23:31:08.657Z%22%29 response pipelines.length > 0`]:
          (r) => r.json().pipelines.length > 0,
      }
    );

    // Get UUID for foreign resources
    var srcConnUid = http
      .get(
        `${connectorPublicHost}/v1alpha/connectors/trigger`,
        {},
        constant.params
      )
      .json().connector.uid;
    var srcConnPermalink = `connectors/${srcConnUid}`;

    var dstConnUid = http
      .get(
        `${connectorPublicHost}/v1alpha/connectors/response`,
        {},
        constant.params
      )
      .json().connector.uid;
    var dstConnPermalink = `connectors/${dstConnUid}`;

    // var modelUid = http.get(`${modelPublicHost}/v1alpha/models/${constant.model_id}`, {}, constant.params).json().model.uid
    // var modelPermalink = `models/${modelUid}`

    check(
      http.request(
        "GET",
        `${pipelinePrivateHost}/v1alpha/admin/pipelines?filter=recipe.components.resource_name:%22${srcConnPermalink}%22`,
        null,
        constant.params
      ),
      {
        [`GET /v1alpha/admin/pipelines?filter=recipe.components.resource_name:%22${srcConnPermalink}%22 response 200`]:
          (r) => r.status == 200,
        [`GET /v1alpha/admin/pipelines?filter=recipe.components.resource_name:%22${srcConnPermalink}%22 response pipelines.length > 0`]:
          (r) => r.json().pipelines.length > 0,
      }
    );

    // check(http.request("GET", `${pipelinePrivateHost}/v1alpha/admin/pipelines?filter=mode=MODE_SYNC%20AND%20recipe.components.resource_name:%22${dstConnPermalink}%22%20AND%20recipe.components.resource_name:%22${modelPermalink}%22`, null, constant.params), {
    //   [`GET /v1alpha/admin/pipelines?filter=mode=MODE_SYNC%20AND%20recipe.components.resource_name:%22${dstConnPermalink}%22%20AND%20recipe.components.resource_name:%22${modelPermalink}%22 response 200`]: (r) => r.status == 200,
    //   [`GET /v1alpha/admin/pipelines?filter=mode=MODE_SYNC%20AND%20recipe.components.resource_name:%22${dstConnPermalink}%22%20AND%20recipe.components.resource_name:%22${modelPermalink}%22 response pipelines.length > 0`]: (r) => r.json().pipelines.length > 0,
    // });

    // Delete the pipelines
    for (const reqBody of reqBodies) {
      check(
        http.request(
          "DELETE",
          `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`,
          JSON.stringify(reqBody),
          constant.params
        ),
        {
          [`DELETE /v1alpha/pipelines x${reqBodies.length} response status is 204`]:
            (r) => r.status === 204,
        }
      );
    }
  });
}

export function CheckLookUp() {
  group("Pipelines API: Look up a pipeline by uid by admin", () => {
    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.detSyncHTTPSimpleRecipe
    );

    // Create a pipeline
    var res = http.request(
      "POST",
      `${pipelinePublicHost}/v1alpha/pipelines`,
      JSON.stringify(reqBody),
      constant.params
    );

    check(res, {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    check(
      http.request(
        "GET",
        `${pipelinePrivateHost}/v1alpha/admin/pipelines/${res.json().pipeline.uid
        }/lookUp`
      ),
      {
        [`GET /v1alpha/admin/pipelines/${res.json().pipeline.uid
          }/lookUp response status is 200"`]: (r) => r.status === 200,
        [`GET /v1alpha/admin/pipelines/${res.json().pipeline.uid
          }/lookUp response pipeline new name"`]: (r) =>
            r.json().pipeline.name === `pipelines/${reqBody.id}`,
      }
    );

    // Delete the pipeline
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`,
        null,
        constant.params
      ),
      {
        [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) =>
          r.status === 204,
      }
    );
  });
}

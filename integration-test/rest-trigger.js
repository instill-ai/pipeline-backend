import http from "k6/http";

import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js";

const recipeWithoutSetup = `
version: v1beta
variable:
  recipients:
    type: array:string
output:
  resp:
    title: Response
    value: \${email-0.output.result}
component:
  email-0:
    type: email
    input:
      recipients: \${variable.recipients}
      cc: null
      bcc: null
      subject: "Dummy email"
      message: "Hi I'm testing integrations"
    condition: null
    task: TASK_SEND_EMAIL
`;

var collectionPath = `/v1beta/namespaces/${constant.defaultUsername}/pipelines`;

function resourcePath(id) {
  return `${collectionPath}/${id}`;
}

function triggerPath(id) {
  return resourcePath(id) + "/trigger";
}

export function CheckTrigger(data) {
  group("Pipelines API: Trigger a pipeline", () => {
    var reqHTTP = Object.assign(
      {
        description: randomString(50),
      },
      constant.simplePipelineWithYAMLRecipe
    );

    var createRes = http.request("POST", pipelinePublicHost + collectionPath, JSON.stringify(reqHTTP), data.header);
    check(createRes, {
      [`POST ${collectionPath} response status is 201 (HTTP pipeline)`]: (r) => r.status === 201,
    });

    var pipelineId = createRes.json().pipeline ? createRes.json().pipeline.id : null;
    if (!pipelineId) {
      console.log("Failed to create pipeline, skipping trigger test");
      return;
    }

    check(http.request("POST", pipelinePublicHost + triggerPath(pipelineId), JSON.stringify(constant.simplePayload), data.header), {
      [`POST ${triggerPath(pipelineId)} response status is 200`]: (r) => r.status === 200,
    });

    check(http.request("DELETE", pipelinePublicHost + resourcePath(pipelineId), null, data.header), {
      [`DELETE ${resourcePath(pipelineId)} response status 204`]: (r) => r.status === 204,
    });
  });

  group("Pipelines API: Trigger a pipeline with YAML recipe", () => {
    var reqHTTP = Object.assign(
      {
        description: randomString(50),
      },
      constant.simplePipelineWithYAMLRecipe
    );

    var createRes = http.request("POST", pipelinePublicHost + collectionPath, JSON.stringify(reqHTTP), data.header);
    check(createRes, {
      [`POST ${collectionPath} response status is 201 (YAML pipeline)`]: (r) => r.status === 201,
    });

    var pipelineId = createRes.json().pipeline ? createRes.json().pipeline.id : null;
    if (!pipelineId) {
      console.log("Failed to create pipeline, skipping trigger test");
      return;
    }

    check(http.request("POST", pipelinePublicHost + triggerPath(pipelineId), JSON.stringify(constant.simplePayload), data.header), {
      [`POST ${triggerPath(pipelineId)} response status is 200`]: (r) => r.status === 200,
    });

    check(http.request("DELETE", pipelinePublicHost + resourcePath(pipelineId), null, data.header), {
      [`DELETE ${resourcePath(pipelineId)} response status 204`]: (r) => r.status === 204,
    });
  });

  group("Pipelines API: Validate pipeline on trigger", () => {
    const payload = {
      data: [{
        variable: {recipients: ["a", "b"]},
      }],
    };

    const missingConnRecipe = `${recipeWithoutSetup}
    setup: \${connection.my-conn}`;

    var reqMiss = {
      description: randomString(10),
      rawRecipe: missingConnRecipe,
      displayName: "Missing Connection Test",
      visibility: "VISIBILITY_PRIVATE",
    };

    var missRes = http.request("POST", pipelinePublicHost + collectionPath, JSON.stringify(reqMiss), data.header);
    check(missRes, {
      [`POST ${collectionPath} (missing-conn) response status is 201`]: (r) => r.status === 201,
    });

    var missId = missRes.json().pipeline ? missRes.json().pipeline.id : null;
    if (missId) {
      check(http.request("POST", pipelinePublicHost + triggerPath(missId), JSON.stringify(payload), data.header), {
        [`POST ${triggerPath(missId)} response status is 400`]: (r) => r.status === 400,
        [`POST ${triggerPath(missId)} contains end-user message`]:
          (r) => r.json().message === "Connection my-conn doesn't exist.",
      });

      check(http.request("DELETE", pipelinePublicHost + resourcePath(missId), null, data.header), {
        [`DELETE ${resourcePath(missId)} response status 204`]: (r) => r.status === 204,
      });
    }

    const invalidRefRecipe = `${recipeWithoutSetup}
    setup: \${connnnnnection.my-conn}`;

    var reqInvalid = {
      description: randomString(10),
      rawRecipe: invalidRefRecipe,
      displayName: "Invalid Ref Test",
      visibility: "VISIBILITY_PRIVATE",
    };

    var invalidRes = http.request("POST", pipelinePublicHost + collectionPath, JSON.stringify(reqInvalid), data.header);
    check(invalidRes, {
      [`POST ${collectionPath} (invalid-ref) response status is 201`]: (r) => r.status === 201,
    });

    var invalidId = invalidRes.json().pipeline ? invalidRes.json().pipeline.id : null;
    if (invalidId) {
      check(http.request("POST", pipelinePublicHost + triggerPath(invalidId), JSON.stringify(payload), data.header), {
        [`POST ${triggerPath(invalidId)} response status is 400`]: (r) => r.status === 400,
        [`POST ${triggerPath(invalidId)} contains end-user message`]: (r) =>
          r.json().message === "String setup only supports connection references (${connection.<conn-id>}).",
      });

      check(http.request("DELETE", pipelinePublicHost + resourcePath(invalidId), null, data.header), {
        [`DELETE ${resourcePath(invalidId)} response status 204`]: (r) => r.status === 204,
      });
    }
  });
}

const breakableRecipe = `
version: v1.0-alpha
variable:
  jota:
    type: json
component:
  jq:
    type: json
    task: TASK_JQ
    input:
      jq-filter: '.foo'
      json-value: \${variable.jota}
output:
  out:
    value: \${jq.output.results[0]}
`;

export function CheckPipelineRuns(data) {
  group("Pipelines API: View pipeline and component runs", () => {
    const creationReq = {
      description: randomString(50),
      rawRecipe: breakableRecipe,
      displayName: "Pipeline Runs Test",
      visibility: "VISIBILITY_PRIVATE",
    };

    // Create pipeline
    var createRes = http.request(
      "POST",
      pipelinePublicHost + collectionPath,
      JSON.stringify(creationReq),
      data.header
    );
    check(createRes, {
      [`POST ${collectionPath} response status is 201 (HTTP pipeline)`]: (r) =>
        r.status === 201,
    });

    var pipelineId = createRes.json().pipeline ? createRes.json().pipeline.id : null;
    if (!pipelineId) {
      console.log("Failed to create pipeline, skipping pipeline runs test");
      return;
    }

    // Trigger pipeline with error
    const nokPayload = JSON.stringify({
      data: [
        {
          variable: {},
        },
      ],
    });

    const nokResp = http.request(
      "POST",
      pipelinePublicHost + triggerPath(pipelineId),
      nokPayload,
      data.header
    );
    check(nokResp, {
      [`POST ${triggerPath(pipelineId)} (NOK) response status is 200`]: (r) =>
        r.status === 200,
      [`POST ${triggerPath(pipelineId)} (NOK) returns error status`]: (r) =>
        r.json().metadata.traces.jq.statuses[0] === "STATUS_ERROR",
    });

    // Successfully trigger pipeline
    const okPayload = JSON.stringify({
      data: [
        {
          variable: {
            jota: { foo: "bar" },
          },
        },
      ],
    });

    const okResp = http.request(
      "POST",
      pipelinePublicHost + triggerPath(pipelineId),
      okPayload,
      data.header
    );
    check(okResp, {
      [`POST ${triggerPath(pipelineId)} (OK) response status is 200`]: (r) =>
        r.status === 200,
      [`POST ${triggerPath(pipelineId)} (OK) contains result`]: (r) =>
        r.json().outputs[0].out === "bar",
      [`POST ${triggerPath(pipelineId)} (OK) returns successful status`]: (r) =>
        r.json().metadata.traces.jq.statuses[0] === "STATUS_COMPLETED",
    });

    const pipelineRuns = http.request(
      "GET",
      pipelinePublicHost + resourcePath(pipelineId) + "/runs",
      null,
      data.header
    );
    check(pipelineRuns, {
      [`GET ${resourcePath(pipelineId)}/runs response status is 200`]: (r) =>
        r.status === 200,
      [`GET ${resourcePath(pipelineId)}/runs contains runs`]: (r) =>
        r.json().pipelineRuns.length === 2,
      [`GET ${resourcePath(pipelineId)}/runs has a successful run`]: (r) =>
        r.json().pipelineRuns[0].status === "RUN_STATUS_COMPLETED",
      [`GET ${resourcePath(pipelineId)}/runs has a failed run`]: (r) =>
        r.json().pipelineRuns[1].status === "RUN_STATUS_FAILED",
    });

    // Use the run id to construct the component-runs endpoint URL
    const okPipelineRunId = pipelineRuns.json().pipelineRuns[0].id;
    const okComponentRunsPath = `${resourcePath(pipelineId)}/runs/${okPipelineRunId}/component-runs`;

    const okComponentRuns = http.request(
      "GET",
      pipelinePublicHost + okComponentRunsPath,
      null,
      data.header
    );
    check(okComponentRuns, {
      [`GET ${okComponentRunsPath} response status is 200`]: (r) =>
        r.status === 200,
      [`GET ${okComponentRunsPath} contains component runs`]: (r) =>
        r.json().componentRuns && r.json().componentRuns.length === 1,
      [`GET ${okComponentRunsPath} matches the component ID`]: (r) =>
        r.json().componentRuns[0].id === "jq",
      [`GET ${okComponentRunsPath} has a successful component run`]: (r) =>
        r.json().componentRuns[0].status === "RUN_STATUS_COMPLETED",
    });

    // Use the run id to construct the component-runs endpoint URL
    const nokPipelineRunId = pipelineRuns.json().pipelineRuns[1].id;
    const nokComponentRunsPath = `${resourcePath(pipelineId)}/runs/${nokPipelineRunId}/component-runs`;
    const nokComponentRuns = http.request(
      "GET",
      pipelinePublicHost + nokComponentRunsPath,
      null,
      data.header
    );
    check(nokComponentRuns, {
      [`GET ${nokComponentRunsPath} (NOK) response status is 200`]: (r) =>
        r.status === 200,
      [`GET ${nokComponentRunsPath} (NOK) contains component runs`]: (r) =>
        r.json().componentRuns && r.json().componentRuns.length === 1,
      [`GET ${nokComponentRunsPath} (NOK) matches the component ID`]: (r) =>
        r.json().componentRuns[0].id === "jq",
      [`GET ${nokComponentRunsPath} (NOK) has a failed component run`]: (r) =>
        r.json().componentRuns[0].status === "RUN_STATUS_FAILED",
    });

    check(
      http.request(
        "DELETE",
        pipelinePublicHost + resourcePath(pipelineId),
        null,
        data.header
      ),
      {
        [`DELETE ${resourcePath(pipelineId)} response status 204`]: (r) =>
          r.status === 204,
      }
    );
  });
}

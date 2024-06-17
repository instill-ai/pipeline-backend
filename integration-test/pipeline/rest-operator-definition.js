import http from "k6/http";
import { check, group } from "k6";

import { pipelinePublicHost } from "./const.js";
import { deepEqual } from "./helper.js"

export function CheckList() {
  group("Component API: List operator definitions", () => {
    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions`, null, null), {
      "GET /v1beta/operator-definitions response status is 200": (r) => r.status === 200,
      "GET /v1beta/operator-definitions response has operatorDefinitions array": (r) => Array.isArray(r.json().operatorDefinitions),
      "GET /v1beta/operator-definitions response totalSize > 0": (r) => r.json().totalSize > 0
    });

    var limitedRecords = http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions`, null, null)
    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?pageSize=0`, null, null), {
      "GET /v1beta/operator-definitions?pageSize=0 response status is 200": (r) => r.status === 200,
      "GET /v1beta/operator-definitions?pageSize=0 response limited records for 10": (r) => r.json().operatorDefinitions.length === limitedRecords.json().operatorDefinitions.length,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?pageSize=1`, null, null), {
      "GET /v1beta/operator-definitions?pageSize=1 response status is 200": (r) => r.status === 200,
      "GET /v1beta/operator-definitions?pageSize=1 response operatorDefinitions size 1": (r) => r.json().operatorDefinitions.length === 1,
    });

    var pageRes = http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?pageSize=1`, null, null)
    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?pageSize=1&pageToken=${pageRes.json().nextPageToken}`, null, null), {
      [`GET /v1beta/operator-definitions?pageSize=1&pageToken=${pageRes.json().nextPageToken} response status is 200`]: (r) => r.status === 200,
      [`GET /v1beta/operator-definitions?pageSize=1&pageToken=${pageRes.json().nextPageToken} response operatorDefinitions size 1`]: (r) => r.json().operatorDefinitions.length === 1,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?pageSize=1&view=VIEW_BASIC`, null, null), {
      "GET /v1beta/operator-definitions?pageSize=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
      "GET /v1beta/operator-definitions?pageSize=1&view=VIEW_BASIC response operatorDefinitions[0].spec is null": (r) => r.json().operatorDefinitions[0].spec === null,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?pageSize=1&view=VIEW_FULL`, null, null), {
      "GET /v1beta/operator-definitions?pageSize=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
      "GET /v1beta/operator-definitions?pageSize=1&view=VIEW_FULL response operatorDefinitions[0].spec is not null": (r) => r.json().operatorDefinitions[0].spec !== null,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?pageSize=1`, null, null), {
      "GET /v1beta/operator-definitions?pageSize=1 response status 200": (r) => r.status === 200,
      "GET /v1beta/operator-definitions?pageSize=1 response operatorDefinitions[0].spec is null": (r) => r.json().operatorDefinitions[0].spec === null,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?pageSize=${limitedRecords.json().totalSize}`, null, null), {
      [`GET /v1beta/operator-definitions?pageSize=${limitedRecords.json().totalSize} response status 200`]: (r) => r.status === 200,
      [`GET /v1beta/operator-definitions?pageSize=${limitedRecords.json().totalSize} response nextPageToken is empty`]: (r) => r.json().nextPageToken === "",
    });
  });
}

export function CheckGet() {
  group("Operator API: Get destination operator definition", () => {
    var allRes = http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions`, null, null)
    var def = allRes.json().operatorDefinitions[0]
    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions/${def.id}`, null, null), {
      [`GET /v1beta/operator-definitions/${def.id} response status is 200`]: (r) => r.status === 200,
      [`GET /v1beta/operator-definitions/${def.id} response has the exact record`]: (r) => deepEqual(r.json().operatorDefinition, def),
      [`GET /v1beta/operator-definitions/${def.id} response has the non-empty resource name ${def.name}`]: (r) => r.json().operatorDefinition.name != "",
      [`GET /v1beta/operator-definitions/${def.id} response has the resource name ${def.name}`]: (r) => r.json().operatorDefinition.name === def.name,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions/${def.id}?view=VIEW_BASIC`, null, null), {
      [`GET /v1beta/operator-definitions/${def.id}?view=VIEW_BASIC response status 200`]: (r) => r.status === 200,
      [`GET /v1beta/operator-definitions/${def.id}?view=VIEW_BASIC response operatorDefinition.spec is null`]: (r) => r.json().operatorDefinition.spec === null,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions/${def.id}?view=VIEW_FULL`, null, null), {
      [`GET /v1beta/operator-definitions/${def.id}?view=VIEW_FULL response status 200`]: (r) => r.status === 200,
      [`GET /v1beta/operator-definitions/${def.id}?view=VIEW_FULL response operatorDefinition.spec is not null`]: (r) => r.json().operatorDefinition.spec !== null,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions/${def.id}`, null, null), {
      [`GET /v1beta/operator-definitions/${def.id} response status 200`]: (r) => r.status === 200,
      [`GET /v1beta/operator-definitions/${def.id} response operatorDefinition.spec is null`]: (r) => r.json().operatorDefinition.spec === null,
    });
  });
}

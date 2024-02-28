import http from "k6/http";
import { check, group } from "k6";

import { pipelinePublicHost } from "./const.js";
import { deepEqual } from "./helper.js"

export function CheckList() {
  group("Component API: List operator definitions", () => {
    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions`, null, null), {
      "GET /v1beta/operator-definitions response status is 200": (r) => r.status === 200,
      "GET /v1beta/operator-definitions response has operator_definitions array": (r) => Array.isArray(r.json().operator_definitions),
      "GET /v1beta/operator-definitions response total_size > 0": (r) => r.json().total_size > 0
    });

    var limitedRecords = http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions`, null, null)
    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?page_size=0`, null, null), {
      "GET /v1beta/operator-definitions?page_size=0 response status is 200": (r) => r.status === 200,
      "GET /v1beta/operator-definitions?page_size=0 response limited records for 10": (r) => r.json().operator_definitions.length === limitedRecords.json().operator_definitions.length,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?page_size=1`, null, null), {
      "GET /v1beta/operator-definitions?page_size=1 response status is 200": (r) => r.status === 200,
      "GET /v1beta/operator-definitions?page_size=1 response operator_definitions size 1": (r) => r.json().operator_definitions.length === 1,
    });

    var pageRes = http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?page_size=1`, null, null)
    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?page_size=1&page_token=${pageRes.json().next_page_token}`, null, null), {
      [`GET /v1beta/operator-definitions?page_size=1&page_token=${pageRes.json().next_page_token} response status is 200`]: (r) => r.status === 200,
      [`GET /v1beta/operator-definitions?page_size=1&page_token=${pageRes.json().next_page_token} response operator_definitions size 1`]: (r) => r.json().operator_definitions.length === 1,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?page_size=1&view=VIEW_BASIC`, null, null), {
      "GET /v1beta/operator-definitions?page_size=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
      "GET /v1beta/operator-definitions?page_size=1&view=VIEW_BASIC response operator_definitions[0].spec is null": (r) => r.json().operator_definitions[0].spec === null,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?page_size=1&view=VIEW_FULL`, null, null), {
      "GET /v1beta/operator-definitions?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
      "GET /v1beta/operator-definitions?page_size=1&view=VIEW_FULL response operator_definitions[0].spec is not null": (r) => r.json().operator_definitions[0].spec !== null,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?page_size=1`, null, null), {
      "GET /v1beta/operator-definitions?page_size=1 response status 200": (r) => r.status === 200,
      "GET /v1beta/operator-definitions?page_size=1 response operator_definitions[0].spec is null": (r) => r.json().operator_definitions[0].spec === null,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions?page_size=${limitedRecords.json().total_size}`, null, null), {
      [`GET /v1beta/operator-definitions?page_size=${limitedRecords.json().total_size} response status 200`]: (r) => r.status === 200,
      [`GET /v1beta/operator-definitions?page_size=${limitedRecords.json().total_size} response next_page_token is empty`]: (r) => r.json().next_page_token === "",
    });
  });
}

export function CheckGet() {
  group("Operator API: Get destination operator definition", () => {
    var allRes = http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions`, null, null)
    var def = allRes.json().operator_definitions[0]
    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions/${def.id}`, null, null), {
      [`GET /v1beta/operator-definitions/${def.id} response status is 200`]: (r) => r.status === 200,
      [`GET /v1beta/operator-definitions/${def.id} response has the exact record`]: (r) => deepEqual(r.json().operator_definition, def),
      [`GET /v1beta/operator-definitions/${def.id} response has the non-empty resource name ${def.name}`]: (r) => r.json().operator_definition.name != "",
      [`GET /v1beta/operator-definitions/${def.id} response has the resource name ${def.name}`]: (r) => r.json().operator_definition.name === def.name,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions/${def.id}?view=VIEW_BASIC`, null, null), {
      [`GET /v1beta/operator-definitions/${def.id}?view=VIEW_BASIC response status 200`]: (r) => r.status === 200,
      [`GET /v1beta/operator-definitions/${def.id}?view=VIEW_BASIC response operator_definition.spec is null`]: (r) => r.json().operator_definition.spec === null,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions/${def.id}?view=VIEW_FULL`, null, null), {
      [`GET /v1beta/operator-definitions/${def.id}?view=VIEW_FULL response status 200`]: (r) => r.status === 200,
      [`GET /v1beta/operator-definitions/${def.id}?view=VIEW_FULL response operator_definition.spec is not null`]: (r) => r.json().operator_definition.spec !== null,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/operator-definitions/${def.id}`, null, null), {
      [`GET /v1beta/operator-definitions/${def.id} response status 200`]: (r) => r.status === 200,
      [`GET /v1beta/operator-definitions/${def.id} response operator_definition.spec is null`]: (r) => r.json().operator_definition.spec === null,
    });
  });
}


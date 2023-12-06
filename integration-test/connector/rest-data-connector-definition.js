import http from "k6/http";
import { check, group } from "k6";

import { pipelinePublicHost } from "./const.js"
import { deepEqual } from "./helper.js"

export function CheckList(header) {

    group("Connector API: List destination connector definitions", () => {

        check(http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions`, null, header), {
            "GET /v1beta/connector-definitions response status is 200": (r) => r.status === 200,
            "GET /v1beta/connector-definitions response has connector_definitions array": (r) => Array.isArray(r.json().connector_definitions),
            "GET /v1beta/connector-definitions response total_size > 0": (r) => r.json().total_size > 0
        });

        var limitedRecords = http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions`, null, header)
        check(http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions?page_size=0`, null, header), {
            "GET /v1beta/connector-definitions?page_size=0 response status is 200": (r) => r.status === 200,
            "GET /v1beta/connector-definitions?page_size=0 response limited records for 10": (r) => r.json().connector_definitions.length === limitedRecords.json().connector_definitions.length,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions?page_size=1`, null, header), {
            "GET /v1beta/connector-definitions?page_size=1 response status is 200": (r) => r.status === 200,
            "GET /v1beta/connector-definitions?page_size=1 response connector_definitions size 1": (r) => r.json().connector_definitions.length === 1,
        });

        var pageRes = http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions?page_size=1`, null, header)
        check(http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions?page_size=1&page_token=${pageRes.json().next_page_token}`, null, header), {
            [`GET /v1beta/connector-definitions?page_size=1&page_token=${pageRes.json().next_page_token} response status is 200`]: (r) => r.status === 200,
            [`GET /v1beta/connector-definitions?page_size=1&page_token=${pageRes.json().next_page_token} response connector_definitions size 1`]: (r) => r.json().connector_definitions.length === 1,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions?page_size=1&view=VIEW_BASIC`, null, header), {
            "GET /v1beta/connector-definitions?page_size=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
            "GET /v1beta/connector-definitions?page_size=1&view=VIEW_BASIC response connector_definitions[0].spec is null": (r) => r.json().connector_definitions[0].spec === null,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions?page_size=1&view=VIEW_FULL`, null, header), {
            "GET /v1beta/connector-definitions?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /v1beta/connector-definitions?page_size=1&view=VIEW_FULL response connector_definitions[0].spec is not null": (r) => r.json().connector_definitions[0].spec !== null,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions?page_size=1`, null, header), {
            "GET /v1beta/connector-definitions?page_size=1 response status 200": (r) => r.status === 200,
            "GET /v1beta/connector-definitions?page_size=1 response connector_definitions[0].spec is null": (r) => r.json().connector_definitions[0].spec === null,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions?page_size=${limitedRecords.json().total_size}`, null, header), {
            [`GET /v1beta/connector-definitions?page_size=${limitedRecords.json().total_size} response status 200`]: (r) => r.status === 200,
            [`GET /v1beta/connector-definitions?page_size=${limitedRecords.json().total_size} response next_page_token is empty`]: (r) => r.json().next_page_token === "",
        });
    });
}

export function CheckGet(header) {
    group("Connector API: Get destination connector definition", () => {
        var allRes = http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions`, null, header)
        var def = allRes.json().connector_definitions[0]
        check(http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions/${def.id}`, null, header), {
            [`GET /v1beta/connector-definitions/${def.id} response status is 200`]: (r) => r.status === 200,
            [`GET /v1beta/connector-definitions/${def.id} response has the exact record`]: (r) => deepEqual(r.json().connector_definition, def),
            [`GET /v1beta/connector-definitions/${def.id} response has the non-empty resource name ${def.name}`]: (r) => r.json().connector_definition.name != "",
            [`GET /v1beta/connector-definitions/${def.id} response has the resource name ${def.name}`]: (r) => r.json().connector_definition.name === def.name,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions/${def.id}?view=VIEW_BASIC`, null, header), {
            [`GET /v1beta/connector-definitions/${def.id}?view=VIEW_BASIC response status 200`]: (r) => r.status === 200,
            [`GET /v1beta/connector-definitions/${def.id}?view=VIEW_BASIC response connector_definition.spec is null`]: (r) => r.json().connector_definition.spec === null,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions/${def.id}?view=VIEW_FULL`, null, header), {
            [`GET /v1beta/connector-definitions/${def.id}?view=VIEW_FULL response status 200`]: (r) => r.status === 200,
            [`GET /v1beta/connector-definitions/${def.id}?view=VIEW_FULL response connector_definition.spec is not null`]: (r) => r.json().connector_definition.spec !== null,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions/${def.id}`, null, header), {
            [`GET /v1beta/connector-definitions/${def.id} response status 200`]: (r) => r.status === 200,
            [`GET /v1beta/connector-definitions/${def.id} response connector_definition.spec is null`]: (r) => r.json().connector_definition.spec === null,
        });
    });

}

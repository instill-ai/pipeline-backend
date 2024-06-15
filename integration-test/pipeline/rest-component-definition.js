import http from "k6/http";
import { check, group } from "k6";

import { pipelinePublicHost } from "./const.js";

export function CheckList() {
  group("Component API: List component definitions", () => {
    // Default pagination.
    var defaultPageSize = 10;
    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions`, null, null), {
      "GET /v1beta/component-definitions response status is 200": (r) => r.status === 200,
      "GET /v1beta/component-definitions response has componentDefinitions array": (r) => Array.isArray(r.json().componentDefinitions),
      "GET /v1beta/component-definitions response totalSize > 0": (r) => r.json().totalSize > 0,
      "GET /v1beta/component-definitions response page 0": (r) => r.json().page === 0,
      [`GET /v1beta/component-definitions response default page size ${defaultPageSize}`]: (r) => r.json().componentDefinitions.length === defaultPageSize,
      [`GET /v1beta/component-definitions response page size in response ${defaultPageSize}`]: (r) => r.json().pageSize === defaultPageSize,
      "GET /v1beta/component-definitions response features Instill Model on top": (r) => r.json().componentDefinitions[0].id === "instill-model",
    });

    var limitedRecords = http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions`, null, null)

    // Page size 0.
    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions?page_size=0`, null, null), {
      "GET /v1beta/component-definitions?page_size=0 response status is 200": (r) => r.status === 200,
      "GET /v1beta/component-definitions?page_size=0 response default page size": (r) => r.json().componentDefinitions.length === limitedRecords.json().componentDefinitions.length,
    });

    // Negative page size.
    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions?page_size=-1`, null, null), {
      "GET /v1beta/component-definitions?page_size=-1 response status is 200": (r) => r.status === 200,
      "GET /v1beta/component-definitions?page_size=-1 response default page size": (r) => r.json().componentDefinitions.length === limitedRecords.json().componentDefinitions.length,
    });

    // Valid, non-default page size.
    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions?page_size=1`, null, null), {
      "GET /v1beta/component-definitions?page_size=1 response status is 200": (r) => r.status === 200,
      "GET /v1beta/component-definitions?page_size=1 response componentDefinitions size 1": (r) => r.json().componentDefinitions.length === 1,
    });

    // Page size over total records.
    var bigPage = limitedRecords.json().totalSize + 10
    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions?page_size=${bigPage}`, null, null), {
      [`GET /v1beta/component-definitions?page_size=${bigPage} response status 200`]: (r) => r.status === 200,
      [`GET /v1beta/component-definitions?page_size=${bigPage} response componentDefinitions size ${limitedRecords.json().totalSize }`]: (r) => r.json().componentDefinitions.length === limitedRecords.json().totalSize,
    });

    // Access non-first page.
    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions?page_size=2&page=2`, null, null), {
      "GET /v1beta/component-definitions?page_size=2&page=2 response status is 200": (r) => r.status === 200,
      "GET /v1beta/component-definitions?page_size=2&page=2 response componentDefinitions size 3": (r) => r.json().componentDefinitions.length === 2,
      "GET /v1beta/component-definitions?page_size=2&page=2 response page 0": (r) => r.json().page === 2,
      "GET /v1beta/component-definitions?page_size=2&page=2 receives a different page": (r) => r.json().componentDefinitions[0].id != limitedRecords.json().componentDefinitions[0].id,
    });

    // Negative page index yields page 0.
    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions?page_size=2&page=-2`, null, null), {
      "GET /v1beta/component-definitions?page_size=2&page=-2 response status is 200": (r) => r.status === 200,
      "GET /v1beta/component-definitions?page_size=2&page=-2 response componentDefinitions size 3": (r) => r.json().componentDefinitions.length === 2,
      "GET /v1beta/component-definitions?page_size=2&page=-2 response page 0": (r) => r.json().page === 0,
    });

    // Page index beyond last page.
    var bigPage = limitedRecords.json().totalSize + 10
    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions?page_size=${bigPage}&page=2`, null, null), {
      [`GET /v1beta/component-definitions?page_size=${bigPage}&page=2 response status 200`]: (r) => r.status === 200,
      [`GET /v1beta/component-definitions?page_size=${bigPage}&page=2 response componentDefinitions size 0`]: (r) => r.json().componentDefinitions.length === 0,
    });

    // Default view is BASIC, i.e. no spec property.
    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions?page_size=1`, null, null), {
      "GET /v1beta/component-definitions?page_size=1 response status 200": (r) => r.status === 200,
      "GET /v1beta/component-definitions?page_size=1 response componentDefinitions[0].spec is null": (r) => r.json().componentDefinitions[0].spec === null,
    });

    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions?page_size=1&view=VIEW_BASIC`, null, null), {
      "GET /v1beta/component-definitions?page_size=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
      "GET /v1beta/component-definitions?page_size=1&view=VIEW_BASIC response componentDefinitions[0].spec is null": (r) => r.json().componentDefinitions[0].spec === null,
    });

    // FULL view.
    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions?page_size=1&view=VIEW_FULL`, null, null), {
      "GET /v1beta/component-definitions?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
      "GET /v1beta/component-definitions?page_size=1&view=VIEW_FULL response componentDefinitions[0].spec is not null": (r) => r.json().componentDefinitions[0].spec !== null,
    });

    // Filter (fuzzy) title
    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions?page_size=1&filter=q_title="JSO"`, null, null), {
      [`GET /v1beta/component-definitions?page_size=1&filter=q_title="JSO" response status 200`]: (r) => r.status === 200,
      [`GET /v1beta/component-definitions?page_size=1&filter=q_title="JSO" single result`]: (r) => r.json().totalSize === 1,
      [`GET /v1beta/component-definitions?page_size=1&filter=q_title="JSO" title is JSON`]: (r) => r.json().componentDefinitions[0].title === "JSON",
    });

    // Filter component type
    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions?page_size=1&filter=component_type=COMPONENT_TYPE_OPERATOR`, null, null), {
      "GET /v1beta/component-definitions?page_size=1&filter=component_type=COMPONENT_TYPE_OPERATOR response status 200": (r) => r.status === 200,
      "GET /v1beta/component-definitions?page_size=1&filter=component_type=COMPONENT_TYPE_OPERATOR total size is smaller": (r) => r.json().totalSize < limitedRecords.json().totalSize,
      "GET /v1beta/component-definitions?page_size=1&filter=component_type=COMPONENT_TYPE_OPERATOR type is COMPONENT_TYPE_OPERATOR": (r) => r.json().componentDefinitions[0].type === "COMPONENT_TYPE_OPERATOR",
    });

    // Filter release stage
    check(http.request("GET", `${pipelinePublicHost}/v1beta/component-definitions?page_size=1&filter=release_stage=RELEASE_STAGE_ALPHA`, null, null), {
      [`GET /v1beta/component-definitions?page_size=1&filter=release_stage=RELEASE_STAGE_ALPHA response status 200`]: (r) => r.status === 200,
      // TODO when there are non-alpha components, update expectations.
      [`GET /v1beta/component-definitions?page_size=1&filter=release_stage=RELEASE_STAGE_ALPHA number of results`]: (r) => r.json().totalSize === limitedRecords.json().totalSize,
      [`GET /v1beta/component-definitions?page_size=1&filter=release_stage=RELEASE_STAGE_ALPHA release_stage is alpha`]: (r) => r.json().componentDefinitions[0].releaseStage === "RELEASE_STAGE_ALPHA",
    });
  });
}

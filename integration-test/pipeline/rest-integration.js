import http from "k6/http";
import { check, group } from "k6";

import { pipelinePublicHost } from "./const.js";
import { deepEqual } from "./helper.js";

export function CheckIntegrations() {
  group("Integration API: Get integration", () => {
    // Inexistent component
    check(http.request("GET", `${pipelinePublicHost}/v1beta/integrations/restapio`, null, null), {
      "GET /v1beta/integrations/restapio response status is 404": (r) => r.status === 404,
      "GET /v1beta/integrations/restapio response contains end-user message": (r) => r.json().message === "Integration does not exist.",
    });

    // Component without setup
    check(http.request("GET", `${pipelinePublicHost}/v1beta/integrations/document`, null, null), {
      "GET /v1beta/integrations/document response status is 404": (r) => r.status === 404,
      "GET /v1beta/integrations/document response contains end-user message": (r) => r.json().message === "Integration does not exist.",
    });

    var id = "pinecone";
    var cdef = http.request("GET", `${pipelinePublicHost}/v1beta/connector-definitions/${id}`, null, null).
      json().connectorDefinition;

    var integration = {
      uid: cdef.uid,
      id: cdef.id,
      title: cdef.title,
      description: cdef.description,
      vendor: cdef.vendor,
      icon: cdef.icon,
      featured: true,
      schemas: [],
      view: "VIEW_BASIC"
    };

    // Basic view
    check(http.request("GET", `${pipelinePublicHost}/v1beta/integrations/${id}`, null, null), {
      [`GET /v1beta/integrations/${id} response status is 200`]: (r) => r.status === 200,
      [`GET /v1beta/integrations/${id} response contains expected integration`]: (r) => deepEqual(r.json().integration, integration),
    });

    // Full view
    check(http.request("GET", `${pipelinePublicHost}/v1beta/integrations/${id}?view=VIEW_FULL`, null, null), {
      [`GET /v1beta/integrations/${id}?view=VIEW_FULL response status is 200`]: (r) => r.status === 200,
      [`GET /v1beta/integrations/${id}?view=VIEW_FULL response contains schema`]: (r) => r.json().integration.schemas[0].method === "METHOD_DICTIONARY",
    });

    // Unfeatured integration
    var unfeaturedID = "cohere";
    check(http.request("GET", `${pipelinePublicHost}/v1beta/integrations/${unfeaturedID}`, null, null), {
      [`GET /v1beta/integrations/${unfeaturedID} response status is 200`]: (r) => r.status === 200,
      [`GET /v1beta/integrations/${unfeaturedID} response has featured: false`]: (r) => r.json().integration.featured === false,
    });
  });

  group("Integration API: List integrations", () => {
    // Default pagination.
    var defaultPageSize = 10;
    var firstPage = http.request( "GET", `${pipelinePublicHost}/v1beta/integrations`, null, null);
    check(firstPage, {
      "GET /v1beta/integrations response status is 200": (r) => r.status === 200,
      "GET /v1beta/integrations response totalSize > 0": (r) => r.json().totalSize > 0,
      "GET /v1beta/integrations starts with featured integrations": (r) => r.json().integrations[0].featured === true,
      "GET /v1beta/integrations has default page size": (r) => r.json().integrations.length === defaultPageSize,
    });

    // Non-default pagination, non-first page
    var tokenPageTwo = firstPage.json().nextPageToken;
    check(http.request("GET", `${pipelinePublicHost}/v1beta/integrations?pageSize=2&pageToken=${tokenPageTwo}`, null, null), {
      [`GET /v1beta/integrations?pageSize=2&pageToken=${tokenPageTwo} response status is 200`]: (r) => r.status === 200,
      [`GET /v1beta/integrations?pageSize=2&pageToken=${tokenPageTwo} has page size 2"`]: (r) => r.json().integrations.length === 2,
      [`GET /v1beta/integrations?pageSize=2&pageToken=${tokenPageTwo} has different elements than page 1"`]: (r) => r.json().integrations[0].id != firstPage.json().integrations[0].id,
    });

    // Filter featured
    check(http.request( "GET", `${pipelinePublicHost}/v1beta/integrations?filter=NOT%20featured`, null, null), {
      "GET /v1beta/integrations?filter=NOT%20featured response status is 200": (r) => r.status === 200,
      "GET /v1beta/integrations?filter=NOT%20featured response totalSize > 0": (r) => r.json().totalSize > 0,
      "GET /v1beta/integrations?filter=NOT%20featured response totalSize < firstPage.totalSize": (r) => r.json().totalSize < firstPage.json().totalSize,
      "GET /v1beta/integrations?filter=NOT%20featured doesn't have featured integrations": (r) => r.json().integrations[0].featured === false,
    });

    // Filter fuzzy title
    check(http.request( "GET", `${pipelinePublicHost}/v1beta/integrations?filter=qIntegration="que"`, null, null), {
      [`GET /v1beta/integrations?filter=qIntegration="que" response status is 200`]: (r) => r.status === 200,
      [`GET /v1beta/integrations?filter=qIntegration="que" response totalSize > 0`]: (r) => r.json().totalSize === 1,
      [`GET /v1beta/integrations?filter=qIntegration="que" returns BigQuery integration`]: (r) => r.json().integrations[0].title === "BigQuery",
    });

    // Filter fuzzy vendor
    check(http.request( "GET", `${pipelinePublicHost}/v1beta/integrations?filter=qIntegration="labs"`, null, null), {
      [`GET /v1beta/integrations?filter=qIntegration="labs" response status is 200`]: (r) => r.status === 200,
      [`GET /v1beta/integrations?filter=qIntegration="labs" response totalSize > 0`]: (r) => r.json().totalSize === 1,
      [`GET /v1beta/integrations?filter=qIntegration="labs" returns Redis integration`]: (r) => r.json().integrations[0].title === "Redis",
      [`GET /v1beta/integrations?filter=qIntegration="labs" intgration vendor matches Redis Labs`]: (r) => r.json().integrations[0].vendor === "Redis Labs",
    });
  });
}

import http from "k6/http";
import { check, group } from "k6";

import { pipelinePublicHost } from "./const.js";
import { deepEqual } from "./helper.js";

export function CheckGet() {
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
}

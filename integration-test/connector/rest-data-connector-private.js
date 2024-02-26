import http from "k6/http";
import {
    check,
    group,
    sleep
} from "k6";
import {
    randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import {
    pipelinePublicHost,
    pipelinePrivateHost
} from "./const.js"

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckList(data) {

    group("Connector API: List destination connectors by admin", () => {

        check(http.request("GET", `${pipelinePrivateHost}/v1beta/admin/connectors?filter=connector_type=CONNECTOR_TYPE_DATA`), {
            [`GET /v1beta/admin/connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1beta/admin/connectors response connectors array is 0 length`]: (r) => r.json().connectors.length === 0,
            [`GET /v1beta/admin/connectors response next_page_token is empty`]: (r) => r.json().next_page_token === "",
            [`GET /v1beta/admin/connectors response total_size is 0`]: (r) => r.json().total_size == 0,
        });

        const numConnectors = 10
        var reqBodies = [];
        for (var i = 0; i < numConnectors; i++) {
            reqBodies[i] = {
                "id": randomString(10),
                "connector_definition_name": constant.csvDstDefRscName,
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        // Create connectors
        for (const reqBody of reqBodies) {
            var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
                JSON.stringify(reqBody), data.header)
            check(resCSVDst, {
                [`POST /v1beta/${constant.namespace}/connectors x${reqBodies.length} response status 201`]: (r) => r.status === 201,
            });
        }

        check(http.request("GET", `${pipelinePrivateHost}/v1beta/admin/connectors?filter=connector_type=CONNECTOR_TYPE_DATA`), {
            [`GET /v1beta/admin/connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1beta/admin/connectors response has connectors array`]: (r) => Array.isArray(r.json().connectors),
            [`GET /v1beta/admin/connectors response has total_size = ${numConnectors}`]: (r) => r.json().total_size == numConnectors,
        });

        var limitedRecords = http.request("GET", `${pipelinePrivateHost}/v1beta/admin/connectors?filter=connector_type=CONNECTOR_TYPE_DATA`)
        check(http.request("GET", `${pipelinePrivateHost}/v1beta/admin/connectors?page_size=0`, null, data.header), {
            "GET /v1beta/admin/connectors?page_size=0 response status is 200": (r) => r.status === 200,
            "GET /v1beta/admin/connectors?page_size=0 response all records": (r) => r.json().connectors.length === limitedRecords.json().connectors.length,
        });

        check(http.request("GET", `${pipelinePrivateHost}/v1beta/admin/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1`), {
            "GET /v1beta/admin/connectors?page_size=1 response status is 200": (r) => r.status === 200,
            "GET /v1beta/admin/connectors?page_size=1 response connectors size 1": (r) => r.json().connectors.length === 1,
        });

        var pageRes = http.request("GET", `${pipelinePrivateHost}/v1beta/admin/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1`)
        check(http.request("GET", `${pipelinePrivateHost}/v1beta/admin/connectors?page_size=1&page_token=${pageRes.json().next_page_token}`, null, data.header), {
            [`GET /v1beta/admin/connectors?page_size=1&page_token=${pageRes.json().next_page_token} response status is 200`]: (r) => r.status === 200,
            [`GET /v1beta/admin/connectors?page_size=1&page_token=${pageRes.json().next_page_token} response connectors size 1`]: (r) => r.json().connectors.length === 1,
        });

        check(http.request("GET", `${pipelinePrivateHost}/v1beta/admin/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1&view=VIEW_BASIC`), {
            "GET /v1beta/admin/connectors?page_size=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
            "GET /v1beta/admin/connectors?page_size=1&view=VIEW_BASIC response connectors[0].configuration is null": (r) => r.json().connectors[0].configuration === null,
            "GET /v1beta/admin/connectors?page_size=1&view=VIEW_BASIC response connectors[0].owner is invalid": (r) => r.json().connectors[0].owner === undefined,
        });

        check(http.request("GET", `${pipelinePrivateHost}/v1beta/admin/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1&view=VIEW_FULL`), {
            "GET /v1beta/admin/connectors?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /v1beta/admin/connectors?page_size=1&view=VIEW_FULL response connectors[0].configuration is not null": (r) => r.json().connectors[0].configuration !== null,
            "GET /v1beta/admin/connectors?page_size=1&view=VIEW_FULL response connectors[0].connector_definition_detail is not null": (r) => r.json().connectors[0].connector_definition_detail !== null,
            "GET /v1beta/admin/connectors?page_size=1&view=VIEW_FULL response connectors[0].owner is valid": (r) => helper.isValidOwner(r.json().connectors[0].owner, data.expectedOwner),
        });

        check(http.request("GET", `${pipelinePrivateHost}/v1beta/admin/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1`), {
            "GET /v1beta/admin/connectors?page_size=1 response status 200": (r) => r.status === 200,
            "GET /v1beta/admin/connectors?page_size=1 response connectors[0].configuration is null": (r) => r.json().connectors[0].configuration === null,
            "GET /v1beta/admin/connectors?page_size=1 response connectors[0].owner is invalid": (r) => r.json().connectors[0].owner === undefined,
        });

        check(http.request("GET", `${pipelinePrivateHost}/v1beta/admin/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=${limitedRecords.json().total_size}`), {
            [`GET /v1beta/admin/connectors?page_size=${limitedRecords.json().total_size} response status 200`]: (r) => r.status === 200,
            [`GET /v1beta/admin/connectors?page_size=${limitedRecords.json().total_size} response next_page_token is empty`]: (r) => r.json().next_page_token === ""
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${reqBody.id}`, null, data.header), {
                [`DELETE /v1beta/admin/connectors x${reqBodies.length} response status is 204`]: (r) => r.status === 204,
            });
        }
    });
}

export function CheckLookUp(data) {

    group("Connector API: Look up destination connectors by UID by admin", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), data.header)

        check(http.request("GET", `${pipelinePrivateHost}/v1beta/admin/connectors/${resCSVDst.json().connector.uid}/lookUp`), {
            [`GET /v1beta/admin/connectors/${resCSVDst.json().connector.uid}/lookUp response status 200`]: (r) => r.status === 200,
            [`GET /v1beta/admin/connectors/${resCSVDst.json().connector.uid}/lookUp response connector uid`]: (r) => r.json().connector.uid === resCSVDst.json().connector.uid,
            [`GET /v1beta/admin/connectors/${resCSVDst.json().connector.uid}/lookUp response connector connector_definition_name`]: (r) => r.json().connector.connector_definition_name === constant.csvDstDefRscName,
            [`GET /v1beta/admin/connectors/${resCSVDst.json().connector.uid}/lookUp response connector owner is invalid`]: (r) => r.json().connector.owner === undefined,
        });

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, data.header), {
            [`DELETE /v1beta/admin/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

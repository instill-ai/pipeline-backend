import http from "k6/http";
import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js"

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate(data) {

    group("Connector API: Create end connector", () => {

        // destination-csv
        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), data.header)

        check(resCSVDst, {
            "POST /v1beta/${constant.namespace}/connectors response status 201": (r) => r.status === 201,
        });
        http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${csvDstConnector.id}/connect`,
            {}, data.header)

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch`, null, data.header), {
            [`GET /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, data.header), {
            [`GET /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response STATE_CONNECTED`]: (r) => r.json().connector.state === "STATE_CONNECTED",
        });

        // Delete test records
        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, data.header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckList(data) {

    group("Connector API: List destination connectors", () => {

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors?filter=connector_type=CONNECTOR_TYPE_DATA`, null, data.header), {
            [`GET /v1beta/${constant.namespace}/connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1beta/${constant.namespace}/connectors response connectors array is 0 length`]: (r) => r.json().connectors.length === 0,
            [`GET /v1beta/${constant.namespace}/connectors response next_page_token is empty`]: (r) => r.json().next_page_token === "",
            [`GET /v1beta/${constant.namespace}/connectors response total_size is 0`]: (r) => r.json().total_size == 0,
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

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors?filter=connector_type=CONNECTOR_TYPE_DATA`, null, data.header), {
            [`GET /v1beta/${constant.namespace}/connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1beta/${constant.namespace}/connectors response has connectors array`]: (r) => Array.isArray(r.json().connectors),
            [`GET /v1beta/${constant.namespace}/connectors response has total_size = ${numConnectors}`]: (r) => r.json().total_size == numConnectors,
        });

        var limitedRecords = http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors?filter=connector_type=CONNECTOR_TYPE_DATA`, null, data.header)
        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=0`, null, data.header), {
            "GET /v1beta/${constant.namespace}/connectors?page_size=0 response status is 200": (r) => r.status === 200,
            "GET /v1beta/${constant.namespace}/connectors?page_size=0 response all records": (r) => r.json().connectors.length === limitedRecords.json().connectors.length,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1`, null, data.header), {
            "GET /v1beta/${constant.namespace}/connectors?page_size=1 response status is 200": (r) => r.status === 200,
            "GET /v1beta/${constant.namespace}/connectors?page_size=1 response connectors size 1": (r) => r.json().connectors.length === 1,
        });

        var pageRes = http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1`, null, data.header)
        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors?page_size=1&page_token=${pageRes.json().next_page_token}`, null, data.header), {
            [`GET /v1beta/${constant.namespace}/connectors?page_size=1&page_token=${pageRes.json().next_page_token} response status is 200`]: (r) => r.status === 200,
            [`GET /v1beta/${constant.namespace}/connectors?page_size=1&page_token=${pageRes.json().next_page_token} response connectors size 1`]: (r) => r.json().connectors.length === 1,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1&view=VIEW_BASIC`, null, data.header), {
            "GET /v1beta/${constant.namespace}/connectors?page_size=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
            "GET /v1beta/${constant.namespace}/connectors?page_size=1&view=VIEW_BASIC response connectors[0].configuration is null": (r) => r.json().connectors[0].configuration === null,
            "GET /v1beta/${constant.namespace}/connectors?page_size=1&view=VIEW_BASIC response connectors[0].owner is invalid": (r) => r.json().connectors[0].owner === undefined,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1&view=VIEW_FULL`, null, data.header), {
            "GET /v1beta/${constant.namespace}/connectors?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /v1beta/${constant.namespace}/connectors?page_size=1&view=VIEW_FULL response connectors[0].configuration is not null": (r) => r.json().connectors[0].configuration !== null,
            "GET /v1beta/${constant.namespace}/connectors?page_size=1&view=VIEW_FULL response connectors[0].connector_definition_detail is not null": (r) => r.json().connectors[0].connector_definition_detail !== null,
            "GET /v1beta/${constant.namespace}/connectors?page_size=1&view=VIEW_FULL response connectors[0].owner is valid": (r) => helper.isValidOwner(r.json().connectors[0].owner, data.expectedOwner),
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1`, null, data.header), {
            "GET /v1beta/${constant.namespace}/connectors?page_size=1 response status 200": (r) => r.status === 200,
            "GET /v1beta/${constant.namespace}/connectors?page_size=1 response connectors[0].configuration is null": (r) => r.json().connectors[0].configuration === null,
            "GET /v1beta/${constant.namespace}/connectors?page_size=1 response connectors[0].owner is invalid": (r) => r.json().connectors[0].owner === undefined,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=${limitedRecords.json().total_size}`, null, data.header), {
            [`GET /v1beta/${constant.namespace}/connectors?page_size=${limitedRecords.json().total_size} response status 200`]: (r) => r.status === 200,
            [`GET /v1beta/${constant.namespace}/connectors?page_size=${limitedRecords.json().total_size} response next_page_token is empty`]: (r) => r.json().next_page_token === ""
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${reqBody.id}`, null, data.header), {
                [`DELETE /v1beta/${constant.namespace}/connectors x${reqBodies.length} response status is 204`]: (r) => r.status === 204,
            });
        }
    });
}

export function CheckGet(data) {

    group("Connector API: Get destination connectors by ID", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig

        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), data.header)

        http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${csvDstConnector.id}/connect`,
            {}, data.header)

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, data.header), {
            [`GET /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status 200`]: (r) => r.status === 200,
            [`GET /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response connector id`]: (r) => r.json().connector.id === csvDstConnector.id,
            [`GET /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response connector connector_definition_name permalink`]: (r) => r.json().connector.connector_definition_name === constant.csvDstDefRscName,
            [`GET /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response connector owner is invalid`]: (r) => r.json().connector.owner === undefined,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch`, null, data.header), {
            [`GET /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, data.header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckUpdate(data) {

    group("Connector API: Update destination connectors", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig

        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), data.header)

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "connector_definition_name": csvDstConnector.connector_definition_name,
            "tombstone": true,
            "description": randomString(50),
            "configuration": {
                destination_path: "/tmp"
            }
        }

        var resCSVDstUpdate = http.request("PATCH", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`,
            JSON.stringify(csvDstConnectorUpdate), data.header)

        check(resCSVDstUpdate, {
            [`PATCH /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status 200`]: (r) => r.status === 200,
            [`PATCH /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response connector id`]: (r) => r.json().connector.id === csvDstConnectorUpdate.id,
            [`PATCH /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response connector connector_definition_name`]: (r) => r.json().connector.connector_definition_name === constant.csvDstDefRscName,
            [`PATCH /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response connector description`]: (r) => r.json().connector.description === csvDstConnectorUpdate.description,
            [`PATCH /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response connector tombstone`]: (r) => r.json().connector.tombstone === false,
            [`PATCH /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response connector configuration`]: (r) => r.json().connector.configuration.destination_path === csvDstConnectorUpdate.configuration.destination_path,
            [`PATCH /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response connector owner is valid`]: (r) => helper.isValidOwner(r.json().connector.owner, data.expectedOwner),
        });

        // Try to update with empty description
        csvDstConnectorUpdate = {

            "description": "",

        }

        resCSVDstUpdate = http.request("PATCH", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`,
            JSON.stringify(csvDstConnectorUpdate), data.header)

        check(resCSVDstUpdate, {
            [`PATCH /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} with empty description response status 200`]: (r) => r.status === 200,
            [`PATCH /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} with empty description response empty description`]: (r) => r.json().connector.description === csvDstConnectorUpdate.description,
        })

        // Try to update with a non-existing name field (which should be ignored because name field is OUTPUT_ONLY)
        csvDstConnectorUpdate = {
            "name": `${constant.namespace}/connectors/${randomString(5)}`,
            "description": randomString(50),

        }

        resCSVDstUpdate = http.request("PATCH", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`,
            JSON.stringify(csvDstConnectorUpdate), data.header)

        check(resCSVDstUpdate, {
            [`PATCH /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} with non-existing name field response status 200`]: (r) => r.status === 200,
        })

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${csvDstConnector.id}`, null, data.header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${csvDstConnector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckLookUp(data) {

    group("Connector API: Look up destination connectors by UID", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), data.header)

        check(http.request("GET", `${pipelinePublicHost}/v1beta/connectors/${resCSVDst.json().connector.uid}/lookUp`, null, data.header), {
            [`GET /v1beta/connectors/${resCSVDst.json().connector.uid}/lookUp response status 200`]: (r) => r.status === 200,
            [`GET /v1beta/connectors/${resCSVDst.json().connector.uid}/lookUp response connector uid`]: (r) => r.json().connector.uid === resCSVDst.json().connector.uid,
            [`GET /v1beta/connectors/${resCSVDst.json().connector.uid}/lookUp response connector connector_definition_name`]: (r) => r.json().connector.connector_definition_name === constant.csvDstDefRscName,
            [`GET /v1beta/connectors/${resCSVDst.json().connector.uid}/lookUp response connector owner is invalid`]: (r) => r.json().connector.owner === undefined,
        });

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, data.header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckConnect(data) {
    group("Connector API: Check Connect", () => {


        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination": "airbyte-destination-csv",
            }
        }

        // Cannot connect with unfinished config
        var resDstCsv = http.request(
            "POST",
            `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), data.header)

        check(resDstCsv, {
            "POST /v1beta/${constant.namespace}/connectors response status for creating MySQL destination connector 201": (r) => r.status === 201,
            "POST /v1beta/${constant.namespace}/connectors response connector name": (r) => r.json().connector.name == `${constant.namespace}/connectors/${csvDstConnector.id}`,
            "POST /v1beta/${constant.namespace}/connectors response connector uid": (r) => helper.isUUID(r.json().connector.uid),
            "POST /v1beta/${constant.namespace}/connectors response connector connector_definition_name": (r) => r.json().connector.connector_definition_name === constant.csvDstDefRscName,
            "POST /v1beta/${constant.namespace}/connectors response connector owner is valid": (r) => helper.isValidOwner(r.json().connector.owner, data.expectedOwner),
        });

        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resDstCsv.json().connector.id}/connect`, null, data.header), {
            [`POST /v1beta/${constant.namespace}/connectors/${resDstCsv.json().connector.id}/connect response status 400`]: (r) => r.status === 400,
        });

        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resDstCsv.json().connector.id}/disconnect`, null, data.header), {
            [`POST /v1beta/${constant.namespace}/connectors/${resDstCsv.json().connector.id}/disconnect response status 200`]: (r) => r.status === 200,
        });

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resDstCsv.json().connector.id}`, null, data.header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${resDstCsv.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        // Cannot connect with unfinished config
        var resDstCsv = http.request(
            "POST",
            `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), data.header)

        check(resDstCsv, {
            "POST /v1beta/${constant.namespace}/connectors response status for creating MySQL destination connector 201": (r) => r.status === 201,
            "POST /v1beta/${constant.namespace}/connectors response connector name": (r) => r.json().connector.name == `${constant.namespace}/connectors/${csvDstConnector.id}`,
            "POST /v1beta/${constant.namespace}/connectors response connector uid": (r) => helper.isUUID(r.json().connector.uid),
            "POST /v1beta/${constant.namespace}/connectors response connector connector_definition_name": (r) => r.json().connector.connector_definition_name === constant.csvDstDefRscName,
            "POST /v1beta/${constant.namespace}/connectors response connector owner is valid": (r) => helper.isValidOwner(r.json().connector.owner, data.expectedOwner),
        });

        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resDstCsv.json().connector.id}/connect`, null, data.header), {
            [`POST /v1beta/${constant.namespace}/connectors/${resDstCsv.json().connector.id}/connect response status 200`]: (r) => r.status === 200,
        });

        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resDstCsv.json().connector.id}/disconnect`, null, data.header), {
            [`POST /v1beta/${constant.namespace}/connectors/${resDstCsv.json().connector.id}/disconnect response status 200`]: (r) => r.status === 200,
        });

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resDstCsv.json().connector.id}`, null, data.header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${resDstCsv.json().connector.id} response status 204`]: (r) => r.status === 204,
        });





    });
}

export function CheckState(data) {

    group("Connector API: Change state destination connectors", () => {
        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig

        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), data.header)
        http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${csvDstConnector.id}/connect`,
            {}, data.header)

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch`, null, data.header), {
            [`GET /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/connect`, null, data.header), {
            [`POST /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/connect response status 200 (with STATE_CONNECTED)`]: (r) => r.status === 200,
        });

        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/disconnect`, null, data.header), {
            [`POST /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/disconnect response status 200 (with STATE_CONNECTED)`]: (r) => r.status === 200,
        });

        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/disconnect`, null, data.header), {
            [`POST /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/disconnect response status 200 (with STATE_DISCONNECTED)`]: (r) => r.status === 200,
        });

        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/connect`, null, data.header), {
            [`POST /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/connect response status 200 (with STATE_DISCONNECTED)`]: (r) => r.status === 200,
        });

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch`, null, data.header), {
            [`GET /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, data.header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckRename(data) {

    group("Connector API: Rename destination connectors", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig

        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), data.header)

        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/rename`,
            JSON.stringify({
                "new_connector_id": `some_id_not_${resCSVDst.json().connector.id}`
            }), data.header), {
            [`POST /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/rename response status 200`]: (r) => r.status === 200,
            [`POST /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/rename response id is some_id_not_${resCSVDst.json().connector.id}`]: (r) => r.json().connector.id === `some_id_not_${resCSVDst.json().connector.id}`,
        });

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/some_id_not_${resCSVDst.json().connector.id}`, null, data.header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/some_id_not_${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckTest(data) {

    group("Connector API: Test destination connectors by ID", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), data.header)

        http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${csvDstConnector.id}/connect`,
            {}, data.header)

        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/testConnection`, null, data.header), {
            [`POST /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/testConnection response status 200`]: (r) => r.status === 200,
            [`POST /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/testConnection response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        });

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, data.header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

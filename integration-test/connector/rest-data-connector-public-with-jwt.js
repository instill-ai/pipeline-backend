import http from "k6/http";
import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js"

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate(header) {

    group(`Connector API: Create destination connectors [with random "Instill-User-Uid" header]`, () => {

        // end
        var httpDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig,
        }


        // Cannot create http destination connector of a non-exist user
        check(http.request("POST",
            `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(httpDstConnector), constant.paramsHTTPWithJwt), {
            [`[with random "Instill-User-Uid" header] POST /v1beta/${constant.namespace}/connectors response for creating HTTP destination status is 401`]: (r) => r.status === 401,
        });

    });

}

export function CheckList(header) {

    group(`Connector API: List destination connectors [with random "Instill-User-Uid" header]`, () => {

        // Cannot list destination connector of a non-exist user
        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors?filter=connector_type=CONNECTOR_TYPE_DATA`, null, constant.paramsHTTPWithJwt), {
            [`[with random "Instill-User-Uid" header] GET /v1beta/${constant.namespace}/connectors response status is 401`]: (r) => r.status === 401,
        });
    });
}

export function CheckGet(header) {

    group(`Connector API: Get destination connectors by ID [with random "Instill-User-Uid" header]`, () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), header)

        http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${csvDstConnector.id}/connect`,
            {}, header)

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch`, null, header), {
            [`GET /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        // Cannot get a destination connector of a non-exist user
        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, constant.paramsHTTPWithJwt), {
            [`[with random "Instill-User-Uid" header] GET /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status is 401`]: (r) => r.status === 401,
        });

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckUpdate(header) {

    group(`Connector API: Update destination connectors [with random "Instill-User-Uid" header]`, () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), header)

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "connector_definition_name": csvDstConnector.connector_definition_name,
            "tombstone": true,
            "description": randomString(50),
            "configuration": {
                destination_path: "/tmp"
            }
        }

        // Cannot patch a destination connector of a non-exist user
        check(http.request(
            "PATCH",
            `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`,
            JSON.stringify(csvDstConnectorUpdate), constant.paramsHTTPWithJwt), {
            [`[with random "Instill-User-Uid" header] PATCH /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status 401`]: (r) => r.status === 401,
        });

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${csvDstConnector.id}`, null, header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${csvDstConnector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckLookUp(header) {

    group(`Connector API: Look up destination connectors by UID [with random "Instill-User-Uid" header]`, () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), header)

        // Cannot look up a destination connector of a non-exist user
        check(http.request("GET", `${pipelinePublicHost}/v1beta/connectors/${resCSVDst.json().connector.uid}/lookUp`, null, constant.paramsHTTPWithJwt), {
            [`[with random "Instill-User-Uid" header] GET /v1beta/connectors/${resCSVDst.json().connector.uid}/lookUp response status 401`]: (r) => r.status === 401,
        });

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckState(header) {

    group(`Connector API: Change state destination connectors [with random "Instill-User-Uid" header]`, () => {
        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), header)

        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/disconnect`, null, constant.paramsHTTPWithJwt), {
            [`[with random "Instill-User-Uid" header] POST /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/disconnect response at UNSPECIFIED state status 401`]: (r) => r.status === 401,
        });

        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/connect`, null, constant.paramsHTTPWithJwt), {
            [`[with random "Instill-User-Uid" header] POST /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/connect response at UNSPECIFIED state status 401`]: (r) => r.status === 401,
        });

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckRename(header) {

    group(`Connector API: Rename destination connectors [with random "Instill-User-Uid" header]`, () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), header)

        // Cannot rename destination connector of a non-exist user
        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/rename`,
            JSON.stringify({
                "new_connector_id": `some_id_not_${resCSVDst.json().connector.id}`
            }), constant.paramsHTTPWithJwt), {
            [`[with random "Instill-User-Uid" header] POST /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/rename response status 401`]: (r) => r.status === 401,
        });

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${csvDstConnector.id}`, null, header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${csvDstConnector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckExecute(header) {

    group(`Connector API: Write destination connectors [with random "Instill-User-Uid" header]`, () => {

        var csvDstConnector, resCSVDst, currentTime, timeoutTime

        // Write classification output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination_path": "/local/test-classification"
            },
        }

        resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), header)

        http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${csvDstConnector.id}/connect`,
            {}, header)

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch`, null, header), {
            [`[with random "Instill-User-Uid" header] GET /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        // Cannot write to destination connector of a non-exist user
        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/execute`,
            JSON.stringify({
                "inputs": constant.clsModelOutputs
            }), constant.paramsHTTPWithJwt), {
            [`[with random "Instill-User-Uid" header] POST /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/execute response status 401 (classification)`]: (r) => r.status === 401,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status 204 (classification)`]: (r) => r.status === 204,
        });
    });
}

export function CheckTest(header) {

    group(`Connector API: Test destination connectors by ID [with random "Instill-User-Uid" header]`, () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
            JSON.stringify(csvDstConnector), header)

        http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${csvDstConnector.id}/connect`,
            {}, header)

        check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch`, null, header), {
            [`GET /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        // Cannot test destination connector of a non-exist user
        check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/testConnection`, null, constant.paramsHTTPWithJwt), {
            [`[with random "Instill-User-Uid" header] POST /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}/testConnection response status is 401`]: (r) => r.status === 401,
        });

        check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id}`, null, header), {
            [`DELETE /v1beta/${constant.namespace}/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

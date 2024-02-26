import grpc from 'k6/net/grpc';
import {
    check,
    group,
    sleep
} from "k6";
import {
    randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"
import * as helper from "./helper.js"

const client = new grpc.Client();
client.load(['../proto/vdp/pipeline/v1beta'], 'pipeline_public_service.proto');

export function CheckCreate(data) {

    group("Connector API: Create destination connectors", () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        // destination-csv
        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)

        check(resCSVDst, {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV response StatusOK": (r) => r.status === grpc.StatusOK,
        });
        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/GetUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/GetUserConnector CSV ${resCSVDst.message.connector.id} response STATE_CONNECTED`]: (r) => r.message.connector.state === "STATE_CONNECTED",
        });

        // destination-mysql (will end up with STATE_ERROR)
        var mySQLDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.mySQLDstDefRscName,
            "configuration": {
                "destination": "airbyte-destiniation-mysql",
                "host": randomString(10),
                "port": 3306,
                "username": randomString(10),
                "database": randomString(10),
            }
        }

        var resDstMySQL = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector',
            {
                parent: `${constant.namespace}`,
                connector: mySQLDstConnector,
            }, data.metadata
        )
        var resp = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${mySQLDstConnector.id}`
        }, data.metadata)

        check(resDstMySQL, {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector MySQL response StatusOK": (r) => r.status === grpc.StatusOK,
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector MySQL response destinationConnector name": (r) => r.message.connector.name == `${constant.namespace}/connectors/${mySQLDstConnector.id}`,
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector MySQL response destinationConnector uid": (r) => helper.isUUID(r.message.connector.uid),
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector MySQL response destinationConnector connectorDefinition": (r) => r.message.connector.connectorDefinitionName === constant.mySQLDstDefRscName,
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector MySQL response destinationConnector owner is valid": (r) => helper.isValidOwner(r.message.connector.owner, data.expectedOwner),
        });

        // TODO: check jsonschema when connect

        // check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
        //     name: `${constant.namespace}/connectors/${resDstMySQL.message.connector.id}`
        // }), {
        //     "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector MySQL destination connector ended up STATE_ERROR": (r) => r.message.state === "STATE_ERROR",
        // })



        // check JSON Schema failure cases
        // var jsonSchemaFailedBodyCSV = {
        //     "id": randomString(10),
        //     "connector_definition_name": constant.csvDstDefRscName,
        //     "description": randomString(50),
        //     "configuration": {} // required destination_path
        // }

        // check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
        //     connector: jsonSchemaFailedBodyCSV
        // }), {
        //     "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector response status for JSON Schema failed body 400 (destination-csv missing destination_path)": (r) => r.status === grpc.StatusInvalidArgument,
        // });

        // var jsonSchemaFailedBodyMySQL = {
        //     "id": randomString(10),
        //     "connector_definition_name": constant.mySQLDstDefRscName,
        //     "description": randomString(50),
        //     "configuration": {
        //         "host": randomString(10),
        //         "port": "3306",
        //         "username": randomString(10),
        //         "database": randomString(10),
        //     } // required port integer type
        // }

        // check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
        //     connector: jsonSchemaFailedBodyMySQL
        // }), {
        //     "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector response status for JSON Schema failed body 400 (destination-mysql port not integer)": (r) => r.status === grpc.StatusInvalidArgument,
        // });

        // Delete test records
        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resDstMySQL.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resDstMySQL.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });

}

export function CheckList(data) {

    group("Connector API: List destination connectors", () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors response connectors array is 0 length`]: (r) => r.message.connectors.length === 0,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors response totalSize is 0`]: (r) => r.message.totalSize == 0,
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
            var resDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
                parent: `${constant.namespace}`,
                connector: reqBody
            }, data.metadata)
            client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
                name: `${constant.namespace}/connectors/${reqBody.id}`
            }, data.metadata)

            check(resDst, {
                [`vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector x${reqBodies.length} HTTP response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors response has connectors array`]: (r) => Array.isArray(r.message.connectors),
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors response has totalSize = ${reqBodies.length}`]: (r) => r.message.totalSize == reqBodies.length,
        });

        var limitedRecords = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, data.metadata)
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 0
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=0 response all records`]: (r) => r.message.connectors.length === limitedRecords.message.connectors.length,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 response size 1`]: (r) => r.message.connectors.length === 1,
        });

        var pageRes = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
            pageToken: `${pageRes.message.nextPageToken}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 pageToken=${pageRes.message.nextPageToken} response size 1`]: (r) => r.message.connectors.length === 1,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
            view: "VIEW_BASIC"
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 view=VIEW_BASIC response connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration === null,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 view=VIEW_BASIC response connectors[0].owner is invalid`]: (r) => r.message.connectors[0].owner === undefined,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
            view: "VIEW_FULL"
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 view=VIEW_FULL response connectors[0].configuration is not null`]: (r) => r.message.connectors[0].configuration !== null,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 view=VIEW_FULL response connectors[0].connectorDefinitionDetail is not null`]: (r) => r.message.connectors[0].connectorDefinitionDetail !== null,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 view=VIEW_FULL response connectors[0].owner is valid`]: (r) => helper.isValidOwner(r.message.connectors[0].owner, data.expectedOwner),
        });


        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 response connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration === null,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=1 response connectors[0].owner is invalid`]: (r) => r.message.connectors[0].owner === undefined,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: `${limitedRecords.message.totalSize}`,
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors pageSize=${limitedRecords.message.totalSize} response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
                name: `${constant.namespace}/connectors/${reqBody.id}`
            }, data.metadata), {
                [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        client.close();
    });
}

export function CheckGet(data) {

    group("Connector API: Get destination connectors by ID", () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)

        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/GetUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/GetUserConnector CSV ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/GetUserConnector CSV ${resCSVDst.message.connector.id} response connector id`]: (r) => r.message.connector.id === csvDstConnector.id,
            [`vdp.pipeline.v1beta.PipelinePublicService/GetUserConnector CSV ${resCSVDst.message.connector.id} response connector connectorDefinition permalink`]: (r) => r.message.connector.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.pipeline.v1beta.PipelinePublicService/GetUserConnector CSV ${resCSVDst.message.connector.id} response connector owner is invalid`]: (r) => r.message.connector.owner === undefined,
        });

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckUpdate(data) {

    group("Connector API: Update destination connectors", () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "name": `${constant.namespace}/connectors/${csvDstConnector.id}`,
            "connector_definition_name": csvDstConnector.connector_definition_name,
            "tombstone": true,
            "description": randomString(50),
            "configuration": {
                destination_path: "/tmp"
            }
        }

        var resCSVDstUpdate = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector', {
            connector: csvDstConnectorUpdate,
            update_mask: "description,configuration",
        }, data.metadata)

        check(resCSVDstUpdate, {
            [`vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector ${resCSVDstUpdate.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector ${resCSVDstUpdate.message.connector.id} response connector connectorDefinition`]: (r) => r.message.connector.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector ${resCSVDstUpdate.message.connector.id} response connector description`]: (r) => r.message.connector.description === csvDstConnectorUpdate.description,
            [`vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector ${resCSVDstUpdate.message.connector.id} response connector tombstone`]: (r) => r.message.connector.tombstone === false,
            [`vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector ${resCSVDstUpdate.message.connector.id} response connector configuration`]: (r) => r.message.connector.configuration.destination_path === csvDstConnectorUpdate.configuration.destination_path,
            [`vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector ${resCSVDstUpdate.message.connector.id} response connector owner is valid`]: (r) => helper.isValidOwner(r.message.connector.owner, data.expectedOwner),
        });

        // Try to update with empty description
        csvDstConnectorUpdate = {
            "name": `${constant.namespace}/connectors/${csvDstConnector.id}`,
            "description": "",
        }

        resCSVDstUpdate = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector', {
            connector: csvDstConnectorUpdate,
            update_mask: "description",
        }, data.metadata)

        check(resCSVDstUpdate, {
            [`vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector ${resCSVDstUpdate.message.connector.id} with empty description response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector ${resCSVDstUpdate.message.connector.id} with empty description response connector description`]: (r) => r.message.connector.description === csvDstConnectorUpdate.description,
            [`vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector ${resCSVDstUpdate.message.connector.id} with empty description response connector owner is valid`]: (r) => helper.isValidOwner(r.message.connector.owner, data.expectedOwner),
        });

        // Try to update with a non-existing name field (which should be ignored because name field is OUTPUT_ONLY)
        csvDstConnectorUpdate = {
            "name": `${constant.namespace}/connectors/${randomString(5)}`,
            "description": randomString(50),
        }

        resCSVDstUpdate = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector', {
            connector: csvDstConnectorUpdate,
            update_mask: "description",
        }, data.metadata)
        check(resCSVDstUpdate, {
            [`vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector with non-existing name field response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        });

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckLookUp(data) {

    group("Connector API: Look up destination connectors by UID", () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/LookUpConnector', {
            permalink: `connectors/${resCSVDst.message.connector.uid}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/LookUpConnector CSV ${resCSVDst.message.connector.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/LookUpConnector CSV ${resCSVDst.message.connector.uid} response connector id`]: (r) => r.message.connector.uid === resCSVDst.message.connector.uid,
            [`vdp.pipeline.v1beta.PipelinePublicService/LookUpConnector CSV ${resCSVDst.message.connector.uid} response connector connectorDefinition permalink`]: (r) => r.message.connector.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.pipeline.v1beta.PipelinePublicService/LookUpConnector CSV ${resCSVDst.message.connector.uid} response connector owner is invalid`]: (r) => r.message.connector.owner === undefined,
        });

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckState(data) {

    group("Connector API: Change state destination connectors", () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)
        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/DisconnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DisconnectUserConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/DisconnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DisconnectUserConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/DisconnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DisconnectUserConnector ${resCSVDst.message.connector.id} response at STATE_DISCONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector ${resCSVDst.message.connector.id} response at STATE_DISCONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckRename(data) {

    group("Connector API: Rename destination connectors", () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)

        let new_id = `some_id_not_${resCSVDst.message.connector.id}`

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/RenameUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`,
            new_connector_id: new_id
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/RenameUserConnector ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/RenameUserConnector ${resCSVDst.message.connector.id} response id is some_id_not_${resCSVDst.message.connector.id}`]: (r) => r.message.connector.id === `some_id_not_${resCSVDst.message.connector.id}`,
        });

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${new_id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${new_id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckExecute(data) {

    group("Connector API: Write destination connectors", () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector, resCSVDst, currentTime, timeoutTime

        // Write classification output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination": "airbyte-destination-csv",
                "destination_path": "/local/test-classification"
            },
        }

        resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)

        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector', {
            "name": `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`,
            "inputs": constant.clsModelOutputs
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector ${resCSVDst.message.connector.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write detection output (empty bounding_boxes)
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination": "airbyte-destination-csv",
                "destination_path": "/local/test-detection-empty-bounding-boxes"
            },
        }

        resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)
        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector', {
            "name": `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`,
            "inputs": constant.detectionEmptyModelOutputs
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector ${resCSVDst.message.connector.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write detection output (multiple models)
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination": "airbyte-destination-csv",
                "destination_path": "/local/test-detection-multi-models"
            },
        }

        resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)
        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector', {
            "name": `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`,
            "inputs": constant.detectionModelOutputs
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector ${resCSVDst.message.connector.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write keypoint output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination": "airbyte-destination-csv",
                "destination_path": "/local/test-keypoint"
            },
        }


        resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)
        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector', {
            "name": `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`,
            "inputs": constant.keypointModelOutputs
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector ${resCSVDst.message.connector.id} response (keypoint) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response (keypoint) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write ocr output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination": "airbyte-destination-csv",
                "destination_path": "/local/test-ocr"
            },
        }


        resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)
        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector', {
            "name": `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`,
            "inputs": constant.ocrModelOutputs
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector ${resCSVDst.message.connector.id} response (ocr) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });
        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response (ocr) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write semantic segmentation output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination": "airbyte-destination-csv",
                "destination_path": "/local/test-semantic-segmentation"
            },
        }

        resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)
        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector', {
            "name": `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`,
            "inputs": constant.semanticSegModelOutputs
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector ${resCSVDst.message.connector.id} response (semantic-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response (semantic-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write instance segmentation output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination": "airbyte-destination-csv",
                "destination_path": "/local/test-instance-segmentation"
            },
        }

        resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)
        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector', {
            "name": `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`,
            "inputs": constant.instSegModelOutputs
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector ${resCSVDst.message.connector.id} response (instance-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response (instance-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write text-to-image output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination": "airbyte-destination-csv",
                "destination_path": "/local/test-text-to-image"
            },
        }

        resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)
        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector', {
            "name": `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`,
            "inputs": constant.textToImageModelOutputs
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector ${resCSVDst.message.connector.id} response (text-to-image) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response (text-to-image) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write unspecified output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination": "airbyte-destination-csv",
                "destination_path": "/local/test-unspecified"
            },
        }

        resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)
        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector', {
            "name": `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`,
            "inputs": constant.unspecifiedModelOutputs
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ExecuteUserConnector ${resCSVDst.message.connector.id} response (unspecified) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response (unspecified) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckTest(data) {

    group("Connector API: Test destination connectors by ID", () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, data.metadata)

        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata)

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/TestUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/TestUserConnector CSV ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/TestUserConnector CSV ${resCSVDst.message.connector.id} response connector STATE_CONNECTED`]: (r) => r.message.state === "STATE_CONNECTED",
        });

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

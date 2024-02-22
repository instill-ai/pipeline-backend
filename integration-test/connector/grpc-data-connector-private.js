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

const clientPrivate = new grpc.Client();
const clientPublic = new grpc.Client();
clientPrivate.load(['../proto/vdp/pipeline/v1beta'], 'pipeline_private_service.proto');
clientPublic.load(['../proto/vdp/pipeline/v1beta'], 'pipeline_public_service.proto');

export function CheckList(metadata) {

    group("Connector API: List data connectors by admin", () => {

        clientPrivate.connect(constant.pipelineGRPCPrivateHost, {
            plaintext: true
        });

        clientPublic.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        check(clientPrivate.invoke('vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin', {}, {}), {
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin response connectors array is 0 length`]: (r) => r.message.connectors.length === 0,
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin response totalSize is 0`]: (r) => r.message.totalSize == 0,
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
            var resDst = clientPublic.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
                parent: `${constant.namespace}`,
                connector: reqBody
            }, metadata)
            clientPublic.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
                name: `${constant.namespace}/connectors/${resDst.message.connector.id}`
            }, metadata)

            check(resDst, {
                [`vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector x${reqBodies.length} HTTP response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        check(clientPrivate.invoke('vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin', {}, {}), {
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectorAdmin response has connectors array`]: (r) => Array.isArray(r.message.connectors),
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin response has totalSize = ${reqBodies.length}`]: (r) => r.message.totalSize == reqBodies.length,
        });

        var limitedRecords = clientPrivate.invoke('vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin', {}, {})
        check(clientPrivate.invoke('vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin', {
            pageSize: 0
        }, {}), {
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=0 response all records`]: (r) => r.message.connectors.length === limitedRecords.message.connectors.length,
        });

        check(clientPrivate.invoke('vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin', {
            pageSize: 1
        }, {}), {
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 response size 1`]: (r) => r.message.connectors.length === 1,
        });

        var pageRes = clientPrivate.invoke('vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin', {
            pageSize: 1
        }, {})

        check(clientPrivate.invoke('vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin', {
            pageSize: 1,
            pageToken: `${pageRes.message.nextPageToken}`
        }, {}), {
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 pageToken=${pageRes.message.nextPageToken} response size 1`]: (r) => r.message.connectors.length === 1,
        });

        check(clientPrivate.invoke('vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin', {
            pageSize: 1,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_BASIC response connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration === null,
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_BASIC response connectors[0].owner is invalid`]: (r) => r.message.connectors[0].owner === undefined,
        });

        check(clientPrivate.invoke('vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin', {
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_FULL response connectors[0].configuration is not null`]: (r) => r.message.connectors[0].configuration !== null,
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_FULL response connectors[0].connectorDefinitionDetail is not null`]: (r) => r.message.connectors[0].connectorDefinitionDetail !== null,
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_FULL response connectors[0].owner is valid`]: (r) => helper.isValidOwnerGRPC(r.message.connectors[0].owner),
        });


        check(clientPrivate.invoke('vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin', {
            pageSize: 1,
        }, {}), {
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 response connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration === null,
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=1 response connectors[0].owner is invalid`]: (r) => r.message.connectors[0].owner === undefined,
        });

        check(clientPrivate.invoke('vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin', {
            pageSize: `${limitedRecords.message.totalSize}`,
        }, {}), {
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePrivateService/ListConnectorsAdmin pageSize=${limitedRecords.message.totalSize} response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
        });

        // Delete the data connectors
        for (const reqBody of reqBodies) {
            check(clientPublic.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
                name: `${constant.namespace}/connectors/${reqBody.id}`
            }, metadata), {
                [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        clientPrivate.close();
        clientPublic.close();
    });
}

export function CheckLookUp(metadata) {

    group("Connector API: Look up data connectors by UID by admin", () => {

        clientPrivate.connect(constant.pipelineGRPCPrivateHost, {
            plaintext: true
        });

        clientPublic.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = clientPublic.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, metadata)

        clientPublic.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, metadata)

        check(clientPrivate.invoke('vdp.pipeline.v1beta.PipelinePrivateService/LookUpConnectorAdmin', {
            permalink: `connectors/${resCSVDst.message.connector.uid}`
        }), {
            [`vdp.pipeline.v1beta.PipelinePrivateService/LookUpConnectorAdmin CSV ${resCSVDst.message.connector.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePrivateService/LookUpConnectorAdmin CSV ${resCSVDst.message.connector.uid} response connector id`]: (r) => r.message.connector.uid === resCSVDst.message.connector.uid,
            [`vdp.pipeline.v1beta.PipelinePrivateService/LookUpConnectorAdmin CSV ${resCSVDst.message.connector.uid} response connector connectorDefinition permalink`]: (r) => r.message.connector.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.pipeline.v1beta.PipelinePrivateService/LookUpConnectorAdmin CSV ${resCSVDst.message.connector.uid} response connector owner is invalid`]: (r) => r.message.connector.owner === undefined,
        });

        check(clientPublic.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        clientPublic.close();
    });
}

import grpc from 'k6/net/grpc';
import {
    check,
    group
} from "k6";

import * as constant from "./const.js"

import {
    deepEqual
} from "./helper.js"

const client = new grpc.Client();
client.load(['../proto/vdp/pipeline/v1beta'], 'pipeline_public_service.proto');

export function CheckList(data) {

    group("Connector API: List data connector definitions", () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions', {
            filter: "connector_type=CONNECTOR_TYPE_DATA"
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions response connectorDefinitions array`]: (r) => Array.isArray(r.message.connectorDefinitions),
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions response totalSize > 0`]: (r) => r.message.totalSize > 0,
        });

        var limitedRecords = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions', {
            filter: "connector_type=CONNECTOR_TYPE_DATA"
        }, {})
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 0
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=0 response connectorDefinitions length = 1`]: (r) => r.message.connectorDefinitions.length === limitedRecords.message.connectorDefinitions.length,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=1 response connectorDefinitions length = 1`]: (r) => r.message.connectorDefinitions.length === 1,
        });

        var pageRes = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1
        }, data.metadata)
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
            pageToken: pageRes.message.nextPageToken
        }, {}), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=1 pageToken=${pageRes.message.nextPageToken} response connectorDefinitions length = 1`]: (r) => r.message.connectorDefinitions.length === 1,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
            view: "VIEW_BASIC"
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=1 view=VIEW_BASIC response connectorDefinitions connectorDefinition spec is null`]: (r) => r.message.connectorDefinitions[0].spec === null,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
            view: "VIEW_FULL"
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=1 view=VIEW_FULL response connectorDefinitions connectorDefinition spec is not null`]: (r) => r.message.connectorDefinitions[0].spec !== null,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=1 response connectorDefinitions connectorDefinition spec is null`]: (r) => r.message.connectorDefinitions[0].spec === null,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: limitedRecords.message.totalSize,
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions pageSize=${limitedRecords.message.totalSize} response nextPageToken is null`]: (r) => r.message.nextPageToken === "",
        });

        client.close();
    });
}

export function CheckGet(data) {
    group("Connector API: Get data connector definition", () => {
        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var allRes = client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListConnectorDefinitions', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, data.metadata)
        var def = allRes.message.connectorDefinitions[0]

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition', {
            name: `connector-definitions/${def.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition id=${def.id}} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition id=${def.id} response has the exact record`]: (r) => deepEqual(r.message.connectorDefinition, def),
            [`vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition id=${def.id} has the non-empty resource name ${def.name}`]: (r) => r.message.connectorDefinition.name != "",
            [`vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition id=${def.id} has the resource name ${def.name}`]: (r) => r.message.connectorDefinition.name === def.name,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition', {
            name: `connector-definitions/${def.id}`,
            view: "VIEW_BASIC"
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition id=${def.id}} view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition id=${def.id} view=VIEW_BASIC response connectorDefinition.spec is null`]: (r) => r.message.connectorDefinition.spec === null,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition', {
            name: `connector-definitions/${def.id}`,
            view: "VIEW_FULL"
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition id=${def.id}} view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition id=${def.id} view=VIEW_FULL response connectorDefinition.spec is not null`]: (r) => r.message.connectorDefinition.spec !== null,
        });

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition', {
            name: `connector-definitions/${def.id}`,
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition id=${def.id}} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.pipeline.v1beta.PipelinePublicService/GetConnectorDefinition id=${def.id} response connectorDefinition.spec is null`]: (r) => r.message.connectorDefinition.spec === null,
        });

        client.close();
    });
}

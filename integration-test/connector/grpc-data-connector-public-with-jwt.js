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
client.load(['../proto/vdp/pipeline/v1alpha'], 'pipeline_public_service.proto');

export function CheckCreate(metadata) {

    group(`Connector API: Create destination connectors [with random "jwt-sub" header]`, () => {

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

        // Cannot create csv destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector CSV response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        // destination-mysql (will end up with STATE_ERROR)
        var mySQLDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.mySQLDstDefRscName,
            "configuration": {
                "host": randomString(10),
                "port": 3306,
                "username": randomString(10),
                "database": randomString(10),
            }
        }

        // Cannot create MySQL destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: mySQLDstConnector
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector MySQL response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        client.close();
    });

}

export function CheckList(metadata) {

    group(`Connector API: List destination connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        // Cannot list destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListUserConnectors', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/ListUserConnectors response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        client.close();
    });
}

export function CheckGet(metadata) {

    group(`Connector API: Get destination connectors by ID [with random "jwt-sub" header]`, () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, metadata)

        // client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ConnectUserConnector', {
        //     name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        // })

        // check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/WatchUserConnector', {
        //     name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        // }), {
        //     "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        // })

        // Cannot get destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/GetUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/GetUserConnector CSV ${resCSVDst.message.connector.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, metadata), {
            [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckUpdate(metadata) {

    group(`Connector API: Update destination connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, metadata)

        client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, metadata)

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

        // Cannot update destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserConnector', {
            connector: csvDstConnectorUpdate,
            update_mask: "description,configuration",
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/UpdateUserConnector ${csvDstConnectorUpdate.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, metadata), {
            [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckLookUp(metadata) {

    group(`Connector API: Look up destination connectors by UID [with random "jwt-sub" header]`, () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, metadata)

        client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, metadata)

        // Cannot look up destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/LookUpConnector', {
            permalink: `connectors/${resCSVDst.message.connector.uid}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/LookUpConnector CSV ${resCSVDst.message.connector.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, metadata), {
            [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckState(metadata) {

    group(`Connector API: Change state destination connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, metadata)

        client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, metadata)

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/ConnectUserConnector ${resCSVDst.message.connector.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/DisconnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/DisconnectUserConnector ${resCSVDst.message.connector.id} response at UNSPECIFIED StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, metadata), {
            "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/ConnectUserConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/DisconnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/DisconnectUserConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/ConnectUserConnector ${resCSVDst.message.connector.id} response at STATE_DISCONNECTED state StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/DisconnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/DisconnectUserConnector ${resCSVDst.message.connector.id} response at STATE_DISCONNECTED state StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, metadata), {
            [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckRename(metadata) {

    group(`Connector API: Rename destination connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, metadata)

        client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, metadata)

        let new_id = `some-id-not-${resCSVDst.message.connector.id}`

        // Cannot rename destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/RenameUserConnector', {
            name: `${constant.namespace}/connectors/resCSVDst.message.connector.id`,
            new_connector_id: new_id
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/RenameUserConnector ${resCSVDst.message.connector.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, metadata), {
            [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckExecute(metadata) {

    group(`Connector API: Write destination connectors [with random "jwt-sub" header]`, () => {

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
                "destination_path": "/local/test-classification"
            },
        }

        resCSVDst = client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, metadata)

        client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, metadata)

        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, metadata), {
            "vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        // Cannot write destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ExecuteUserConnector', {
            "name": `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`,
            "inputs": constant.clsModelOutputs
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/ExecuteUserConnector ${resCSVDst.message.connector.id} response (classification) StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, metadata), {
            [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckTest(metadata) {

    group(`Connector API: Test destination connectors' connection [with random "jwt-sub" header]`, () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, metadata)

        client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, metadata)

        // Cannot test destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TestUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.pipeline.v1alpha.PipelinePublicService/TestUserConnector CSV ${resCSVDst.message.connector.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, metadata), {
            [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

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

    group(`Connector API: Create destination connectors [with random "Instill-User-Uid" header]`, () => {

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
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: csvDstConnector
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

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

        // Cannot create MySQL destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector', {
            parent: `${constant.namespace}`,
            connector: mySQLDstConnector
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector MySQL response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        client.close();
    });

}

export function CheckList(data) {

    group(`Connector API: List destination connectors [with random "Instill-User-Uid" header]`, () => {

        client.connect(constant.pipelineGRPCPublicHost, {
            plaintext: true
        });

        // Cannot list destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/ListUserConnectors response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        client.close();
    });
}

export function CheckGet(data) {

    group(`Connector API: Get destination connectors by ID [with random "Instill-User-Uid" header]`, () => {

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

        // client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
        //     name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        // })

        // check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
        //     name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        // }), {
        //     "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        // })

        // Cannot get destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/GetUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/GetUserConnector CSV ${resCSVDst.message.connector.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckUpdate(data) {

    group(`Connector API: Update destination connectors [with random "Instill-User-Uid" header]`, () => {

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

        client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
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

        // Cannot update destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector', {
            connector: csvDstConnectorUpdate,
            update_mask: "description,configuration",
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/UpdateUserConnector ${csvDstConnectorUpdate.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckLookUp(data) {

    group(`Connector API: Look up destination connectors by UID [with random "Instill-User-Uid" header]`, () => {

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

        // Cannot look up destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/LookUpConnector', {
            permalink: `connectors/${resCSVDst.message.connector.uid}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/LookUpConnector CSV ${resCSVDst.message.connector.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckState(data) {

    group(`Connector API: Change state destination connectors [with random "Instill-User-Uid" header]`, () => {

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

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector ${resCSVDst.message.connector.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/DisconnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/DisconnectUserConnector ${resCSVDst.message.connector.id} response at UNSPECIFIED StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/WatchUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, data.metadata), {
            "vdp.pipeline.v1beta.PipelinePublicService/CreateUserConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/DisconnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/DisconnectUserConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/ConnectUserConnector ${resCSVDst.message.connector.id} response at STATE_DISCONNECTED state StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/DisconnectUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/DisconnectUserConnector ${resCSVDst.message.connector.id} response at STATE_DISCONNECTED state StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
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

    group(`Connector API: Rename destination connectors [with random "Instill-User-Uid" header]`, () => {

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

        let new_id = `some_id_not_${resCSVDst.message.connector.id}`

        // Cannot rename destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/RenameUserConnector', {
            name: `${constant.namespace}/connectors/resCSVDst.message.connector.id`,
            new_connector_id: new_id
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/RenameUserConnector ${resCSVDst.message.connector.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckTest(data) {

    group(`Connector API: Test destination connectors' connection [with random "Instill-User-Uid" header]`, () => {

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

        // Cannot test destination connector of a non-exist user
        check(client.invoke('vdp.pipeline.v1beta.PipelinePublicService/TestUserConnector', {
            name: `${constant.namespace}/connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "Instill-User-Uid" header] vdp.pipeline.v1beta.PipelinePublicService/TestUserConnector CSV ${resCSVDst.message.connector.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector`, {
            name: `${constant.namespace}/connectors/${csvDstConnector.id}`
        }, data.metadata), {
            [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

package base

const OpenAPITemplate = `
{
    "openapi": "3.0.0",
    "info": {
        "version": "1.0.0",
        "title": "Connector OpenAPI"
    },
    "paths": {
        "/execute": {
            "post": {
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": {
                                "type": "object",
                                "properties": {
                                    "inputs": {
                                        "type": "array",
                                        "items": {}
                                    }
                                }
                            }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "type": "object",
                                    "properties": {
                                        "outputs": {
                                            "type": "array",
                                            "items": {}
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
`

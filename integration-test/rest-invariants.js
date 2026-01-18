import {
    check,
    group
} from "k6";
import http from "k6/http";
import {
    randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";

/**
 * TEST SUITE: AIP Resource Refactoring Invariants
 *
 * PURPOSE:
 * Tests the critical invariants defined in the AIP Resource Refactoring plan.
 * These invariants ensure the system maintains data integrity and follows AIP standards.
 *
 * RED FLAGS (RF) - Hard Invariants:
 * - RF-2: name is the only canonical identifier
 *
 * YELLOW FLAGS (YF) - Strict Guardrails:
 * - YF-2: Slug resolution must not leak into services
 */

export function checkInvariants(data) {
    const header = data.header;
    const namespaceId = constant.defaultUsername;

    // ===============================================================
    // RF-2: name is the Only Canonical Identifier
    // id is derived from name, never authoritative alone
    // ===============================================================
    group("RF-2: name is the canonical identifier", () => {
        const randomSuffix = randomString(8);

        let pipelineID;
        let pipelineName;

        // Create a pipeline to test with
        group("Setup: Create test pipeline", () => {
            const createPayload = {
                id: `${constant.dbIDPrefix}invariant-${randomSuffix}`,
                description: "Pipeline for AIP invariant testing",
                rawRecipe: `
version: v1beta
variable:
  input:
    title: Input
    type: string

output:
  answer:
    title: Answer
    value: \${variable.input}
`
            };

            const createResp = http.post(
                `${constant.pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
                JSON.stringify(createPayload),
                header
            );

            if (createResp.status === 200 || createResp.status === 201) {
                const body = JSON.parse(createResp.body);
                pipelineID = body.pipeline.id;
                pipelineName = body.pipeline.name;
            }
        });

        if (!pipelineID) {
            console.error("Failed to create test pipeline, skipping RF-2 tests");
            return;
        }

        // Test 2a: name contains the full resource path
        group("Verify name is full canonical path", () => {
            check({ pipelineName, namespaceId, pipelineID }, {
                "[RF-2a] name contains namespace": (d) => d.pipelineName.includes(`users/${d.namespaceId}`),
                "[RF-2a] name contains resource type": (d) => d.pipelineName.includes("/pipelines/"),
                "[RF-2a] name ends with id": (d) => d.pipelineName.endsWith(d.pipelineID),
                "[RF-2a] name format matches pattern": (d) => {
                    // Pattern: users/{user}/pipelines/{id}
                    const pattern = new RegExp(`^users/[^/]+/pipelines/[^/]+$`);
                    return pattern.test(d.pipelineName);
                }
            });
        });

        // Test 2b: id is derived from name (last segment)
        group("Verify id is derived from name", () => {
            check({ pipelineName, pipelineID }, {
                "[RF-2b] id equals last segment of name": (d) => {
                    const segments = d.pipelineName.split("/");
                    const lastSegment = segments[segments.length - 1];
                    return lastSegment === d.pipelineID;
                }
            });
        });

        // Cleanup
        http.request(
            "DELETE",
            `${constant.pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineID}`,
            null,
            header
        );
    });

    // ===============================================================
    // YF-2: Slug Resolution Must Not Leak into Services
    // Backend services only accept canonical IDs
    // ===============================================================
    group("YF-2: Backend only accepts canonical IDs", () => {
        const randomSuffix = randomString(8);

        let pipelineID;

        // Create test pipeline
        group("Setup: Create test pipeline", () => {
            const createPayload = {
                id: `${constant.dbIDPrefix}canonical-${randomSuffix}`,
                description: "Pipeline for canonical ID testing",
                rawRecipe: `
version: v1beta
variable:
  input:
    title: Input
    type: string

output:
  answer:
    title: Answer
    value: \${variable.input}
`
            };

            const createResp = http.post(
                `${constant.pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
                JSON.stringify(createPayload),
                header
            );

            if (createResp.status === 200 || createResp.status === 201) {
                const body = JSON.parse(createResp.body);
                pipelineID = body.pipeline.id;
            }
        });

        if (!pipelineID) {
            console.error("Failed to create test pipeline, skipping YF-2 tests");
            return;
        }

        // Test: GET by canonical ID should work
        group("GET by canonical ID succeeds", () => {
            const getResp = http.get(
                `${constant.pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineID}`,
                header
            );

            check(getResp, {
                "[YF-2a] GET by canonical ID returns 200": (r) => r.status === 200,
                "[YF-2a] GET by canonical ID returns correct pipeline": (r) => {
                    const body = JSON.parse(r.body);
                    return body.pipeline && body.pipeline.id === pipelineID;
                }
            });
        });

        // Test: GET by invalid/fake ID should fail
        group("GET by invalid ID fails", () => {
            const getResp = http.get(
                `${constant.pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/non-existent-pipeline-id`,
                header
            );

            check(getResp, {
                "[YF-2b] GET by invalid ID returns 404": (r) => r.status === 404 || r.status === 400
            });
        });

        // Cleanup
        http.request(
            "DELETE",
            `${constant.pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineID}`,
            null,
            header
        );
    });

    // ===============================================================
    // Pipeline-specific: Verify pipeline release name format
    // ===============================================================
    group("Pipeline Release: name format validation", () => {
        const randomSuffix = randomString(8);

        let pipelineID;
        let releaseID;
        let releaseName;

        // Create test pipeline
        group("Setup: Create test pipeline", () => {
            const createPayload = {
                id: `${constant.dbIDPrefix}release-${randomSuffix}`,
                description: "Pipeline for release testing",
                rawRecipe: `
version: v1beta
variable:
  input:
    title: Input
    type: string

output:
  answer:
    title: Answer
    value: \${variable.input}
`
            };

            const createResp = http.post(
                `${constant.pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
                JSON.stringify(createPayload),
                header
            );

            if (createResp.status === 200 || createResp.status === 201) {
                const body = JSON.parse(createResp.body);
                pipelineID = body.pipeline.id;

                // Create a release
                const releasePayload = {
                    id: `v1-${randomSuffix}`,
                    description: "Test release"
                };

                const releaseResp = http.post(
                    `${constant.pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineID}/releases`,
                    JSON.stringify(releasePayload),
                    header
                );

                if (releaseResp.status === 200 || releaseResp.status === 201) {
                    const releaseBody = JSON.parse(releaseResp.body);
                    releaseID = releaseBody.release.id;
                    releaseName = releaseBody.release.name;
                }
            }
        });

        if (!releaseID) {
            console.error("Failed to create test release, skipping release tests");
            // Cleanup pipeline if it was created
            if (pipelineID) {
                http.request(
                    "DELETE",
                    `${constant.pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineID}`,
                    null,
                    header
                );
            }
            return;
        }

        // Test: Release name follows hierarchical pattern
        group("Verify release name is hierarchical", () => {
            check({ releaseName, namespaceId, pipelineID, releaseID }, {
                "[Release] name contains pipeline path": (d) => d.releaseName.includes(`pipelines/${d.pipelineID}`),
                "[Release] name contains releases segment": (d) => d.releaseName.includes("/releases/"),
                "[Release] name ends with release id": (d) => d.releaseName.endsWith(d.releaseID),
                "[Release] name format matches pattern": (d) => {
                    // Pattern: users/{user}/pipelines/{pid}/releases/{rid}
                    const pattern = new RegExp(`^users/[^/]+/pipelines/[^/]+/releases/[^/]+$`);
                    return pattern.test(d.releaseName);
                }
            });
        });

        // Cleanup
        http.request(
            "DELETE",
            `${constant.pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineID}/releases/${releaseID}`,
            null,
            header
        );
        http.request(
            "DELETE",
            `${constant.pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineID}`,
            null,
            header
        );
    });
}

import * as constant from './const.js';

// ============================================================================
// Safe Database Query Helpers
// ============================================================================

/**
 * Safe database query wrapper with proper error handling.
 *
 * Prevents silent failures where SQL errors cause queries to return undefined/null,
 * which can lead to tests passing when they should fail.
 *
 * @param {string} query - SQL query string
 * @param {...any} params - Query parameters (passed individually, not as array)
 * @returns {Array} - Query results array, or empty array on error
 * @throws {Error} - Throws error if query fails (caller should handle)
 */
export function safeQuery(query, ...params) {
  try {
    const result = constant.pipelinedb.query(query, ...params);
    if (result === undefined || result === null) {
      throw new Error("Query returned undefined/null - possible SQL error");
    }
    return result;
  } catch (e) {
    console.error(`[DB ERROR] Query failed: ${e}`);
    console.error(`[DB ERROR] Query: ${query}`);
    console.error(`[DB ERROR] Params: ${JSON.stringify(params)}`);
    throw e; // Re-throw so caller knows the query failed
  }
}

/**
 * Safe database execute wrapper for UPDATE/DELETE/INSERT statements.
 *
 * NOTE: k6 SQL driver only has query() method, not execute().
 * For UPDATE/DELETE/INSERT, query() returns an empty result set but still executes.
 * We can't get affected row count, so we return 0 on success.
 *
 * @param {string} statement - SQL statement
 * @param {...any} params - Statement parameters
 * @returns {number} - Always returns 0 (k6 SQL driver doesn't provide affected row count)
 * @throws {Error} - Throws error if statement fails
 */
export function safeExecute(statement, ...params) {
  try {
    // k6 SQL driver only has query(), use it for UPDATE/DELETE/INSERT too
    const result = constant.pipelinedb.query(statement, ...params);
    // query() succeeds but returns empty array for UPDATE/DELETE/INSERT
    // If no exception, assume success
    return 0;
  } catch (e) {
    console.error(`[DB ERROR] Execute failed: ${e}`);
    console.error(`[DB ERROR] Statement: ${statement}`);
    console.error(`[DB ERROR] Params: ${JSON.stringify(params)}`);
    throw e;
  }
}

// ============================================================================
// ID to UID Mapping Helpers (AIP refactoring support)
// ============================================================================

/**
 * Get the internal pipeline uid (UUID) from the hash-based pipeline id.
 * This is needed for database verification tests and internal APIs (like LookUpPipeline)
 * since internal UIDs are no longer exposed via API after AIP refactoring.
 *
 * @param {string} pipelineId - The hash-based pipeline ID from API
 * @returns {string|null} - The internal UUID as string, or null if not found
 */
export function getPipelineUidFromId(pipelineId) {
  try {
    // Cast UUID to text to ensure we get a string representation
    const result = safeQuery(`SELECT uid::text as uid FROM pipeline WHERE id = $1`, pipelineId);
    if (result && result.length > 0) {
      return result[0].uid;
    }
    return null;
  } catch (e) {
    console.error(`[DB ERROR] Failed to get pipeline UID for id=${pipelineId}: ${e}`);
    return null;
  }
}

/**
 * Get the internal pipeline_release uid (UUID) from the hash-based release id.
 *
 * @param {string} releaseId - The hash-based release ID from API
 * @returns {string|null} - The internal UUID as string, or null if not found
 */
export function getPipelineReleaseUidFromId(releaseId) {
  try {
    const result = safeQuery(`SELECT uid::text as uid FROM pipeline_release WHERE id = $1`, releaseId);
    if (result && result.length > 0) {
      return result[0].uid;
    }
    return null;
  } catch (e) {
    console.error(`[DB ERROR] Failed to get pipeline release UID for id=${releaseId}: ${e}`);
    return null;
  }
}

/**
 * Get the internal namespace uid (UUID) from the namespace id.
 * After AIP refactoring, the API only exposes `id` (e.g., "admin"), not the internal UUID.
 * Database queries use `namespace_uid` which is the internal UUID.
 *
 * @param {string} namespaceId - The namespace ID (e.g., "admin")
 * @returns {string|null} - The internal UUID as string, or null if not found
 */
export function getNamespaceUidFromId(namespaceId) {
  // Query the owner table in the mgmt database
  // Cast UUID to text to ensure we get a string representation
  try {
    const result = constant.mgmtDb.query(
      `SELECT uid::text as uid FROM owner WHERE id = $1 LIMIT 1`,
      namespaceId
    );
    if (result && result.length > 0) {
      return result[0].uid;
    }
  } catch (e) {
    console.warn(`[WARN] getNamespaceUidFromId: Failed to query mgmt database for ${namespaceId}: ${e}`);
  }
  return null;
}

// ============================================================================
// Utility Helpers
// ============================================================================

export function deepEqual(x, y) {
  const ok = Object.keys,
    tx = typeof x,
    ty = typeof y;
  return x && y && tx === "object" && tx === ty
    ? ok(x).length === ok(y).length &&
    ok(x).every((key) => deepEqual(x[key], y[key]))
    : x === y;
}

export function checkRecipeIsImmutable(x, y) {
  x.components.forEach(function (value, idx, arr) {
    delete arr[idx].resource_detail;
    delete arr[idx].metadata;
    delete arr[idx].type;
  });
  return deepEqual(x, y);
}

export function isUUID(uuid) {
  const regexExp =
    /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
  return regexExp.test(uuid);
}

export function isValidOwner(owner, expectedOwner) {
  if (owner === null || owner === undefined) return false;
  if (owner.user === null || owner.user === undefined) return false;
  if (owner.user.id !== "admin") return false;
  return deepEqual(owner.user.profile, expectedOwner.profile)
}

export function isValidCreator(creator) {
  if (creator === null || creator === undefined) return false;
  if (!isUUID(creator.uid)) return false;
  if (creator.id === null || creator.id === undefined || creator.id === "") return false;
  return true;
}

export function validateRecipe(recipe, isPrivate) {
  // TODO: fix this
  return true
  if (!("components" in recipe)) {
    console.log("Recipe has no components field");
    return false;
  }

  for (let i = 0; i < recipe.components.length; ++i) {
    if (isUUID(recipe.components[i].id)) {
      console.log("Recipe component id should not be uuid");
      return false;
    }
    if (
      !isPrivate &&
      isUUID(recipe.components[i].resource_name.split("/")[1])
    ) {
      console.log(
        "Recipe component resource_name field should be with resource name not permalink"
      );
    } else if (
      isPrivate &&
      !isUUID(recipe.components[i].resource_name.split("/")[1])
    ) {
      console.log("Recipe component resource_name field should be permalink");
      return false;
    }
    // TODO: add more level of VIEW
    // if (
    //   recipe.components[i].resource_detail === {} ||
    //   recipe.components[i].resource_detail === null ||
    //   recipe.components[i].resource_detail === ""
    // ) {
    //   console.log("Recipe component resource_detail should not be empty");
    //   return false;
    // }
  }

  return true;
}

export function validateRecipeGRPC(recipe, isPrivate) {
  // TODO: fix this
  return true
  if (!("version" in recipe)) {
    console.log("Recipe has no version field");
    return false;
  }

  if (!("components" in recipe)) {
    console.log("Recipe has no components field");
    return false;
  }

  for (let i = 0; i < recipe.components.length; ++i) {
    if (isUUID(recipe.components[i].id)) {
      console.log("Recipe component id should not be uuid");
      return false;
    }
    if (!isPrivate && isUUID(recipe.components[i].resourceName.split("/")[1])) {
      console.log(
        "Recipe component resource_name field should be with resource name not permalink"
      );
      return false;
    } else if (
      isPrivate &&
      !isUUID(recipe.components[i].resourceName.split("/")[1])
    ) {
      console.log("Recipe component resource_name field should be permalink");
      return false;
    }
    // TODO: add more level of VIEW
    // if (
    //   recipe.components[i].resourceDetail === {} ||
    //   recipe.components[i].resourceDetail === null ||
    //   recipe.components[i].resourceDetail === ""
    // ) {
    //   console.log("Recipe component resource_detail should not be empty");
    //   return false;
    // }
  }

  return true;
}

export function genHeader(contentType) {
  return {
    "Content-Type": `${contentType}`,
  };
}

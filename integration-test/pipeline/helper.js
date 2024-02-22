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

export function isValidOwnerHTTP(owner) {
  const expectedProfile = {
    "display_name": "Instill",
    "bio": "",
    "avatar": "",
    "public_email": "",
    "company_name": "Instill AI",
    "social_profile_links": {}
  }
  if (owner === null || owner === undefined) return false;
  if (owner.user === null || owner.user === undefined) return false;
  if (owner.user.id !== "admin") return false;
  return deepEqual(owner.user.profile, expectedProfile)
}

export function isValidOwnerGRPC(owner) {
  const expectedProfile = {
    "displayName": "Instill",
    "bio": "",
    "avatar": "",
    "publicEmail": "",
    "companyName": "Instill AI",
    "socialProfileLinks": {}
  }
  if (owner === null || owner === undefined) return false;
  if (owner.user === null || owner.user === undefined) return false;
  if (owner.user.id !== "admin") return false;
  return deepEqual(owner.user.profile, expectedProfile)
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

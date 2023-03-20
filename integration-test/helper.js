export function deepEqual(x, y) {
  const ok = Object.keys, tx = typeof x, ty = typeof y;
  return x && y && tx === 'object' && tx === ty ? (
    ok(x).length === ok(y).length &&
    ok(x).every(key => deepEqual(x[key], y[key]))
  ) : (x === y);
}

export function isUUID(uuid) {
  const regexExp = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
  return regexExp.test(uuid)
}

export function validateRecipe(recipe) {
  if (!('source' in recipe)) {
    console.log("Recipe has no source field")
    return false
  }

  if (!('model_instances' in recipe)) {
    console.log("Recipe has no model_instances field")
    return false
  }

  if (!('destination' in recipe)) {
    console.log("Recipe has no destination field")
    return false
  }

  if (isUUID(recipe.source.split('/')[1])) {
    console.log("Recipe source field should be with resource name not permalink")
    return false
  }

  for (const modelInstance of recipe.model_instances) {
    if (isUUID(modelInstance.split('/')[1])) {
      console.log("Recipe model_instance field should be with resource name not permalink")
      return false
    }
  }

  if (isUUID(recipe.destination.split('/')[1])) {
    console.log("Recipe destination field should be with resource name not permalink")
    return false
  }

  return true
}

export function validateRecipeGRPC(recipe) {
  if (!('source' in recipe)) {
    console.log("Recipe has no source field")
    return false
  }

  if (!('modelInstances' in recipe)) {
    console.log("Recipe has no model_instances field")
    return false
  }

  if (!('destination' in recipe)) {
    console.log("Recipe has no destination field")
    return false
  }

  if (isUUID(recipe.source.split('/')[1])) {
    console.log("Recipe source field should be with resource name not permalink")
    return false
  }

  for (const modelInstance of recipe.modelInstances) {
    if (isUUID(modelInstance.split('/')[1])) {
      console.log("Recipe model_instance field should be with resource name not permalink")
      return false
    }
  }

  if (isUUID(recipe.destination.split('/')[1])) {
    console.log("Recipe destination field should be with resource name not permalink")
    return false
  }

  return true
}

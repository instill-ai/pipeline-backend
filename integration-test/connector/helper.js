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


export function isValidOwner(owner, expectedOwner) {
    if (owner === null || owner === undefined) return false;
    if (owner.user === null || owner.user === undefined) return false;
    if (owner.user.id !== expectedOwner.id) return false;
    return deepEqual(owner.user.profile, expectedOwner.profile)
  }

export function genHeader(contentType) {
    return {
      "Content-Type": `${contentType}`,
    };
  }

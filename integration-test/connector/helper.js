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

export function genHeader(contentType) {
    return {
      "Content-Type": `${contentType}`,
    };
  }

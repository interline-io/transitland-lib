# Admin REST API Migration Guide

The admin REST API response shapes have changed as part of the authorization system refactoring. All endpoints use the same URL paths and HTTP methods. Request bodies are unchanged. Only response JSON shapes differ.

## Unchanged endpoints

These endpoints have no response changes:

| Endpoint | Description |
|----------|-------------|
| `POST /tenants/:id` | Update tenant name |
| `POST /tenants/:id/groups` | Create group under tenant |
| `POST /groups/:id` | Update group name |
| `POST /tenants/:id/permissions` | Add permission to tenant |
| `DELETE /tenants/:id/permissions` | Remove permission from tenant |
| `POST /groups/:id/permissions` | Add permission to group |
| `DELETE /groups/:id/permissions` | Remove permission from group |
| `POST /groups/:id/tenant` | Set group's parent tenant |
| `POST /feeds/:id/group` | Set feed's parent group |
| `POST /feed_versions/:id/permissions` | Add permission to feed version |
| `DELETE /feed_versions/:id/permissions` | Remove permission from feed version |
| `GET /users` | List users |
| `GET /users/:id` | Get user |

## Changed endpoints

### `GET /me`

```jsonc
// Before
{
  "user": {"id": "ian", "name": "Ian", "email": "ian@example.com"},
  "groups": [{"id": 1, "name": "BA-group"}],
  "expanded_groups": [{"id": 1, "name": "BA-group"}, {"id": 2, "name": "CT-group"}],
  "external_data": {"gatekeeper": "..."},
  "roles": ["admin"]
}

// After
{
  "id": "ian",
  "name": "Ian",
  "email": "ian@example.com",
  "groups": [{"id": 1, "name": "BA-group"}],
  "expanded_groups": [{"id": 1, "name": "BA-group"}, {"id": 2, "name": "CT-group"}],
  "external_data": {"gatekeeper": "..."},
  "roles": ["admin"]
}
```

**Changes:** User fields (`id`, `name`, `email`) are at the top level instead of nested under `user`.

### `GET /tenants`, `GET /groups`, `GET /feeds`, `GET /feed_versions`

All list endpoints now return a flat array of `ObjectRef` instead of a type-specific wrapper.

```jsonc
// Before (GET /tenants)
{
  "tenants": [
    {"id": 1, "name": "tl-tenant"},
    {"id": 2, "name": "other-tenant"}
  ]
}

// After (GET /tenants)
[
  {"type": "tenant", "id": 1},
  {"type": "tenant", "id": 2}
]
```

```jsonc
// Before (GET /feeds)
{
  "feeds": [
    {"id": 5, "onestop_id": "BA", "name": "BART"},
    {"id": 6, "onestop_id": "CT", "name": "Caltrain"}
  ]
}

// After (GET /feeds)
[
  {"type": "feed", "id": 5},
  {"type": "feed", "id": 6}
]
```

**Changes:**
- Response is a JSON array, not an object with a type-specific key
- Each item is `{"type": "...", "id": N}` instead of a full entity object
- Entity names and other fields (e.g., `onestop_id`) are no longer included in list responses
- Use the permissions endpoint (`GET /:type/:id`) to get full details for a specific entity

### `GET /tenants/:id`, `GET /groups/:id`, `GET /feeds/:id`, `GET /feed_versions/:id`

All permissions endpoints now return a generic `ObjectPermissions` shape instead of type-specific responses.

```jsonc
// Before (GET /tenants/1)
{
  "tenant": {"id": 1, "name": "tl-tenant"},
  "groups": [{"id": 2, "name": "BA-group"}],
  "actions": {
    "can_view": true,
    "can_edit": true,
    "can_edit_members": true,
    "can_create_org": true,
    "can_delete_org": true
  },
  "users": {
    "admins": [{"type": "user", "id": "ian", "name": "Ian", "relation": "admin"}],
    "members": [{"type": "user", "id": "drew", "name": "Drew", "relation": "member"}]
  }
}

// After (GET /tenants/1)
{
  "ref": {"type": "tenant", "id": 1, "name": "tl-tenant"},
  "actions": {
    "can_view": true,
    "can_edit": true,
    "can_edit_members": true,
    "can_create_org": true,
    "can_delete_org": true
  },
  "subjects": [
    {"subject": {"type": "user", "name": "ian"}, "relation": "admin", "name": "Ian"},
    {"subject": {"type": "user", "name": "drew"}, "relation": "member", "name": "Drew"}
  ],
  "children": [
    {"type": "org", "id": 2, "name": "BA-group"}
  ]
}
```

```jsonc
// Before (GET /groups/2)
{
  "group": {"id": 2, "name": "BA-group"},
  "tenant": {"id": 1, "name": "tl-tenant"},
  "feeds": [{"id": 5, "onestop_id": "BA", "name": "BART"}],
  "actions": {"can_view": true, ...},
  "users": {
    "managers": [...],
    "editors": [...],
    "viewers": [...]
  }
}

// After (GET /groups/2)
{
  "ref": {"type": "org", "id": 2, "name": "BA-group"},
  "actions": {"can_view": true, ...},
  "subjects": [
    {"subject": {"type": "user", "name": "ian"}, "relation": "viewer", "name": "Ian"}
  ],
  "parent": {"type": "tenant", "id": 1, "name": "tl-tenant"},
  "children": [
    {"type": "feed", "id": 5, "name": "BART"}
  ]
}
```

**Changes:**

| Before | After |
|--------|-------|
| Entity at top level (`tenant`, `group`, `feed`, `feed_version`) | `ref` with `{type, id, name}` |
| Parent as type-specific field (`tenant` on groups, `group` on feeds) | `parent` with `{type, id, name}` |
| Children as type-specific field (`groups` on tenants, `feeds` on groups) | `children` array of `{type, id, name}` |
| Users grouped by role (`admins`, `members`, `managers`, `editors`, `viewers`) | `subjects` flat array with `.relation` field |
| Actions include `false` values via `omitempty` (absent = false) | Actions only include `true` values (absent = false) |
| Feed-specific fields in children (`onestop_id`) | Only `name` available on children |

### Key differences summary

1. **Generic shape** — all entity types return the same JSON structure
2. **`subjects` replaces role-grouped users** — filter by `.relation` client-side instead of accessing `.users.admins`, `.users.editors`, etc.
3. **`parent`/`children` replace type-specific fields** — `parent` is always a single `ObjectRef`, `children` is always an array
4. **Actions only include granted permissions** — denied actions are absent from the map, not present as `false`
5. **List endpoints return bare `ObjectRef` arrays** — no entity details; use the detail endpoint for names

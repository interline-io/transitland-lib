# Admin REST API Migration Guide

The authorization system internals have been refactored, but the admin REST API response shapes are preserved. All endpoints use the same URL paths, HTTP methods, request bodies, and response JSON shapes as before.

Request bodies use integer enum values for `type` and `relation` fields (e.g., `"type": 5` for user, `"relation": 1` for admin), matching the previous proto3 JSON format.

## Unchanged endpoints

All endpoints retain their existing response shapes:

| Endpoint | Description |
|----------|-------------|
| `GET /me` | Current user info |
| `GET /tenants` | List tenants |
| `GET /tenants/:id` | Tenant permissions |
| `POST /tenants/:id` | Update tenant name |
| `POST /tenants/:id/groups` | Create group under tenant |
| `POST /tenants/:id/permissions` | Add permission to tenant |
| `DELETE /tenants/:id/permissions` | Remove permission from tenant |
| `GET /groups` | List groups |
| `GET /groups/:id` | Group permissions |
| `POST /groups/:id` | Update group name |
| `POST /groups/:id/permissions` | Add permission to group |
| `DELETE /groups/:id/permissions` | Remove permission from group |
| `POST /groups/:id/tenant` | Set group's parent tenant |
| `GET /feeds` | List feeds |
| `GET /feeds/:id` | Feed permissions |
| `POST /feeds/:id/group` | Set feed's parent group |
| `GET /feed_versions` | List feed versions |
| `GET /feed_versions/:id` | Feed version permissions |
| `POST /feed_versions/:id/permissions` | Add permission to feed version |
| `DELETE /feed_versions/:id/permissions` | Remove permission from feed version |
| `GET /users` | List users |
| `GET /users/:id` | Get user |

## Minor behavioral changes

- **Actions map**: Only granted permissions (`true`) are included in the `actions` object. Previously, denied actions could appear as `false` due to proto3 default value behavior; now they are simply absent. Clients that check `actions.can_edit === true` are unaffected. Clients that check `"can_edit" in actions` should verify the value is `true`.
- **Entity existence checks**: Permissions endpoints now return "not found" for non-existent entity IDs, even for global admins. Previously, global admins could query permissions on any ID without an existence check.
- **Parse error handling**: Permission mutation endpoints (`POST/DELETE .../permissions`) now correctly return an error and stop processing if the JSON request body is malformed. Previously, a parse failure could fall through and attempt the operation with zero values.

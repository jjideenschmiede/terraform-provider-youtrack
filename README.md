# Terraform Provider for YouTrack

A Terraform provider that manages the configuration of JetBrains YouTrack through its REST API, from notification templates to projects, custom fields and value bundles.

> [!IMPORTANT]
> This is a community-maintained provider, developed by J&J Ideenschmiede GmbH. It is not an official JetBrains product and is not supported by JetBrains. JetBrains and YouTrack are trademarks of JetBrains s.r.o.

---

<div>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".github/images/youtrack-certified-partner-white.svg">
    <img src=".github/images/youtrack-certified-partner-black.svg" alt="YouTrack Certified Partner" align="middle">
  </picture>
</div>

## YouTrack Apps, TeamCity Plugins & more

### J&J Ideenschmiede GmbH - Your Partner for JetBrains Solutions

We are **YouTrack Certified** and **TeamCity Certified** developers, specializing in custom solutions for JetBrains products.

👉🏼 Visit our [website](https://www.jj-ideenschmiede.de) to learn more about our services.

---

## Features

| Resource | Description |
| --- | --- |
| `youtrack_notification_template` | Content of a built-in notification template (FreeMarker) |
| `youtrack_project` | Projects, optionally created from the Scrum or Kanban template |
| `youtrack_custom_field` | Global custom field prototypes |
| `youtrack_enum_bundle` | Enum value sets with order, colors and archived values |
| `youtrack_state_bundle` | State value sets including resolved states |
| `youtrack_project_custom_field` | Attaches a custom field with its bundle to a project |
| `youtrack_user_group` | User groups |
| `youtrack_tag` | Issue tags |
| `youtrack_saved_query` | Saved searches |
| `youtrack_issue_link_type` | Issue link types |

| Data source | Description |
| --- | --- |
| `youtrack_user` | Looks up a user by login |
| `youtrack_user_group` | Looks up a group by name |
| `youtrack_custom_field` | Looks up a custom field by name |

All resources support `terraform import`. The full reference is in [docs](docs/index.md).

## Requirements

- YouTrack 2026.1 or later, the user group endpoints were added in this version
- A permanent token with the permissions for the managed entities, for example *Update Settings* for notification templates and *Low-level Admin Write* for custom fields and link types
- Terraform 1.5 or later

## Usage

```hcl
terraform {
  required_providers {
    youtrack = {
      source = "jjideenschmiede/youtrack"
    }
  }
}

provider "youtrack" {
  url   = "https://example.youtrack.cloud"
  token = var.youtrack_token
}

resource "youtrack_notification_template" "templates" {
  for_each = fileset("${path.module}/templates", "*.ftl")
  id       = trimsuffix(each.value, ".ftl")
  content  = file("${path.module}/templates/${each.value}")
}
```

`url` and `token` can also be set with the environment variables `YOUTRACK_URL` and `YOUTRACK_TOKEN`.

> [!WARNING]
> Destroying `youtrack_project` deletes the project with all of its issues, destroying `youtrack_project_custom_field` removes the field values from all issues of the project, and destroying `youtrack_custom_field` also deletes the bundles attached to it. A `youtrack_user_group` can only be destroyed when `successor_group_id` is set, the successor takes over its permissions and visibility settings. YouTrack has no API to reset notification templates, so destroying `youtrack_notification_template` only removes it from the state.

## Development

> [!NOTE]
> You need Go 1.25 or later and Terraform installed.

```bash
make build
make test
```

`make test` runs Terraform against an in-memory YouTrack mock and needs no running instance.

### Acceptance tests

> [!NOTE]
> The acceptance tests need Docker. Never run them against a production instance, they create, change and delete projects, fields and groups, and they overwrite the `helpdesk_footer` notification template.

```bash
make testacc
make youtrack-down
```

`make testacc` starts YouTrack in the same version as our production instance as a Docker container on port `8089`, skips the configuration wizard, creates a permanent token for `admin` / `admin` and runs every resource through create, update, an empty follow-up plan and import. `make youtrack-down` removes the container, its volumes and the token. Image and port can be changed with `YOUTRACK_IMAGE` and `YOUTRACK_PORT`.

YouTrack creates some entities on its own that remain after the tests: a bundle named `<project>: <field>` whenever an enum field is attached to a project, even if a bundle is set, and the saved search `<project> Backlog` of the Kanban template.

### Manual testing

To try the provider against a real instance, build it and point Terraform at the binary with `dev_overrides` in your `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "jjideenschmiede/youtrack" = "/path/to/your/GOPATH/bin"
  }
  direct {}
}
```

The documentation in `docs` is generated from the schema and the files in `examples` with `make docs`.

The OpenAPI specification of your own instance is available at `https://<your-instance>/api/openapi.json`. Notification templates are not part of it, the provider uses `GET` and `POST /api/admin/notificationTemplates/{id}`.

## License

This provider is licensed under the [Mozilla Public License 2.0](LICENSE).

// Copyright 2026 J&J Ideenschmiede GmbH. All rights reserved.

package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func testAccPreCheck(t *testing.T) {
	for _, name := range []string{"YOUTRACK_URL", "YOUTRACK_TOKEN"} {
		if os.Getenv(name) == "" {
			t.Fatalf("%s must be set for acceptance tests", name)
		}
	}
}

func testAccConfig(suffix string, step int) string {
	values := map[int]struct {
		projectName, archived, template, bundleValues, stateValues, emptyText, tagName, query, linkName string
	}{
		1: {
			projectName:  "Terraform Test",
			archived:     "false",
			template:     "<#macro footer>v1</#macro>",
			bundleValues: `[{ name = "Critical", color_id = "20" }, { name = "Normal" }, { name = "Minor", description = "Can wait" }]`,
			stateValues:  `[{ name = "Open" }, { name = "Done", is_resolved = true }]`,
			emptyText:    "No customer",
			tagName:      "tf-tag",
			query:        "#Unresolved",
			linkName:     "tf-link",
		},
		2: {
			projectName:  "Terraform Test renamed",
			archived:     "true",
			template:     "<#macro footer>v2</#macro>",
			bundleValues: `[{ name = "Blocker", color_id = "1" }, { name = "Normal" }, { name = "Critical", color_id = "20", archived = true }]`,
			stateValues:  `[{ name = "Open" }, { name = "In Review" }, { name = "Done", is_resolved = true }]`,
			emptyText:    "Kein Kunde",
			tagName:      "tf-tag-renamed",
			query:        "#Resolved",
			linkName:     "tf-link-renamed",
		},
	}[step]

	return fmt.Sprintf(`
data "youtrack_user" "admin" {
  login = "admin"
}

data "youtrack_custom_field" "priority" {
  name = "Priority"
}

data "youtrack_user_group" "all_users" {
  name = "All Users"
}

resource "youtrack_user_group" "team" {
  name               = "tf-group-%[1]s-%[2]d"
  successor_group_id = data.youtrack_user_group.all_users.id
}

data "youtrack_user_group" "team" {
  name = youtrack_user_group.team.name
}

resource "youtrack_project" "test" {
  name             = "%[3]s %[1]s"
  short_name       = "TF%[1]s"
  leader_id        = data.youtrack_user.admin.id
  description      = "Created by the acceptance test"
  archived         = %[4]s
  project_template = "kanban"
}

resource "youtrack_custom_field" "customer" {
  name                       = "tf-customer-%[1]s"
  type                       = "enum[1]"
  aliases                    = "kunde"
  is_displayed_in_issue_list = true
}

resource "youtrack_enum_bundle" "customers" {
  name   = "tf-customers-%[1]s"
  values = %[6]s
}

resource "youtrack_state_bundle" "states" {
  name   = "tf-states-%[1]s"
  values = %[7]s
}

resource "youtrack_project_custom_field" "customer" {
  project_id       = youtrack_project.test.id
  field_id         = youtrack_custom_field.customer.id
  bundle_id        = youtrack_enum_bundle.customers.id
  can_be_empty     = true
  empty_field_text = "%[8]s"
}

resource "youtrack_tag" "tag" {
  name                 = "%[9]s-%[1]s"
  color_id             = "5"
  visible_for_group_id = youtrack_user_group.team.id
}

resource "youtrack_saved_query" "query" {
  name                 = "tf-query-%[1]s"
  query                = "%[10]s"
  visible_for_group_id = youtrack_user_group.team.id
}

resource "youtrack_issue_link_type" "link" {
  name             = "%[11]s-%[1]s"
  source_to_target = "blocks"
  target_to_source = "is blocked by"
  directed         = true
}

resource "youtrack_notification_template" "footer" {
  id      = "helpdesk_footer"
  content = "%[5]s"
}
`, suffix, step, values.projectName, values.archived, values.template, values.bundleValues, values.stateValues, values.emptyText, values.tagName, values.query, values.linkName)
}

func importID(name string, attributes ...string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		rs, ok := state.RootModule().Resources[name]
		if !ok {
			return "", fmt.Errorf("resource %s not found", name)
		}
		parts := make([]string, 0, len(attributes))
		for _, attribute := range attributes {
			parts = append(parts, rs.Primary.Attributes[attribute])
		}
		return strings.Join(parts, "/"), nil
	}
}

func TestAccAllResources(t *testing.T) {
	suffix := strings.ToUpper(acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))

	importStep := func(name string, ignore []string, attributes ...string) resource.TestStep {
		step := resource.TestStep{
			Config:                  testAccConfig(suffix, 2),
			ResourceName:            name,
			ImportState:             true,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: ignore,
		}
		if len(attributes) > 0 {
			step.ImportStateIdFunc = importID(name, attributes...)
		}
		return step
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(suffix, 1),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.youtrack_custom_field.priority", tfjsonpath.New("type"), knownvalue.StringExact("enum[1]")),
					statecheck.ExpectKnownValue("youtrack_project.test", tfjsonpath.New("archived"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("youtrack_enum_bundle.customers", tfjsonpath.New("value_ids"), knownvalue.MapSizeExact(3)),
					statecheck.ExpectKnownValue("youtrack_state_bundle.states", tfjsonpath.New("values").AtSliceIndex(1).AtMapKey("is_resolved"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("youtrack_project_custom_field.customer", tfjsonpath.New("empty_field_text"), knownvalue.StringExact("No customer")),
					statecheck.ExpectKnownValue("youtrack_notification_template.footer", tfjsonpath.New("content"), knownvalue.StringExact("<#macro footer>v1</#macro>")),
				},
			},
			{
				Config: testAccConfig(suffix, 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: testAccConfig(suffix, 2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("youtrack_project.test", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("youtrack_project_custom_field.customer", plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("youtrack_project.test", tfjsonpath.New("archived"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("youtrack_enum_bundle.customers", tfjsonpath.New("values").AtSliceIndex(0).AtMapKey("name"), knownvalue.StringExact("Blocker")),
					statecheck.ExpectKnownValue("youtrack_enum_bundle.customers", tfjsonpath.New("values").AtSliceIndex(2).AtMapKey("archived"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("youtrack_state_bundle.states", tfjsonpath.New("values").AtSliceIndex(1).AtMapKey("name"), knownvalue.StringExact("In Review")),
					statecheck.ExpectKnownValue("youtrack_project_custom_field.customer", tfjsonpath.New("empty_field_text"), knownvalue.StringExact("Kein Kunde")),
					statecheck.ExpectKnownValue("youtrack_saved_query.query", tfjsonpath.New("query"), knownvalue.StringExact("#Resolved")),
				},
			},
			{
				Config: testAccConfig(suffix, 2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			importStep("youtrack_user_group.team", []string{"successor_group_id"}),
			importStep("youtrack_project.test", []string{"project_template"}),
			importStep("youtrack_custom_field.customer", nil),
			importStep("youtrack_enum_bundle.customers", nil),
			importStep("youtrack_state_bundle.states", nil),
			importStep("youtrack_project_custom_field.customer", nil, "project_id", "id"),
			importStep("youtrack_tag.tag", nil),
			importStep("youtrack_saved_query.query", nil),
			importStep("youtrack_issue_link_type.link", nil),
			importStep("youtrack_notification_template.footer", nil),
		},
	})
}

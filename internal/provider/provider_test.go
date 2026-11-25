// Copyright 2026 J&J Ideenschmiede GmbH
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

var protoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"youtrack": providerserver.NewProtocol6WithError(New("test")()),
}

func providerConfig(url string) string {
	return fmt.Sprintf(`
provider "youtrack" {
  url   = %q
  token = "test-token"
}
`, url)
}

func TestNotificationTemplateResource(t *testing.T) {
	server := newFakeYouTrack(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
resource "youtrack_notification_template" "footer" {
  id      = "helpdesk_footer"
  content = "<#macro footer>v1</#macro>"
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("youtrack_notification_template.footer", tfjsonpath.New("content"), knownvalue.StringExact("<#macro footer>v1</#macro>")),
				},
			},
			{
				Config: providerConfig(server.URL) + `
resource "youtrack_notification_template" "footer" {
  id      = "helpdesk_footer"
  content = "<#macro footer>v2</#macro>"
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("youtrack_notification_template.footer", tfjsonpath.New("content"), knownvalue.StringExact("<#macro footer>v2</#macro>")),
				},
			},
			{
				ResourceName:      "youtrack_notification_template.footer",
				ImportState:       true,
				ImportStateId:     "helpdesk_footer",
				ImportStateVerify: true,
			},
		},
	})
}

func TestProjectResource(t *testing.T) {
	server := newFakeYouTrack(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
resource "youtrack_project" "demo" {
  name        = "Demo"
  short_name  = "DEMO"
  leader_id   = "1-1"
  description = "Created by Terraform"
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("youtrack_project.demo", tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("youtrack_project.demo", tfjsonpath.New("archived"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("youtrack_project.demo", tfjsonpath.New("from_email"), knownvalue.StringExact("")),
				},
			},
			{
				Config: providerConfig(server.URL) + `
resource "youtrack_project" "demo" {
  name           = "Demo renamed"
  short_name     = "DEMO"
  leader_id      = "1-1"
  archived       = true
  reply_to_email = "support@example.com"
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("youtrack_project.demo", tfjsonpath.New("name"), knownvalue.StringExact("Demo renamed")),
					statecheck.ExpectKnownValue("youtrack_project.demo", tfjsonpath.New("archived"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("youtrack_project.demo", tfjsonpath.New("description"), knownvalue.Null()),
					statecheck.ExpectKnownValue("youtrack_project.demo", tfjsonpath.New("reply_to_email"), knownvalue.StringExact("support@example.com")),
				},
			},
		},
	})
}

func TestEnumBundleResource(t *testing.T) {
	server := newFakeYouTrack(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
resource "youtrack_enum_bundle" "priority" {
  name = "Priorities"
  values = [
    { name = "Critical", color_id = "20" },
    { name = "Normal" },
    { name = "Minor", description = "Can wait" },
  ]
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("youtrack_enum_bundle.priority", tfjsonpath.New("values").AtSliceIndex(0).AtMapKey("color_id"), knownvalue.StringExact("20")),
					statecheck.ExpectKnownValue("youtrack_enum_bundle.priority", tfjsonpath.New("values").AtSliceIndex(1).AtMapKey("color_id"), knownvalue.Null()),
					statecheck.ExpectKnownValue("youtrack_enum_bundle.priority", tfjsonpath.New("value_ids"), knownvalue.MapSizeExact(3)),
				},
			},
			{
				Config: providerConfig(server.URL) + `
resource "youtrack_enum_bundle" "priority" {
  name = "Priorities"
  values = [
    { name = "Blocker" },
    { name = "Normal" },
    { name = "Critical", color_id = "20", archived = true },
  ]
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("youtrack_enum_bundle.priority", tfjsonpath.New("values").AtSliceIndex(0).AtMapKey("name"), knownvalue.StringExact("Blocker")),
					statecheck.ExpectKnownValue("youtrack_enum_bundle.priority", tfjsonpath.New("values").AtSliceIndex(2).AtMapKey("archived"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("youtrack_enum_bundle.priority", tfjsonpath.New("value_ids"), knownvalue.MapSizeExact(3)),
				},
			},
			{
				ResourceName:      "youtrack_enum_bundle.priority",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

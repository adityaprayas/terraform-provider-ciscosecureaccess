// Copyright 2025 Cisco Systems, Inc. and its affiliates
//
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/CiscoDevNet/go-ciscosecureaccess/client"
	"github.com/CiscoDevNet/go-ciscosecureaccess/virtualappliances"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testVirtualApplianceResourceName = "ciscosecureaccess_virtual_appliance.test_resource"

func testVirtualApplianceOriginID(t *testing.T) int64 {
	t.Helper()
	raw := os.Getenv("CISCOSECUREACCESS_TEST_VA_ORIGIN_ID")
	if raw == "" {
		t.Skip("CISCOSECUREACCESS_TEST_VA_ORIGIN_ID not set; skipping virtual appliance tests")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		t.Fatalf("CISCOSECUREACCESS_TEST_VA_ORIGIN_ID must be numeric, got: %s", raw)
	}
	return id
}

func TestVirtualAppliance_import(t *testing.T) {
	rateLimitedTest(t, func() {
		originID := testVirtualApplianceOriginID(t)

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccCiscoSecureAccessProviderFactories,
			CheckDestroy:             testAccCheckVirtualApplianceDestroy,
			Steps: []resource.TestStep{
				{
					Config:            testAccVirtualApplianceConfig(originID),
					ResourceName:      testVirtualApplianceResourceName,
					ImportState:       true,
					ImportStateId:     fmt.Sprintf("%d", originID),
					ImportStateVerify: true,
				},
			},
		})
	}, minWaitTime)
}

func testAccVirtualApplianceConfig(originID int64) string {
	return fmt.Sprintf(`
resource "ciscosecureaccess_virtual_appliance" "test_resource" {
  origin_id = %d
}`, originID)
}

func testAccCheckVirtualApplianceDestroy(s *terraform.State) error {
	ctx := context.Background()
	factory := &client.SSEClientFactory{
		KeyId:     os.Getenv("CISCOSECUREACCESS_KEY_ID"),
		KeySecret: os.Getenv("CISCOSECUREACCESS_KEY_SECRET"),
	}
	c := factory.GetVirtualAppliancesClient(ctx)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ciscosecureaccess_virtual_appliance" {
			continue
		}
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			continue
		}
		_, httpRes, _ := c.VirtualAppliancesAPI.GetVirtualAppliance(ctx, id).Execute()
		if httpRes == nil || httpRes.StatusCode != 404 {
			return fmt.Errorf("virtual appliance %d still exists after destroy", id)
		}
	}
	return nil
}

func TestSetVirtualApplianceState_WithSiteID(t *testing.T) {
	updatedAt := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	appliance := virtualappliances.NewVirtualApplianceObject(12345, "va-test", true, "healthy", "virtual", updatedAt)
	appliance.SetSiteId(101)
	connected := "true"
	stateObj := virtualappliances.VirtualApplianceObjectState{ConnectedToConnector: &connected}
	appliance.SetState(stateObj)

	var model virtualApplianceResourceModel
	setVirtualApplianceState(&model, appliance)

	if model.OriginId.ValueInt64() != 12345 {
		t.Errorf("OriginId: got %d, want 12345", model.OriginId.ValueInt64())
	}
	if model.Name.ValueString() != "va-test" {
		t.Errorf("Name: got %q, want %q", model.Name.ValueString(), "va-test")
	}
	if model.SiteId.IsNull() {
		t.Error("SiteId: expected non-null, got null")
	}
	if model.SiteId.ValueInt64() != 101 {
		t.Errorf("SiteId: got %d, want 101", model.SiteId.ValueInt64())
	}
	if !model.IsUpgradable.ValueBool() {
		t.Error("IsUpgradable: got false, want true")
	}
	if model.Health.ValueString() != "healthy" {
		t.Errorf("Health: got %q, want %q", model.Health.ValueString(), "healthy")
	}
	if model.Type.ValueString() != "virtual" {
		t.Errorf("Type: got %q, want %q", model.Type.ValueString(), "virtual")
	}
	expectedState := fmt.Sprintf("%v", stateObj)
	if model.State.ValueString() != expectedState {
		t.Errorf("State: got %q, want %q", model.State.ValueString(), expectedState)
	}
	expectedUpdatedAt := updatedAt.Format("2006-01-02T15:04:05Z07:00")
	if model.StateUpdatedAt.ValueString() != expectedUpdatedAt {
		t.Errorf("StateUpdatedAt: got %q, want %q", model.StateUpdatedAt.ValueString(), expectedUpdatedAt)
	}
}

func TestSetVirtualApplianceState_WithoutSiteID(t *testing.T) {
	appliance := virtualappliances.NewVirtualApplianceObject(67890, "va-no-site", false, "degraded", "virtual", time.Date(2025, 6, 7, 8, 9, 10, 0, time.UTC))

	var model virtualApplianceResourceModel
	setVirtualApplianceState(&model, appliance)

	if !model.SiteId.IsNull() {
		t.Errorf("SiteId: expected null when SiteId absent, got %d", model.SiteId.ValueInt64())
	}
	if model.OriginId.ValueInt64() != 67890 {
		t.Errorf("OriginId: got %d, want 67890", model.OriginId.ValueInt64())
	}
	if model.Name.ValueString() != "va-no-site" {
		t.Errorf("Name: got %q, want %q", model.Name.ValueString(), "va-no-site")
	}
	if model.IsUpgradable.ValueBool() {
		t.Error("IsUpgradable: got true, want false")
	}
	if model.Health.ValueString() != "degraded" {
		t.Errorf("Health: got %q, want %q", model.Health.ValueString(), "degraded")
	}
	if model.Type.ValueString() != "virtual" {
		t.Errorf("Type: got %q, want %q", model.Type.ValueString(), "virtual")
	}
}

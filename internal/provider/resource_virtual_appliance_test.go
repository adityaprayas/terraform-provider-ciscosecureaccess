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

	"github.com/CiscoDevNet/go-ciscosecureaccess/client"
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

// Copyright 2025 Cisco Systems, Inc. and its affiliates
//
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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

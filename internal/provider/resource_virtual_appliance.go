// Copyright 2025 Cisco Systems, Inc. and its affiliates
//
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/avast/retry-go/v4"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/CiscoDevNet/go-ciscosecureaccess/client"
	"github.com/CiscoDevNet/go-ciscosecureaccess/virtualappliances"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &virtualApplianceResource{}
	_ resource.ResourceWithConfigure   = &virtualApplianceResource{}
	_ resource.ResourceWithImportState = &virtualApplianceResource{}
)

// NewVirtualApplianceResource is a helper function to simplify the provider implementation.
func NewVirtualApplianceResource() resource.Resource {
	return &virtualApplianceResource{}
}

// virtualApplianceResource is the resource implementation.
type virtualApplianceResource struct {
	client virtualappliances.APIClient
}

// virtualApplianceResourceModel maps the resource schema data.
type virtualApplianceResourceModel struct {
	OriginId       types.Int64  `tfsdk:"origin_id"`
	Name           types.String `tfsdk:"name"`
	SiteId         types.Int64  `tfsdk:"site_id"`
	IsUpgradable   types.Bool   `tfsdk:"is_upgradable"`
	Health         types.String `tfsdk:"health"`
	Type           types.String `tfsdk:"type"`
	State          types.String `tfsdk:"state"`
	StateUpdatedAt types.String `tfsdk:"state_updated_at"`
}

// Metadata returns the resource type name.
func (r *virtualApplianceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_appliance"
}

// Configure adds the provider configured client to the resource.
func (r *virtualApplianceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	factory, ok := req.ProviderData.(*client.SSEClientFactory)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.SSEClientFactory, got: %T", req.ProviderData))
		return
	}
	r.client = *factory.GetVirtualAppliancesClient(ctx)
}

// Schema defines the schema for the resource.
func (r *virtualApplianceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             0,
		Description:         "Cisco Secure Access Virtual Appliance resource. Virtual appliances are registered externally and must be imported into Terraform before they can be managed.",
		MarkdownDescription: "Cisco Secure Access Virtual Appliance resource. Virtual appliances are registered externally and must be imported into Terraform before they can be managed.",
		Attributes: map[string]schema.Attribute{
			"origin_id": schema.Int64Attribute{
				Description:         "Origin ID of the virtual appliance.",
				MarkdownDescription: "Origin ID of the virtual appliance.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "Name of the virtual appliance.",
				MarkdownDescription: "Name of the virtual appliance.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": schema.Int64Attribute{
				Description:         "Site ID of the virtual appliance.",
				MarkdownDescription: "Site ID of the virtual appliance.",
				Optional:            true,
				Computed:            true,
			},
			"is_upgradable": schema.BoolAttribute{
				Description:         "Whether the virtual appliance can be upgraded to the latest version.",
				MarkdownDescription: "Whether the virtual appliance can be upgraded to the latest version.",
				Computed:            true,
			},
			"health": schema.StringAttribute{
				Description:         "Health of the virtual appliance.",
				MarkdownDescription: "Health of the virtual appliance.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description:         "Type of the virtual appliance.",
				MarkdownDescription: "Type of the virtual appliance.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"state": schema.StringAttribute{
				Description:         "State of the virtual appliance.",
				MarkdownDescription: "State of the virtual appliance.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"state_updated_at": schema.StringAttribute{
				Description:         "Date and time when the virtual appliance state was updated.",
				MarkdownDescription: "Date and time when the virtual appliance state was updated.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create returns an error because virtual appliances are registered externally.
func (r *virtualApplianceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Creating Virtual Appliance is not supported")
	resp.Diagnostics.AddError(
		"Virtual Appliance Creation Not Supported",
		"Cisco Secure Access virtual appliances are registered externally and cannot be created by Terraform. Import an existing virtual appliance using its origin_id.",
	)
}

// Read refreshes the Terraform state with the latest virtual appliance data.
func (r *virtualApplianceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state virtualApplianceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	originId := state.OriginId.ValueInt64()
	tflog.Debug(ctx, "Reading virtual appliance", map[string]interface{}{"origin_id": originId})

	readResp, httpRes, err := r.client.VirtualAppliancesAPI.GetVirtualAppliance(ctx, originId).Execute()
	if httpRes != nil && httpRes.StatusCode == 404 {
		tflog.Info(ctx, "Virtual appliance not found, removing from state", map[string]interface{}{"origin_id": originId})
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading virtual appliance",
			fmt.Sprintf("Could not read virtual appliance origin_id %d: %s", originId, err.Error()),
		)
		return
	}
	if readResp == nil {
		resp.Diagnostics.AddError(
			"Error reading virtual appliance",
			fmt.Sprintf("Received nil response while reading virtual appliance origin_id %d", originId),
		)
		return
	}

	setVirtualApplianceState(&state, readResp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the virtual appliance and sets the updated Terraform state on success.
func (r *virtualApplianceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating Virtual Appliance")

	var plan virtualApplianceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	originId := plan.OriginId.ValueInt64()
	updateRequest := *virtualappliances.NewUpdateVirtualApplianceRequest(plan.SiteId.ValueInt64())
	var updateResp *virtualappliances.VirtualApplianceObject
	err := retry.Do(
		func() error {
			var httpRes *http.Response
			var err error
			updateResp, httpRes, err = r.client.VirtualAppliancesAPI.UpdateVirtualAppliance(ctx, originId).UpdateVirtualApplianceRequest(updateRequest).Execute()
			if err != nil {
				if httpRes != nil {
					bodyBytes, _ := io.ReadAll(httpRes.Body)
					if httpRes.StatusCode == 409 || httpRes.StatusCode == 429 {
						return fmt.Errorf("retryable error (status %d): %v - %s", httpRes.StatusCode, err, string(bodyBytes))
					}
					resp.Diagnostics.AddError(
						"Error updating virtual appliance",
						fmt.Sprintf("Could not update virtual appliance origin_id %d: %s", originId, err.Error()),
					)
					return retry.Unrecoverable(err)
				}
				resp.Diagnostics.AddError(
					"Error updating virtual appliance",
					fmt.Sprintf("Could not update virtual appliance origin_id %d: %s", originId, err.Error()),
				)
				return retry.Unrecoverable(err)
			}
			return nil
		},
		retry.Attempts(retryMaxAttempts),
		retry.Delay(retryBaseDelay),
		retry.Context(ctx),
	)
	if err != nil {
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.AddError(
				"Error updating virtual appliance",
				fmt.Sprintf("Could not update virtual appliance origin_id %d: %s", originId, err.Error()),
			)
		}
		return
	}
	if updateResp == nil {
		resp.Diagnostics.AddError(
			"Error updating virtual appliance",
			fmt.Sprintf("Received nil response while updating virtual appliance origin_id %d", originId),
		)
		return
	}

	setVirtualApplianceState(&plan, updateResp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the virtual appliance and removes the Terraform state on success.
func (r *virtualApplianceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state virtualApplianceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	originId := state.OriginId.ValueInt64()
	tflog.Info(ctx, "Deleting virtual appliance", map[string]interface{}{"origin_id": originId})

	var httpRes *http.Response
	err := retry.Do(
		func() error {
			var err error
			httpRes, err = r.client.VirtualAppliancesAPI.DeleteVirtualAppliance(ctx, originId).Execute()
			if httpRes != nil && httpRes.StatusCode == 404 {
				return nil
			}
			if err != nil {
				if httpRes != nil {
					bodyBytes, _ := io.ReadAll(httpRes.Body)
					if httpRes.StatusCode == 409 || httpRes.StatusCode == 429 {
						return fmt.Errorf("retryable error (status %d): %v - %s", httpRes.StatusCode, err, string(bodyBytes))
					}
					resp.Diagnostics.AddError(
						"Error deleting virtual appliance",
						fmt.Sprintf("Could not delete virtual appliance origin_id %d: %s", originId, err.Error()),
					)
					return retry.Unrecoverable(err)
				}
				resp.Diagnostics.AddError(
					"Error deleting virtual appliance",
					fmt.Sprintf("Could not delete virtual appliance origin_id %d: %s", originId, err.Error()),
				)
				return retry.Unrecoverable(err)
			}
			return nil
		},
		retry.Attempts(retryMaxAttempts),
		retry.Delay(retryBaseDelay),
		retry.Context(ctx),
	)
	if httpRes != nil && httpRes.StatusCode == 404 {
		tflog.Info(ctx, "Virtual appliance already deleted", map[string]interface{}{"origin_id": originId})
		return
	}
	if err != nil {
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.AddError(
				"Error deleting virtual appliance",
				fmt.Sprintf("Could not delete virtual appliance origin_id %d: %s", originId, err.Error()),
			)
		}
		return
	}

	if httpRes != nil {
		tflog.Info(ctx, "Successfully deleted virtual appliance", map[string]interface{}{
			"origin_id": originId,
			"status":    httpRes.Status,
		})
	}
}

// ImportState imports an existing virtual appliance by origin_id.
func (r *virtualApplianceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	originId, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Virtual Appliance Import ID",
			fmt.Sprintf("Expected a numeric origin_id for import, got %q: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("origin_id"), originId)...)
}

func setVirtualApplianceState(state *virtualApplianceResourceModel, appliance *virtualappliances.VirtualApplianceObject) {
	state.OriginId = types.Int64Value(appliance.GetOriginId())
	state.Name = types.StringValue(appliance.GetName())
	if siteId, ok := appliance.GetSiteIdOk(); ok {
		state.SiteId = types.Int64Value(*siteId)
	} else {
		state.SiteId = types.Int64Null()
	}
	state.IsUpgradable = types.BoolValue(appliance.GetIsUpgradable())
	state.Health = types.StringValue(appliance.GetHealth())
	state.Type = types.StringValue(appliance.GetType())
	state.State = types.StringValue(fmt.Sprintf("%v", appliance.GetState()))
	state.StateUpdatedAt = types.StringValue(appliance.GetStateUpdatedAt().Format("2006-01-02T15:04:05Z07:00"))
}

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/sam"
)

var (
	_ resource.Resource                = &PerimeterResource{}
	_ resource.ResourceWithConfigure   = &PerimeterResource{}
	_ resource.ResourceWithImportState = &PerimeterResource{}
)

func NewPerimeterResource() resource.Resource {
	return &PerimeterResource{}
}

type PerimeterResource struct {
	client *ClientConfig
}

func (r *PerimeterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_perimeter"
}

func (r *PerimeterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Workbench perimeter resource in Sam, controlling access to a VPC-SC perimeter.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Perimeter resource identifier",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "Perimeter resource ID (must be unique across all perimeters)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owners": schema.SetAttribute{
				MarkdownDescription: "Email addresses for the owner policy",
				Required:            true,
				ElementType:         types.StringType,
			},
			"users": schema.SetAttribute{
				MarkdownDescription: "Email addresses for the user policy",
				Required:            true,
				ElementType:         types.StringType,
			},
			"sync_google_group": schema.BoolAttribute{
				MarkdownDescription: "Synchronize the user policy with a Google Group. Once enabled, cannot be disabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"google_group_email": schema.StringAttribute{
				MarkdownDescription: "Google Group email address (populated when sync_google_group is true)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					googleGroupEmailPlanModifier{},
				},
			},
		},
	}
}

type googleGroupEmailPlanModifier struct{}

func (m googleGroupEmailPlanModifier) Description(_ context.Context) string {
	return "Preserves state value when sync_google_group is unchanged."
}

func (m googleGroupEmailPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m googleGroupEmailPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() {
		return
	}

	var syncPlan, syncState types.Bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("sync_google_group"), &syncPlan)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("sync_google_group"), &syncState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if syncPlan.Equal(syncState) {
		resp.PlanValue = req.StateValue
	}
}

func (r *PerimeterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *PerimeterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.PerimeterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := api.NewSamClient(ctx, r.client.Host, r.client.UseIdToken, r.client.ImpersonateServiceAccount)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Sam client: %s", err))
		return
	}

	owners, diags := data.OwnersAsStringSlice(ctx)
	resp.Diagnostics.Append(diags...)
	users, diags := data.UsersAsStringSlice(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceId := data.ResourceId.ValueString()
	if err := api.CreatePerimeter(ctx, c, resourceId, owners, users); err != nil {
		resp.Diagnostics.AddError("Error creating perimeter", fmt.Sprintf("Unable to create perimeter: %s", err))
		return
	}

	// Save state immediately so the resource is tracked even if sync fails.
	// Without this, a sync failure would leave the perimeter orphaned in Sam
	// with no Terraform state, requiring manual import on retry (409).
	state := r.readPerimeterState(ctx, c, resourceId, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)

	if data.SyncGoogleGroup.ValueBool() {
		if err := api.SyncPerimeterGoogleGroup(ctx, c, resourceId); err != nil {
			resp.Diagnostics.AddError("Error syncing perimeter", fmt.Sprintf("Unable to sync perimeter google group: %s", err))
			return
		}
		// Re-read to pick up google_group_email.
		state = r.readPerimeterState(ctx, c, resourceId, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	}
}

func (r *PerimeterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.PerimeterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := api.NewSamClient(ctx, r.client.Host, r.client.UseIdToken, r.client.ImpersonateServiceAccount)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Sam client: %s", err))
		return
	}

	state := r.readPerimeterState(ctx, c, data.ResourceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PerimeterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state models.PerimeterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.SyncGoogleGroup.ValueBool() && !plan.SyncGoogleGroup.ValueBool() {
		resp.Diagnostics.AddError(
			"Sync cannot be disabled",
			"sync_google_group cannot be changed from true to false. Delete and recreate the resource instead.",
		)
		return
	}

	c, err := api.NewSamClient(ctx, r.client.Host, r.client.UseIdToken, r.client.ImpersonateServiceAccount)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Sam client: %s", err))
		return
	}

	resourceId := plan.ResourceId.ValueString()

	if !plan.Owners.Equal(state.Owners) {
		owners, diags := plan.OwnersAsStringSlice(ctx)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := api.OverwritePerimeterPolicy(ctx, c, resourceId, "owner", owners); err != nil {
			resp.Diagnostics.AddError("Error updating owners", fmt.Sprintf("Unable to update owner policy: %s", err))
			return
		}
	}

	if !plan.Users.Equal(state.Users) {
		users, diags := plan.UsersAsStringSlice(ctx)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := api.OverwritePerimeterPolicy(ctx, c, resourceId, "user", users); err != nil {
			resp.Diagnostics.AddError("Error updating users", fmt.Sprintf("Unable to update user policy: %s", err))
			return
		}
	}

	if plan.SyncGoogleGroup.ValueBool() && !state.SyncGoogleGroup.ValueBool() {
		if err := api.SyncPerimeterGoogleGroup(ctx, c, resourceId); err != nil {
			resp.Diagnostics.AddError("Error syncing perimeter", fmt.Sprintf("Unable to sync perimeter google group: %s", err))
			return
		}
	}

	newState := r.readPerimeterState(ctx, c, resourceId, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *PerimeterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.PerimeterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := api.NewSamClient(ctx, r.client.Host, r.client.UseIdToken, r.client.ImpersonateServiceAccount)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Sam client: %s", err))
		return
	}

	if err := api.DeletePerimeter(ctx, c, data.ResourceId.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting perimeter", fmt.Sprintf("Unable to delete perimeter: %s", err))
		return
	}
	tflog.Trace(ctx, "Deleted perimeter")
}

func (r *PerimeterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), req.ID)...)
}

func (r *PerimeterResource) readPerimeterState(ctx context.Context, c *sam.ClientWithResponses, resourceId string, diags *diag.Diagnostics) *models.PerimeterModel {
	policies, err := api.GetPerimeterPolicies(ctx, c, resourceId)
	if err != nil {
		diags.AddError("Error reading perimeter", fmt.Sprintf("Unable to get perimeter policies: %s", err))
		return nil
	}

	syncStatus, err := api.GetPerimeterSyncStatus(ctx, c, resourceId)
	if err != nil {
		diags.AddError("Error reading sync status", fmt.Sprintf("Unable to get perimeter sync status: %s", err))
		return nil
	}

	return models.NewPerimeterModel(resourceId, policies, syncStatus)
}

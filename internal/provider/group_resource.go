package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/schemas"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource              = &GroupResource{}
	_ resource.ResourceWithConfigure = &GroupResource{}
)

// NewGroupResource initializes a new workspace resource.
func NewGroupResource() resource.Resource {
	return &GroupResource{}
}

// GroupResource defines the resource implementation.
type GroupResource struct {
	client *ClientConfig
}

// Metadata returns the resource type name.
func (r *GroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

// Schema defines the resource-level schema for configuration.
func (r *GroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workbench provisioned group resource",

		Attributes: map[string]schema.Attribute{
			"group_name": schemas.GroupResourceSchema,
			"group_email": schema.StringAttribute{
				MarkdownDescription: "Workbench group email",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"internal_name": schema.StringAttribute{
				MarkdownDescription: "Workbench group internal name",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "Workbench group organization ID",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_user_facing_id": schema.StringAttribute{
				MarkdownDescription: "Workbench group organization user facing ID",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Workbench group description",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"expiration_days": schema.Int64Attribute{
				MarkdownDescription: "Workbench group expiration days",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"expiration_notification": schema.BoolAttribute{
				MarkdownDescription: "Workbench group expiration notification",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"require_grant_reason": schema.BoolAttribute{
				MarkdownDescription: "Workbench group require grant reason",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sync_group": schema.BoolAttribute{
				MarkdownDescription: "Workbench group sync group",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"last_updated_date": schema.StringAttribute{
				MarkdownDescription: "Workbench group last updated date",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"last_updated_by": schema.StringAttribute{
				MarkdownDescription: "Workbench group last updated by",
				Computed:            true,
			},
			"created_date": schema.StringAttribute{
				MarkdownDescription: "Workbench group created date",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Workbench group created by",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure initializes the client for the resource with the configuration.
func (r *GroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

// Create creates the group resource.
func (r *GroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve provider data from the configuration.
	var data models.GroupModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	c, err := api.NewUserClient(ctx, r.client.Host, r.client.UseIdToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating user client",
			fmt.Sprintf("Unable to create user client: %s", err),
		)
		return
	}
	if err := api.CreateGroup(ctx, c, data.OrgUfid.ValueString(), data.ToCreateGroupRequest()); err != nil {
		resp.Diagnostics.AddError(
			"Error creating group",
			fmt.Sprintf("Unable to create group: %s", err),
		)
		return
	}

	group, err := api.DescribeGroup(ctx, c, data.GroupName.ValueString(), data.OrgUfid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting group",
			fmt.Sprintf("Unable to get group: %s", err),
		)
		return
	}
	groupState := models.NewGroupModel(group)
	resp.Diagnostics.Append(resp.State.Set(ctx, groupState)...)
}

// Update updates the resource.
func (r *GroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state models.GroupModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.SyncGroup.ValueBool() && !data.SyncGroup.ValueBool() {
		resp.Diagnostics.AddError(
			"Sync group cannot be disabled",
			"Sync group cannot be disabled, please use the delete and create resource instead",
		)
		return
	}

	// Create a new client
	c, err := api.NewUserClient(ctx, r.client.Host, r.client.UseIdToken)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench user manager client, unexpected error: %s", err))
		return
	}

	// Update the group
	if data.Description.ValueString() != state.Description.ValueString() ||
		data.ExpirationDays.ValueInt64() != state.ExpirationDays.ValueInt64() ||
		data.ExpirationNotification.ValueBool() != state.ExpirationNotification.ValueBool() ||
		data.RequireGrantReason.ValueBool() != state.RequireGrantReason.ValueBool() {
		if err := api.UpdateGroup(ctx, c, data.GroupName.ValueString(), data.OrgUfid.ValueString(), data.ToUpdateGroupRequest()); err != nil {
			resp.Diagnostics.AddError(
				"Error updating group",
				fmt.Sprintf("Unable to update group: %s", err),
			)
			return
		}
	}

	// Sync the group
	if data.SyncGroup.ValueBool() && !state.SyncGroup.ValueBool() {
		if err := api.SyncGroup(ctx, c, data.GroupName.ValueString(), data.OrgUfid.ValueString()); err != nil {
			resp.Diagnostics.AddError(
				"Error syncing group",
				fmt.Sprintf("Unable to sync group: %s", err),
			)
			return
		}
	}

	// Describe the group
	g, err := api.DescribeGroup(ctx, c, data.GroupName.ValueString(), data.OrgUfid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting group",
			fmt.Sprintf("Unable to get group: %s", err),
		)
		return
	}
	groupState := models.NewGroupModel(g)
	resp.Diagnostics.Append(resp.State.Set(ctx, groupState)...)
}

// Read reads the group resource.
func (r *GroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.GroupModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Create a new client
	c, err := api.NewUserClient(ctx, r.client.Host, r.client.UseIdToken)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench user manager client, unexpected error: %s", err))
		return
	}
	// Describe the group
	g, err := api.DescribeGroup(ctx, c, data.GroupName.ValueString(), data.OrgUfid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting group",
			fmt.Sprintf("Unable to get group: %s", err),
		)
		return
	}
	groupState := models.NewGroupModel(g)
	resp.Diagnostics.Append(resp.State.Set(ctx, groupState)...)
}

// Delete deletes the resource.
func (r *GroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve provider data from the configuration.
	var data models.GroupModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	c, err := api.NewUserClient(ctx, r.client.Host, r.client.UseIdToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating user client",
			fmt.Sprintf("Unable to create user client: %s", err),
		)
		return
	}
	if err := api.DeleteGroup(ctx, c, data.GroupName.ValueString(), data.OrgUfid.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting group",
			fmt.Sprintf("Unable to delete group: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "Deleted a group")
}

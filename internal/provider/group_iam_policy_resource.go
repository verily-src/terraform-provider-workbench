package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
	"github.com/verily-src/terraform-provider-workbench/internal/schemas"
	"golang.org/x/sync/errgroup"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &GroupIamPolicyResource{}
	_ resource.ResourceWithConfigure   = &GroupIamPolicyResource{}
	_ resource.ResourceWithImportState = &GroupIamPolicyResource{}
)

// NewGroupIamPolicyResource initializes a new group resource.
func NewGroupIamPolicyResource() resource.Resource {
	return &GroupIamPolicyResource{}
}

// GroupIamPolicyResource defines the resource implementation.
type GroupIamPolicyResource struct {
	client *ClientConfig
}

// Metadata returns the resource type name.
func (r *GroupIamPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_iam_policy"
}

// Schema defines the resource-level schema for configuration.
func (r *GroupIamPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workbench provisioned group IAM member resource",

		Attributes: map[string]schema.Attribute{
			"group": schemas.GroupResourceSchema,
			"organization": schema.StringAttribute{
				MarkdownDescription: "Workbench organization ID, either UUID or UFID. If it is a UFID, it must be prefixed with a tilde (~).",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"iams": schema.SetNestedAttribute{
				MarkdownDescription: "Workbench group user IAM policies",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.StringAttribute{
							MarkdownDescription: "Workbench IAM role of the member",
							Required:            true,
							Validators: []validator.String{
								groupIamRoleValidator(),
							},
						},
						"principals": schema.SetNestedAttribute{
							Description: "List of principals (users, groups, or public) in the IAM binding.",
							Required:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"user": schema.StringAttribute{
										Description: "Email of a user.",
										Optional:    true,
									},
									"group": schema.SingleNestedAttribute{
										Description: "Identifier of a group.",
										Optional:    true,
										Attributes: map[string]schema.Attribute{
											"group": schema.StringAttribute{
												Description: "Name of the group.",
												Required:    true,
											},
											"organization": schema.StringAttribute{
												Description: "UxID of the organization. If it is a UFID, it must be prefixed with a tilde (~).",
												Optional:    true,
											},
										},
									},
									"public": schema.BoolAttribute{
										Description: "True if the group is public.",
										Optional:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure initializes the client for the resource with the configuration.
func (r *GroupIamPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource.
func (r *GroupIamPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve provider data from the configuration.
	var data models.GroupIamPolicyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	grantReqs := data.BuildGrantRequests()
	groupNameParam, orgIDParams := data.Params()
	err := r.setGroupRoles(ctx, groupNameParam, orgIDParams, grantReqs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error granting group role",
			fmt.Sprintf("Unable to grant group role: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, (&data).Normalized())...)
}

func (r *GroupIamPolicyResource) setGroupRoles(ctx context.Context, groupName user.GroupNameParam, orgIdParam *user.OrgIdQueryParam, requests []user.SetAccessRequest) error {
	c, err := api.NewUserClient(ctx, r.client.Host)
	if err != nil {
		return fmt.Errorf("unable to create Workbench client, unexpected error: %w", err)
	}
	jsonReqs, _ := json.MarshalIndent(requests, "", "  ")
	tflog.Trace(ctx, "Setting group roles", map[string]any{"roles": string(jsonReqs)})

	g, ctx := errgroup.WithContext(ctx)
	for _, request := range requests {
		g.Go(func() error {
			return api.SetGroupRole(ctx, c, groupName, orgIdParam, request)
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	tflog.Trace(ctx, "Group roles set successfully")
	return nil
}

// Update updates the resource.
func (r *GroupIamPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state models.GroupIamPolicyModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleted, added := models.DiffGroupIamPolicies(&state, &data)
	if deleted == nil && added == nil {
		tflog.Trace(ctx, "No changes detected, skipping update")
	}

	groupNameParam, orgIDParams := data.Params()

	// Revoke the existing role and grant the new one
	var reqs []user.SetAccessRequest
	for _, binding := range deleted {
		reqs = append(reqs, binding.BuildRevokeRequests()...)
	}
	for _, binding := range added {
		reqs = append(reqs, binding.BuildGrantRequests()...)
	}

	err := r.setGroupRoles(ctx, groupNameParam, orgIDParams, reqs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error revoking group roles",
			fmt.Sprintf("Unable to revoke group roles: %s", err),
		)
		return
	}

	tflog.Trace(ctx, "Updated binding")
	resp.Diagnostics.Append(resp.State.Set(ctx, (&data).Normalized())...)
}

// Read reads the resource.
func (r *GroupIamPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.GroupIamPolicyModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new client
	c, err := api.NewUserClient(ctx, r.client.Host)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}

	groupNameParam, orgIDParams := data.Params()
	roles, err := api.GetGroupRoles(ctx, c, groupNameParam, orgIDParams)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading group roles into data, got error: %s", err))
		return
	}

	state := models.NewGroupIamPolicyModel(data.GroupName.ValueString(), data.OrgID.ValueString(), roles)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Delete deletes the resource.
func (r *GroupIamPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.GroupIamPolicyModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	revokeReq := data.BuildRevokeRequests()
	groupNameParam, orgIDParams := data.Params()
	err := r.setGroupRoles(ctx, groupNameParam, orgIDParams, revokeReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error revoking group roles",
			fmt.Sprintf("Unable to revoke group roles: %s", err),
		)
		return
	}
}

var groupPattern = regexp.MustCompile("^(?:organizations/([^/]+)/)?groups/(.+)/roles")

type groupIamPolicyID struct {
	OrgID     string
	GroupName string
}

func parseGroupIamPolicyID(id string) *groupIamPolicyID {
	parts := groupPattern.FindStringSubmatch(id)
	if parts == nil {
		return nil
	}

	return &groupIamPolicyID{
		OrgID:     parts[1],
		GroupName: parts[2],
	}
}

func (r *GroupIamPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	groupIamPolicyID := parseGroupIamPolicyID(req.ID)
	if groupIamPolicyID == nil {
		resp.Diagnostics.AddError(
			"Invalid Group IAM Policy ID",
			fmt.Sprintf("Unable to parse Group IAM Policy ID %q", req.ID),
		)
		return
	}

	if groupIamPolicyID.OrgID != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), groupIamPolicyID.OrgID)...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group"), groupIamPolicyID.GroupName)...)
}

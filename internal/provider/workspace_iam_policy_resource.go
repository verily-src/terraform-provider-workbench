package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
	"github.com/verily-src/terraform-provider-workbench/internal/schemas"
	"golang.org/x/sync/errgroup"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &WorkspaceIamPolicyResource{}
	_ resource.ResourceWithConfigure   = &WorkspaceIamPolicyResource{}
	_ resource.ResourceWithImportState = &WorkspaceIamPolicyResource{}
)

// NewWorkspaceIamPolicyResource initializes a new workspace resource.
func NewWorkspaceIamPolicyResource() resource.Resource {
	return &WorkspaceIamPolicyResource{}
}

// WorkspaceIamPolicyResource defines the resource implementation.
type WorkspaceIamPolicyResource struct {
	client *ClientConfig
}

// Metadata returns the resource type name.
func (r *WorkspaceIamPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_iam_policy"
}

// Schema defines the resource-level schema for configuration.
func (r *WorkspaceIamPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Workbench provisioned workspace IAM policies.

⚠️ **Caution:** You can accidentally lock yourself out of your workspace using this resource! Deleting this IAM policy may result in **loss of access** to the associated workspace. Proceed with caution.`,

		Attributes: map[string]schema.Attribute{
			"workspace_id": schemas.WorkspaceResourceSchema,
			"iams": schema.SetNestedAttribute{
				MarkdownDescription: "Workbench workspace user IAM policies",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.StringAttribute{
							MarkdownDescription: "Workbench IAM role of the member",
							Required:            true,
							Validators: []validator.String{
								workspaceIamRoleValidator(),
							},
						},
						"members": schema.SetAttribute{
							MarkdownDescription: "List of members in the IAM binding",
							Required:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

// Configure initializes the client for the resource with the configuration.
func (r *WorkspaceIamPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *WorkspaceIamPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve provider data from the configuration.
	var data models.WorkspaceIamPolicyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	grantReqs := data.BuildGrantRequests()
	err := r.setRoles(ctx, data.WorkspaceID.ValueString(), grantReqs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error granting workspace role",
			fmt.Sprintf("Unable to grant workspace role: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, (&data).Normalized())...)
}

func (r *WorkspaceIamPolicyResource) setRoles(ctx context.Context, workspaceID string, requests []wsm.SetAccessRequest) error {
	c, err := api.NewWSMClient(ctx, r.client.Host)
	if err != nil {
		return fmt.Errorf("unable to create Workbench client, unexpected error: %w", err)
	}
	jsonReqs, _ := json.MarshalIndent(requests, "", "  ")
	tflog.Trace(ctx, "Setting workspace roles", map[string]any{"roles": string(jsonReqs)})

	g, ctx := errgroup.WithContext(ctx)
	for _, request := range requests {
		g.Go(func() error {
			return api.SetRole(ctx, c, workspaceID, request)
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	tflog.Trace(ctx, "Workspace roles set successfully")
	return nil
}

// Update updates the resource.
func (r *WorkspaceIamPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state models.WorkspaceIamPolicyModel

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

	deleted, added := models.DiffWorkspaceIamPolicies(&state, &data)
	if deleted == nil && added == nil {
		tflog.Trace(ctx, "No changes detected, skipping update")
	}

	workspaceID := data.WorkspaceID.ValueString()
	if workspaceID != state.WorkspaceID.ValueString() {
		// Throw an error if the workspace IDs do not match. They should always
		// match, since modifying the workspace ID should cause a destroy and
		// recreate.
		resp.Diagnostics.AddError(
			"Workspace ID Mismatch",
			"State and data workspace IDs do not match. This is unexpected behavior and should be reported to the provider developers.",
		)
		return
	}

	// Revoke the existing role and grant the new one
	revokeReqs := models.WorkspaceIamBindingsToPolicy(workspaceID, deleted).BuildRevokeRequests()
	grantReqs := models.WorkspaceIamBindingsToPolicy(workspaceID, added).BuildGrantRequests()

	err := r.setRoles(ctx, workspaceID, append(revokeReqs, grantReqs...))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error revoking workspace roles",
			fmt.Sprintf("Unable to revoke workspace roles: %s", err),
		)
		return
	}

	tflog.Trace(ctx, "Updated binding")
	resp.Diagnostics.Append(resp.State.Set(ctx, (&data).Normalized())...)
}

// Read reads the resource.
func (r *WorkspaceIamPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.WorkspaceIamPolicyModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new client
	c, err := api.NewWSMClient(ctx, r.client.Host)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}

	roles, err := api.GetRoles(ctx, c, data.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading workspace roles into data, got error: %s", err))
		return
	}

	state := models.NewWorkspaceIamPolicyModel(data.WorkspaceID.ValueString(), roles)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Delete deletes the resource.
func (r *WorkspaceIamPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.WorkspaceIamPolicyModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	revokeReq := data.BuildRevokeRequests()
	err := r.setRoles(ctx, data.WorkspaceID.ValueString(), revokeReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error revoking workspace roles",
			fmt.Sprintf("Unable to revoke workspace roles: %s", err),
		)
		return
	}
}

var workspaceIamPolicyID = regexp.MustCompile("workspaces/(.+)/roles")

func ParseWorkspaceIamPolicyID(id string) string {
	parts := workspaceIamPolicyID.FindStringSubmatch(id)
	if parts == nil {
		return ""
	}

	return parts[1]
}

func (r *WorkspaceIamPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	workspaceIamPolicyID := ParseWorkspaceIamPolicyID(req.ID)
	if workspaceIamPolicyID == "" {
		resp.Diagnostics.AddError(
			"Invalid WSM IAM Policy ID",
			fmt.Sprintf("Unable to parse WSM IAM Policy ID %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), workspaceIamPolicyID)...)
}

package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource              = &WorkspaceIamMemberResource{}
	_ resource.ResourceWithConfigure = &WorkspaceIamMemberResource{}
)

// NewWorkspaceIamMemberResource initializes a new workspace resource.
func NewWorkspaceIamMemberResource() resource.Resource {
	return &WorkspaceIamMemberResource{
		WorkspaceIamBindingResource: &WorkspaceIamBindingResource{},
	}
}

// WorkspaceIamMemberResource defines the resource implementation.
type WorkspaceIamMemberResource struct {
	*WorkspaceIamBindingResource
}

// Metadata returns the resource type name.
func (r *WorkspaceIamMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_iam_member"
}

// Schema defines the resource-level schema for configuration.
func (r *WorkspaceIamMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	r.WorkspaceIamBindingResource.Schema(ctx, req, resp)
	resp.Schema.MarkdownDescription = "Workbench provisioned workspace IAM member resource"
}

// Read reads the resource.
func (r *WorkspaceIamMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.WorkspaceIamBindingModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the members list is empty, we don't need to check the API if any are
	// still present
	if data.Members == nil || len(*data.Members) == 0 {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// Create a new client
	c, err := api.NewWSMClient(ctx, r.client.Host, r.client.UseIdToken)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}

	roles, err := api.GetRoles(ctx, c, data.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading workspace roles into data, got error: %s", err))
		return
	}

	roleBinding := &wsm.RoleBinding{
		Role:    wsm.IamRole(data.Role.ValueString()),
		Members: nil,
	}

	for _, rb := range *roles {
		if rb.Role != roleBinding.Role {
			continue
		}

		if rb.Members == nil {
			break
		}

		// Include only members from the original data that still hold the
		// role in the workspace, i.e. the intersection of the two member
		// sets.
		members := make([]string, 0, len(*data.Members))
		for _, member := range *rb.Members {
			if slices.ContainsFunc(*data.Members, func(m types.String) bool {
				return m.ValueString() == member
			}) {
				members = append(members, member)
			}
		}
		roleBinding.Members = &members
		break
	}

	state := models.NewWorkspaceIamBindingModel(data.WorkspaceID.ValueString(), roleBinding)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

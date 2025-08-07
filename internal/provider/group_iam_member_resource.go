package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource              = &GroupIamMemberResource{}
	_ resource.ResourceWithConfigure = &GroupIamMemberResource{}
)

// NewGroupIamMemberResource initializes a new group resource.
func NewGroupIamMemberResource() resource.Resource {
	return &GroupIamMemberResource{
		GroupIamBindingResource: &GroupIamBindingResource{},
	}
}

// GroupIamMemberResource defines the resource implementation.
type GroupIamMemberResource struct {
	*GroupIamBindingResource
}

// Metadata returns the resource type name.
func (r *GroupIamMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_iam_member"
}

// Schema defines the resource-level schema for configuration.
func (r *GroupIamMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	r.GroupIamBindingResource.Schema(ctx, req, resp)
	resp.Schema.MarkdownDescription = "Workbench provisioned group IAM member resource"
}

// Read reads the resource.
func (r *GroupIamMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.GroupIamBindingModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the principals list is empty, we don't need to check the API if any are
	// still present
	if len(data.Principals) == 0 {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// Create a new client
	c, err := api.NewUserClient(ctx, r.client.Host)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}

	groupNameParam, orgIDParam := data.Params()
	roles, err := api.GetGroupRoles(ctx, c, groupNameParam, orgIDParam)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading group roles into data, got error: %s", err))
		return
	}

	principals := make([]models.GroupPrincipal, 0, len(data.Principals))

	for _, rb := range *roles {
		if rb.Roles == nil || len(rb.Roles) == 0 {
			// If there are no roles, we can skip this principal
			continue
		}
		if !slices.Contains(rb.Roles, user.GroupRole(data.Role.ValueString())) {
			// If the role does not match the requested role, we can skip this principal
			continue
		}
		p := rb.Principal
		// Include only principals from the original data that still hold the
		// role in the group, i.e. the intersection of the two principal
		// sets.
		if matches := slices.ContainsFunc(data.Principals, func(gp models.GroupPrincipal) bool {
			return models.PrincipalMatchesTf(p, gp)
		}); matches {
			// If we found a matching group principal, we can use it
			principals = append(principals, models.ConvertGroupPrincipalToTf(p))
		}
	}
	data.Principals = principals

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

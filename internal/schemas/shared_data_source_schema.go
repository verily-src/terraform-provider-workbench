package schemas

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// WorkspaceDataSourceSchema defines the schema for the workspace data source.
var WorkspaceDataSourceSchema = schema.StringAttribute{
	MarkdownDescription: "Workspace workspace unique UUID",
	Required:            true,
}

// GroupDataSourceSchema defines the schema for the group data source.
var GroupDataSourceSchema = schema.StringAttribute{
	MarkdownDescription: "Workbench group name",
	Required:            true,
}

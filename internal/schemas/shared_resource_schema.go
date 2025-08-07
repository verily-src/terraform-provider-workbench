package schemas

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// PropertiesResourceSchema defines the schema for properties in a resource.
var PropertiesResourceSchema = schema.SetNestedAttribute{
	MarkdownDescription: "Workbench properties in key-value pair",
	Optional:            true,
	NestedObject: schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				MarkdownDescription: "Key of the property",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Value of the property",
				Required:            true,
			},
		},
	},
}

// WorkspaceResourceSchema defines the schema for the workspace resource.
var WorkspaceResourceSchema = schema.StringAttribute{
	MarkdownDescription: "Workspace workspace unique UUID",
	Required:            true,
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	},
}

// GroupResourceSchema defines the schema for the group resource.
var GroupResourceSchema = schema.StringAttribute{
	MarkdownDescription: "Workbench managed group name",
	Required:            true,
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	},
}

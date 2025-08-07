package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

func workspaceIamRoleValidator() validator.String {
	return stringvalidator.OneOf(
		string(wsm.IamRoleOWNER),
		string(wsm.IamRoleWRITER),
		string(wsm.IamRoleREADER),
		string(wsm.IamRoleDISCOVERER),
	)
}

func groupIamRoleValidator() validator.String {
	return stringvalidator.OneOf(
		string(user.GroupRoleADMIN),
		string(user.GroupRoleSUPPORT),
		string(user.GroupRoleMEMBER),
		string(user.GroupRoleREADER),
	)
}

func storageClassValidator() validator.String {
	return stringvalidator.OneOf(
		string(wsm.ARCHIVE),
		string(wsm.COLDLINE),
		string(wsm.NEARLINE),
		string(wsm.STANDARD),
	)
}

func cloningInstructionValidator() validator.String {
	return stringvalidator.OneOf(
		string(wsm.COPYDEFINITION),
		string(wsm.COPYLINKREFERENCE),
		string(wsm.COPYNOTHING),
		string(wsm.COPYREFERENCE),
		string(wsm.COPYRESOURCE),
	)
}

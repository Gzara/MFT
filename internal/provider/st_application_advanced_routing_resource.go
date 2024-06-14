package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

/*
{
	"name": "string",
	"type": "AdvancedRouting",
	"notes": "string",
	"additionalAttributes": {
	  "additionalProp1": "string",
	  "additionalProp2": "string",
	  "additionalProp3": "string"
	},
	"businessUnits": [
	  "string"
	]
  }
*/

type stAdvancedRoutingApplication struct {
	Id          types.String `tfsdk:"id" helper:",computed,state"`
	Name        types.String `tfsdk:"name" helper:",required"`
	LastUpdated types.String `tfsdk:"last_updated" helper:",computed,noread,nowrite"`

	Type          types.String   `tfsdk:"type" helper:",default:AdvancedRouting"`
	Notes         types.String   `tfsdk:"notes"`
	BusinessUnits []types.String `tfsdk:"business_units" helper:"businessUnits"`

	AdditionalAttributes types.Map `tfsdk:"additional_attributes" helper:"additionalAttributes,elementtype:string,optional"`
}

func NewSTAdvancedRoutingApplicationResource() resource.Resource {
	return NewSTResource(&stAdvancedRoutingApplication{}, "st_advanced_routing_application", "", "/api/v2.0/applications", "/api/v2.0/applications/{name}")
}

func init() {
	registerResource(NewSTAdvancedRoutingApplicationResource)
}

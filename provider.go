package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type gouterProvider struct {
	client *Client
}

type gouterProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

func (p *gouterProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "gouter"
}

func (p *gouterProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Required:    true,
				Description: "Gouter API endpoint, e.g. http://127.0.0.1:18081/api/v1",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Optional Bearer token for API authentication",
			},
		},
	}
}

func (p *gouterProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var model gouterProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if model.Endpoint.IsUnknown() || model.Endpoint.ValueString() == "" {
		resp.Diagnostics.AddError("missing endpoint", "endpoint is required")
		return
	}
	p.client = NewClient(model.Endpoint.ValueString(), model.Token.ValueString())
	resp.ResourceData = p
}

func (p *gouterProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newLinkResource,
		newBGPPeerResource,
		newRouteResource,
	}
}

func (p *gouterProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}

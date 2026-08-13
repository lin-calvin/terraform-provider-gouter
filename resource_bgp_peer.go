package main

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type bgpPeerResource struct {
	client *Client
}

type bgpPeerResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Address     types.String `tfsdk:"address"`
	ASN         types.Int64  `tfsdk:"asn"`
	PeerBGPPort types.Int64  `tfsdk:"peer_bgp_port"`
	Families    types.List   `tfsdk:"families"`
	RRClient    types.Bool   `tfsdk:"rr_client"`
	PassiveMode types.Bool   `tfsdk:"passive_mode"`
}

func newBGPPeerResource() resource.Resource {
	return &bgpPeerResource{}
}

func (r *bgpPeerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bgp_peer"
}

func (r *bgpPeerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name":          schema.StringAttribute{Required: true},
			"address":       schema.StringAttribute{Required: true},
			"asn":           schema.Int64Attribute{Required: true},
			"peer_bgp_port": schema.Int64Attribute{Optional: true},
			"families":      schema.ListAttribute{ElementType: types.StringType, Optional: true},
			"rr_client":     schema.BoolAttribute{Optional: true},
			"passive_mode":  schema.BoolAttribute{Optional: true},
		},
	}
}

func (r *bgpPeerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	p, ok := req.ProviderData.(*gouterProvider)
	if !ok {
		return
	}
	r.client = p.client
}

func (r *bgpPeerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bgpPeerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := buildBGPPeerBody(plan)
	if err := r.client.Post("/bgp/peers", body, nil); err != nil {
		resp.Diagnostics.AddError("create bgp peer", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *bgpPeerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bgpPeerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var result map[string]any
	if err := r.client.Get("/bgp/peers", state.Name.ValueString(), &result); err != nil {
		if errors.Is(err, ErrNotFound) {
			// 资源已不存在：从 state 移除，让 Terraform 决定重建
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read bgp peer", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *bgpPeerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bgpPeerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := buildBGPPeerBody(plan)
	if err := r.client.Post("/bgp/peers", body, nil); err != nil {
		resp.Diagnostics.AddError("update bgp peer", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *bgpPeerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bgpPeerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete("/bgp/peers", state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("delete bgp peer", err.Error())
		return
	}
}

func (r *bgpPeerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func buildBGPPeerBody(plan bgpPeerResourceModel) map[string]any {
	body := map[string]any{
		"name":    plan.Name.ValueString(),
		"address": plan.Address.ValueString(),
		"asn":     plan.ASN.ValueInt64(),
	}
	if !plan.PeerBGPPort.IsUnknown() && plan.PeerBGPPort.ValueInt64() > 0 {
		body["peer_bgp_port"] = plan.PeerBGPPort.ValueInt64()
	}
	if !plan.Families.IsUnknown() && len(plan.Families.Elements()) > 0 {
		var families []string
		for _, e := range plan.Families.Elements() {
			if s, ok := e.(types.String); ok {
				families = append(families, s.ValueString())
			}
		}
		body["families"] = families
	}
	if !plan.RRClient.IsUnknown() {
		body["rr_client"] = plan.RRClient.ValueBool()
	}
	if !plan.PassiveMode.IsUnknown() {
		body["passive_mode"] = plan.PassiveMode.ValueBool()
	}
	return body
}

var _ resource.Resource = &bgpPeerResource{}

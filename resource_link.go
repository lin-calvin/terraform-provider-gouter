package main

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type linkResource struct {
	client *Client
}

type linkResourceModel struct {
	Name      types.String `tfsdk:"name"`
	Address   types.String `tfsdk:"address"`
	PeerIP    types.String `tfsdk:"peer_ip"`
	Tun       types.Bool   `tfsdk:"tun"`
	WGPort    types.Int64  `tfsdk:"wg_listen_port"`
	WGPriv    types.String `tfsdk:"wg_private_key"`
	WGPub     types.String `tfsdk:"wg_public_key"`
	WGMtu     types.Int64  `tfsdk:"wg_mtu"`
	WGEp      types.String `tfsdk:"wg_endpoint"`
	WGAIP     types.String `tfsdk:"wg_allowed_ips"`
	WGKA      types.Int64  `tfsdk:"wg_persistent_keepalive"`
	MPLSPort  types.Int64  `tfsdk:"mpls_listen_port"`
	MPLSPeers types.List   `tfsdk:"mpls_peers"`
}

func newLinkResource() resource.Resource {
	return &linkResource{}
}

func (r *linkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_link"
}

func (r *linkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name":                    schema.StringAttribute{Required: true},
			"address":                 schema.StringAttribute{Optional: true},
			"peer_ip":                 schema.StringAttribute{Optional: true},
			"tun":                     schema.BoolAttribute{Optional: true},
			"wg_listen_port":          schema.Int64Attribute{Optional: true},
			"wg_private_key":          schema.StringAttribute{Optional: true, Sensitive: true},
			"wg_public_key":           schema.StringAttribute{Optional: true},
			"wg_mtu":                  schema.Int64Attribute{Optional: true},
			"wg_endpoint":             schema.StringAttribute{Optional: true},
			"wg_allowed_ips":          schema.StringAttribute{Optional: true},
			"wg_persistent_keepalive": schema.Int64Attribute{Optional: true},
			"mpls_listen_port":        schema.Int64Attribute{Optional: true},
			"mpls_peers":              schema.ListAttribute{ElementType: types.StringType, Optional: true},
		},
	}
}

func (r *linkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	p, ok := req.ProviderData.(*gouterProvider)
	if !ok {
		return
	}
	r.client = p.client
}

func (r *linkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan linkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := buildLinkBody(plan)
	if err := r.client.Post("/links", body, nil); err != nil {
		resp.Diagnostics.AddError("create link", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *linkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state linkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := state.Name.ValueString()
	var result map[string]any
	if err := r.client.Get("/links", name, &result); err != nil {
		if errors.Is(err, ErrNotFound) {
			// 资源已不存在：从 state 移除，让 Terraform 决定重建
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read link", err.Error())
		return
	}
	// state is already populated from Create; GET just validates it exists
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *linkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan linkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := buildLinkBody(plan)
	if err := r.client.Post("/links", body, nil); err != nil {
		resp.Diagnostics.AddError("update link", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *linkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state linkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete("/links", state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("delete link", err.Error())
		return
	}
}

func (r *linkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func buildLinkBody(plan linkResourceModel) map[string]any {
	body := map[string]any{
		"name":    plan.Name.ValueString(),
		"address": plan.Address.ValueString(),
		"peer_ip": plan.PeerIP.ValueString(),
	}
	switch {
	case plan.Tun.ValueBool():
		body["tun"] = map[string]any{"mtu": 1420}
	case !plan.MPLSPort.IsUnknown() && plan.MPLSPort.ValueInt64() > 0:
		mp := map[string]any{
			"listen_port": plan.MPLSPort.ValueInt64(),
		}
		if !plan.MPLSPeers.IsUnknown() && len(plan.MPLSPeers.Elements()) > 0 {
			var peers []string
			for _, e := range plan.MPLSPeers.Elements() {
				if s, ok := e.(types.String); ok {
					peers = append(peers, s.ValueString())
				}
			}
			mp["peers"] = peers
		}
		body["mpls_udp"] = mp
	default:
		wg := make(map[string]any)
		wg["listen_port"] = plan.WGPort.ValueInt64()
		wg["private_key"] = plan.WGPriv.ValueString()
		wg["public_key"] = plan.WGPub.ValueString()
		if !plan.WGEp.IsUnknown() && plan.WGEp.ValueString() != "" {
			wg["endpoint"] = plan.WGEp.ValueString()
		}
		mtu := plan.WGMtu.ValueInt64()
		if mtu == 0 {
			mtu = 1420
		}
		wg["mtu"] = mtu
		wg["allowed_ips"] = plan.WGAIP.ValueString()
		if !plan.WGKA.IsUnknown() && plan.WGKA.ValueInt64() > 0 {
			wg["persistent_keepalive"] = plan.WGKA.ValueInt64()
		}
		body["wireguard"] = wg
	}
	return body
}

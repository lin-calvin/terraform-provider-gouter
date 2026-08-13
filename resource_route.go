package main

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type routeResource struct {
	client *Client
}

type routeResourceModel struct {
	Prefix  types.String `tfsdk:"prefix"`
	NextHop types.String `tfsdk:"next_hop"`
	Via     types.String `tfsdk:"via"`
	Export  types.Bool   `tfsdk:"export"`
	InLabel types.Int64  `tfsdk:"in_label"`
	Labels  types.List   `tfsdk:"labels"`
}

func newRouteResource() resource.Resource {
	return &routeResource{}
}

func (r *routeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route"
}

func (r *routeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"prefix":   schema.StringAttribute{Required: true},
			"next_hop": schema.StringAttribute{Required: true},
			"via":      schema.StringAttribute{Optional: true},
			"export":   schema.BoolAttribute{Optional: true},
			"in_label": schema.Int64Attribute{Optional: true},
			"labels":   schema.ListAttribute{ElementType: types.Int64Type, Optional: true},
		},
	}
}

func (r *routeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	p, ok := req.ProviderData.(*gouterProvider)
	if !ok {
		return
	}
	r.client = p.client
}

func (r *routeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan routeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := buildRouteBody(plan)
	if err := r.client.Post("/routes", body, nil); err != nil {
		resp.Diagnostics.AddError("create route", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *routeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state routeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var result map[string]any
	if err := r.client.Get("/routes", state.Prefix.ValueString(), &result); err != nil {
		if errors.Is(err, ErrNotFound) {
			// 资源已不存在：从 state 移除，让 Terraform 决定重建
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read route", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *routeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan routeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := buildRouteBody(plan)
	if err := r.client.Post("/routes", body, nil); err != nil {
		resp.Diagnostics.AddError("update route", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *routeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state routeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete("/routes", state.Prefix.ValueString()); err != nil {
		resp.Diagnostics.AddError("delete route", err.Error())
		return
	}
}

func (r *routeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("prefix"), req, resp)
}

func buildRouteBody(plan routeResourceModel) map[string]any {
	body := map[string]any{
		"prefix":   plan.Prefix.ValueString(),
		"next_hop": plan.NextHop.ValueString(),
	}
	if !plan.Via.IsUnknown() && plan.Via.ValueString() != "" {
		body["via"] = plan.Via.ValueString()
	}
	if !plan.Export.IsUnknown() {
		body["export"] = plan.Export.ValueBool()
	}
	if !plan.InLabel.IsUnknown() && plan.InLabel.ValueInt64() > 0 {
		body["in_label"] = plan.InLabel.ValueInt64()
	}
	if !plan.Labels.IsUnknown() && len(plan.Labels.Elements()) > 0 {
		var labels []uint32
		for _, e := range plan.Labels.Elements() {
			if i, ok := e.(types.Int64); ok {
				labels = append(labels, uint32(i.ValueInt64()))
			}
		}
		body["labels"] = labels
	}
	return body
}

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NoteResource struct{}

type NoteResourceModel struct {
	Path    types.String `tfsdk:"path"`
	Content types.String `tfsdk:"content"`
}

func NewNoteResource() resource.Resource {
	return &NoteResource{}
}

func (r *NoteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_note"
}

func (r *NoteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "지정한 경로에 텍스트 파일을 생성하고 관리합니다.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Required:    true,
				Description: "파일을 생성할 경로 (예: /tmp/hello.txt)",
			},
			"content": schema.StringAttribute{
				Required:    true,
				Description: "파일에 쓸 내용",
			},
		},
	}
}

func (r *NoteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NoteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := os.WriteFile(data.Path.ValueString(), []byte(data.Content.ValueString()), 0644); err != nil {
		resp.Diagnostics.AddError("파일 생성 실패", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NoteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NoteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, err := os.ReadFile(data.Path.ValueString())
	if os.IsNotExist(err) {
		// 파일이 외부에서 삭제된 경우 state에서도 제거
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("파일 읽기 실패", err.Error())
		return
	}

	data.Content = types.StringValue(string(content))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NoteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NoteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := os.WriteFile(data.Path.ValueString(), []byte(data.Content.ValueString()), 0644); err != nil {
		resp.Diagnostics.AddError("파일 수정 실패", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NoteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NoteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := os.Remove(data.Path.ValueString()); err != nil && !os.IsNotExist(err) {
		resp.Diagnostics.AddError("파일 삭제 실패", err.Error())
	}
}

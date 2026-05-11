package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type LocalNoteProvider struct{}

func New() provider.Provider {
	return &LocalNoteProvider{}
}

func (p *LocalNoteProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "localnote"
}

// provider {} 블록에서 받을 설정값이 없으므로 빈 스키마
func (p *LocalNoteProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{}
}

func (p *LocalNoteProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *LocalNoteProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNoteResource,
	}
}

func (p *LocalNoteProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

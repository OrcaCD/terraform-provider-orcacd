package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OrcaCD/terraform-provider-orcacd/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestProviderSchema(t *testing.T) {
	server := providerserver.NewProtocol6(New("test")())()
	schema, err := server.GetProviderSchema(t.Context(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(schema.Diagnostics) != 0 {
		t.Fatalf("unexpected schema diagnostics: %v", schema.Diagnostics)
	}
	for _, kind := range []string{"agent", "application", "notification", "repository"} {
		name := "orcacd_" + kind
		if _, ok := schema.ResourceSchemas[name]; !ok {
			t.Errorf("missing resource %s", name)
		}
		if _, ok := schema.DataSourceSchemas[name]; !ok {
			t.Errorf("missing data source %s", name)
		}
	}
	if agent := schema.ResourceSchemas["orcacd_agent"]; agent != nil {
		for _, attr := range agent.Block.Attributes {
			if attr.Name == "token" && attr.Computed && attr.Sensitive {
				return
			}
		}
		t.Error("agent token must be computed and sensitive")
	}
}

func TestApplicationUpdatePreservesHubSettings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		const current = `{"id":"app","name":"example","branch":"main","imagePollEnabled":true,"imagePollIntervalSeconds":120,"deleteOldImages":true}`
		switch r.Method {
		case http.MethodGet:
			fmt.Fprint(w, current)
		case http.MethodPut:
			var body api.UpdateApplicationRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			if body.ImagePollEnabled == nil || !*body.ImagePollEnabled ||
				body.ImagePollIntervalSeconds == nil || *body.ImagePollIntervalSeconds != 120 ||
				body.DeleteOldImages == nil || !*body.DeleteOldImages {
				t.Error("update reset application image settings")
			}
			if body.Branch != "stable" || body.Icon != nil {
				t.Errorf("unexpected application update: %+v", body)
			}
			fmt.Fprint(w, `{"id":"app","name":"example","branch":"stable"}`)
		default:
			http.Error(w, "unexpected request", http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)
	client, err := api.New(server.URL, "fixture-token")
	if err != nil {
		t.Fatal(err)
	}
	prior := applicationModel{objectID: objectID{ID: types.StringValue("app"), Name: types.StringValue("example")}}
	plan := prior
	plan.Branch = types.StringValue("stable")
	if err := updateApplication(t.Context(), client, &plan, plan, prior); err != nil {
		t.Fatal(err)
	}
	if plan.Branch.ValueString() != "stable" {
		t.Error("updated branch was not written to state")
	}
}

package logging

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Regression test for
// https://github.com/hashicorp/terraform-provider-google/issues/15725:
// importing a project bucket config must store the bare project ID, not
// "projects/{{id}}", otherwise the ForceNew "project" field shows a
// permanent diff and forces replacement.
func TestLoggingProjectBucketConfig_importStoresBareProjectID(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, loggingProjectBucketConfigSchema, map[string]interface{}{})
	d.SetId("projects/example-project-id/locations/global/buckets/example-bucket-id")

	results, err := resourceLoggingBucketConfigImportState("project")(d, nil)
	if err != nil {
		t.Fatalf("unexpected import error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(results))
	}

	got := results[0]
	if v := got.Get("project").(string); v != "example-project-id" {
		t.Errorf("project: expected %q, got %q", "example-project-id", v)
	}
	if v := got.Get("location").(string); v != "global" {
		t.Errorf("location: expected %q, got %q", "global", v)
	}
	if v := got.Get("bucket_id").(string); v != "example-bucket-id" {
		t.Errorf("bucket_id: expected %q, got %q", "example-bucket-id", v)
	}
}

// Other parent types keep the "{{type}}/{{id}}" form that their ID functions
// expect; make sure the project-specific fix did not change them.
func TestLoggingBucketConfig_importKeepsPrefixForNonProjectParents(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		parent string
		id     string
		want   string
	}{
		"folder": {
			parent: "folder",
			id:     "folders/123/locations/global/buckets/b",
			want:   "folders/123",
		},
		"organization": {
			parent: "organization",
			id:     "organizations/456/locations/global/buckets/b",
			want:   "organizations/456",
		},
		"billing_account": {
			parent: "billing_account",
			id:     "billingAccounts/ABC-DEF/locations/global/buckets/b",
			want:   "billingAccounts/ABC-DEF",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := map[string]*schema.Schema{
				tc.parent:   {Type: schema.TypeString, Required: true},
				"location":  {Type: schema.TypeString, Required: true},
				"bucket_id": {Type: schema.TypeString, Required: true},
			}
			d := schema.TestResourceDataRaw(t, s, map[string]interface{}{})
			d.SetId(tc.id)

			results, err := resourceLoggingBucketConfigImportState(tc.parent)(d, nil)
			if err != nil {
				t.Fatalf("unexpected import error: %v", err)
			}
			if v := results[0].Get(tc.parent).(string); v != tc.want {
				t.Errorf("%s: expected %q, got %q", tc.parent, tc.want, v)
			}
		})
	}
}

func TestLoggingProjectBucketConfig_projectDiffSuppress(t *testing.T) {
	t.Parallel()

	cases := []struct {
		old, new string
		suppress bool
	}{
		{"my-project", "my-project", true},
		{"projects/my-project", "my-project", true},
		{"my-project", "projects/my-project", true},
		{"projects/my-project", "projects/my-project", true},
		{"my-project", "other-project", false},
		{"projects/my-project", "other-project", false},
		{"", "my-project", false},
	}

	for _, tc := range cases {
		if got := projectBucketConfigProjectDiffSuppress("project", tc.old, tc.new, nil); got != tc.suppress {
			t.Errorf("DiffSuppress(%q, %q) = %v, want %v", tc.old, tc.new, got, tc.suppress)
		}
	}
}

func TestLoggingProjectBucketConfig_idNormalizesProjectPrefix(t *testing.T) {
	t.Parallel()

	for _, project := range []string{"my-project", "projects/my-project"} {
		d := schema.TestResourceDataRaw(t, loggingProjectBucketConfigSchema, map[string]interface{}{
			"project":   project,
			"location":  "global",
			"bucket_id": "my-bucket",
		})
		id, err := projectBucketConfigID(d, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := "projects/my-project/locations/global/buckets/my-bucket"; id != want {
			t.Errorf("project=%q: expected id %q, got %q", project, want, id)
		}
	}
}

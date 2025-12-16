package terraform

import (
	"encoding/json"
	"testing"
)

// TestReplacePaths_MixedTypes tests that the ReplacePaths field can handle
// both strings (attribute names) and numbers (array indices) in the same path.
// This addresses the bug where Terraform's JSON output contains numeric indices
// that were previously causing: "json: cannot unmarshal number into Go struct field"
func TestReplacePaths_MixedTypes(t *testing.T) {
	// Minimal Terraform plan JSON with replace_paths containing mixed types
	planJSON := `{
		"format_version": "1.2",
		"terraform_version": "1.14.2",
		"resource_changes": [
			{
				"address": "module.staging.google_bigquery_table.example",
				"type": "google_bigquery_table",
				"name": "example",
				"mode": "managed",
				"change": {
					"actions": ["delete", "create"],
					"before": null,
					"after": null,
					"replace_paths": [
						["external_data_configuration", 0, "schema"]
					]
				}
			}
		],
		"planned_values": {
			"root_module": {}
		},
		"resource_drift": [],
		"applyable": true,
		"complete": true,
		"errored": false
	}`

	var plan TerraformPlan
	err := json.Unmarshal([]byte(planJSON), &plan)
	if err != nil {
		t.Fatalf("Failed to unmarshal plan JSON: %v", err)
	}

	// Verify we got the resource change
	if len(plan.ResourceChanges) != 1 {
		t.Fatalf("Expected 1 resource change, got %d", len(plan.ResourceChanges))
	}

	rc := plan.ResourceChanges[0]

	// Verify basic fields
	if rc.Address != "module.staging.google_bigquery_table.example" {
		t.Errorf("Expected address 'module.staging.google_bigquery_table.example', got '%s'", rc.Address)
	}

	// Verify replace_paths exists
	if rc.Change.ReplacePaths == nil {
		t.Fatal("Expected ReplacePaths to be non-nil")
	}

	if len(rc.Change.ReplacePaths) != 1 {
		t.Fatalf("Expected 1 replace path, got %d", len(rc.Change.ReplacePaths))
	}

	path := rc.Change.ReplacePaths[0]
	if len(path) != 3 {
		t.Fatalf("Expected path length 3, got %d", len(path))
	}

	// Verify the first element is a string
	if path[0] != "external_data_configuration" {
		t.Errorf("Expected first element to be 'external_data_configuration', got %v", path[0])
	}

	// Verify the second element is a number (parsed as float64 in JSON)
	// This is the critical test - it must be a number, not a string
	if num, ok := path[1].(float64); !ok {
		t.Errorf("Expected second element to be a number (float64), got %T", path[1])
	} else if num != 0 {
		t.Errorf("Expected second element to be 0, got %v", num)
	}

	// Verify the third element is a string
	if path[2] != "schema" {
		t.Errorf("Expected third element to be 'schema', got %v", path[2])
	}
}

// TestReplacePaths_StringOnly tests that paths with only strings still work correctly
func TestReplacePaths_StringOnly(t *testing.T) {
	planJSON := `{
		"format_version": "1.2",
		"terraform_version": "1.14.2",
		"resource_changes": [
			{
				"address": "aws_instance.example",
				"type": "aws_instance",
				"name": "example",
				"mode": "managed",
				"change": {
					"actions": ["delete", "create"],
					"before": null,
					"after": null,
					"replace_paths": [
						["availability_zone"],
						["instance_type"]
					]
				}
			}
		],
		"planned_values": {
			"root_module": {}
		},
		"resource_drift": [],
		"applyable": true,
		"complete": true,
		"errored": false
	}`

	var plan TerraformPlan
	err := json.Unmarshal([]byte(planJSON), &plan)
	if err != nil {
		t.Fatalf("Failed to unmarshal plan JSON: %v", err)
	}

	if len(plan.ResourceChanges) != 1 {
		t.Fatalf("Expected 1 resource change, got %d", len(plan.ResourceChanges))
	}

	rc := plan.ResourceChanges[0]

	if len(rc.Change.ReplacePaths) != 2 {
		t.Fatalf("Expected 2 replace paths, got %d", len(rc.Change.ReplacePaths))
	}

	// Verify first path
	if len(rc.Change.ReplacePaths[0]) != 1 {
		t.Errorf("Expected first path length 1, got %d", len(rc.Change.ReplacePaths[0]))
	}
	if rc.Change.ReplacePaths[0][0] != "availability_zone" {
		t.Errorf("Expected 'availability_zone', got %v", rc.Change.ReplacePaths[0][0])
	}

	// Verify second path
	if len(rc.Change.ReplacePaths[1]) != 1 {
		t.Errorf("Expected second path length 1, got %d", len(rc.Change.ReplacePaths[1]))
	}
	if rc.Change.ReplacePaths[1][0] != "instance_type" {
		t.Errorf("Expected 'instance_type', got %v", rc.Change.ReplacePaths[1][0])
	}
}

// TestReplacePaths_Nil tests that missing replace_paths doesn't cause issues
func TestReplacePaths_Nil(t *testing.T) {
	planJSON := `{
		"format_version": "1.2",
		"terraform_version": "1.14.2",
		"resource_changes": [
			{
				"address": "aws_instance.example",
				"type": "aws_instance",
				"name": "example",
				"mode": "managed",
				"change": {
					"actions": ["update"],
					"before": null,
					"after": null
				}
			}
		],
		"planned_values": {
			"root_module": {}
		},
		"resource_drift": [],
		"applyable": true,
		"complete": true,
		"errored": false
	}`

	var plan TerraformPlan
	err := json.Unmarshal([]byte(planJSON), &plan)
	if err != nil {
		t.Fatalf("Failed to unmarshal plan JSON: %v", err)
	}

	if len(plan.ResourceChanges) != 1 {
		t.Fatalf("Expected 1 resource change, got %d", len(plan.ResourceChanges))
	}

	rc := plan.ResourceChanges[0]

	// ReplacePaths should be nil or empty for non-replace actions
	if rc.Change.ReplacePaths != nil && len(rc.Change.ReplacePaths) > 0 {
		t.Errorf("Expected ReplacePaths to be nil or empty, got %v", rc.Change.ReplacePaths)
	}
}

// TestReplacePaths_ComplexPath tests a complex nested path with multiple indices
func TestReplacePaths_ComplexPath(t *testing.T) {
	planJSON := `{
		"format_version": "1.2",
		"terraform_version": "1.14.2",
		"resource_changes": [
			{
				"address": "google_compute_instance.example",
				"type": "google_compute_instance",
				"name": "example",
				"mode": "managed",
				"change": {
					"actions": ["delete", "create"],
					"before": null,
					"after": null,
					"replace_paths": [
						["network_interface", 0, "access_config", 0, "nat_ip"]
					]
				}
			}
		],
		"planned_values": {
			"root_module": {}
		},
		"resource_drift": [],
		"applyable": true,
		"complete": true,
		"errored": false
	}`

	var plan TerraformPlan
	err := json.Unmarshal([]byte(planJSON), &plan)
	if err != nil {
		t.Fatalf("Failed to unmarshal plan JSON: %v", err)
	}

	if len(plan.ResourceChanges) != 1 {
		t.Fatalf("Expected 1 resource change, got %d", len(plan.ResourceChanges))
	}

	rc := plan.ResourceChanges[0]
	path := rc.Change.ReplacePaths[0]

	expectedPath := []interface{}{"network_interface", float64(0), "access_config", float64(0), "nat_ip"}
	if len(path) != len(expectedPath) {
		t.Fatalf("Expected path length %d, got %d", len(expectedPath), len(path))
	}

	for i, expected := range expectedPath {
		switch v := expected.(type) {
		case string:
			if path[i] != v {
				t.Errorf("At position %d, expected string '%s', got %v", i, v, path[i])
			}
		case float64:
			if num, ok := path[i].(float64); !ok {
				t.Errorf("At position %d, expected float64, got %T", i, path[i])
			} else if num != v {
				t.Errorf("At position %d, expected %v, got %v", i, v, num)
			}
		}
	}
}

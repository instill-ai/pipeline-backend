package data

import (
	"testing"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func TestMapCopy(t *testing.T) {
	t.Run("Copy simple map", func(t *testing.T) {
		original := Map{
			"key1": NewString("value1"),
			"key2": NewNumberFromInteger(42),
			"key3": NewBoolean(true),
		}

		copied := original.Copy()

		// Verify copied map has same values
		if !copied.Equal(original) {
			t.Error("Copied map should equal original")
		}

		// Verify it's a deep copy by modifying original
		original["key1"] = NewString("modified")
		if copied["key1"].(*stringData).String() != "value1" {
			t.Error("Modifying original should not affect copy")
		}
	})

	t.Run("Copy nested map", func(t *testing.T) {
		original := Map{
			"outer": Map{
				"inner": Map{
					"value": NewString("nested"),
				},
			},
		}

		copied := original.Copy()

		// Verify it's a deep copy
		originalInner := original["outer"].(Map)["inner"].(Map)
		originalInner["value"] = NewString("modified")

		copiedInner := copied["outer"].(Map)["inner"].(Map)
		if copiedInner["value"].(*stringData).String() != "nested" {
			t.Error("Modifying nested original should not affect copy")
		}
	})

	t.Run("Copy map with array", func(t *testing.T) {
		original := Map{
			"array": Array{
				NewString("item1"),
				NewString("item2"),
			},
		}

		copied := original.Copy()

		// Modify original array
		originalArray := original["array"].(Array)
		originalArray[0] = NewString("modified")

		// Verify copy is unaffected
		copiedArray := copied["array"].(Array)
		if copiedArray[0].(*stringData).String() != "item1" {
			t.Error("Modifying array in original should not affect copy")
		}
	})

	t.Run("Copy nil map", func(t *testing.T) {
		var original Map
		copied := original.Copy()

		if copied != nil {
			t.Error("Copy of nil map should be nil")
		}
	})

	t.Run("Copy empty map", func(t *testing.T) {
		original := Map{}
		copied := original.Copy()

		if copied == nil {
			t.Error("Copy of empty map should not be nil")
		}
		if len(copied) != 0 {
			t.Error("Copy of empty map should be empty")
		}
	})
}

func TestArrayCopy(t *testing.T) {
	t.Run("Copy simple array", func(t *testing.T) {
		original := Array{
			NewString("value1"),
			NewNumberFromInteger(42),
			NewBoolean(true),
		}

		copied := original.Copy()

		// Verify copied array has same values
		if !copied.Equal(original) {
			t.Error("Copied array should equal original")
		}

		// Verify it's a deep copy by modifying original
		original[0] = NewString("modified")
		if copied[0].(*stringData).String() != "value1" {
			t.Error("Modifying original should not affect copy")
		}
	})

	t.Run("Copy nested array", func(t *testing.T) {
		original := Array{
			Array{
				Array{
					NewString("nested"),
				},
			},
		}

		copied := original.Copy()

		// Verify it's a deep copy
		originalInner := original[0].(Array)[0].(Array)
		originalInner[0] = NewString("modified")

		copiedInner := copied[0].(Array)[0].(Array)
		if copiedInner[0].(*stringData).String() != "nested" {
			t.Error("Modifying nested original should not affect copy")
		}
	})

	t.Run("Copy array with map", func(t *testing.T) {
		original := Array{
			Map{
				"key": NewString("value"),
			},
		}

		copied := original.Copy()

		// Modify original map
		originalMap := original[0].(Map)
		originalMap["key"] = NewString("modified")

		// Verify copy is unaffected
		copiedMap := copied[0].(Map)
		if copiedMap["key"].(*stringData).String() != "value" {
			t.Error("Modifying map in original should not affect copy")
		}
	})

	t.Run("Copy nil array", func(t *testing.T) {
		var original Array
		copied := original.Copy()

		if copied != nil {
			t.Error("Copy of nil array should be nil")
		}
	})

	t.Run("Copy empty array", func(t *testing.T) {
		original := Array{}
		copied := original.Copy()

		if copied == nil {
			t.Error("Copy of empty array should not be nil")
		}
		if len(copied) != 0 {
			t.Error("Copy of empty array should be empty")
		}
	})
}

func TestCopyValue(t *testing.T) {
	t.Run("Copy nil value", func(t *testing.T) {
		var original format.Value
		copied := copyValue(original)

		if copied != nil {
			t.Error("Copy of nil value should be nil")
		}
	})

	t.Run("Copy primitive types", func(t *testing.T) {
		// Primitive types should be returned as-is since they're immutable
		tests := []format.Value{
			NewString("test"),
			NewNumberFromInteger(123),
			NewBoolean(true),
			NewNull(),
		}

		for _, original := range tests {
			copied := copyValue(original)
			if !copied.Equal(original) {
				t.Errorf("Copied value should equal original for type %T", original)
			}
		}
	})

	t.Run("Copy complex nested structure", func(t *testing.T) {
		original := Map{
			"users": Array{
				Map{
					"name": NewString("Alice"),
					"age":  NewNumberFromInteger(30),
					"tags": Array{
						NewString("admin"),
						NewString("active"),
					},
				},
				Map{
					"name": NewString("Bob"),
					"age":  NewNumberFromInteger(25),
					"tags": Array{
						NewString("user"),
					},
				},
			},
			"metadata": Map{
				"version": NewNumberFromInteger(1),
				"active":  NewBoolean(true),
			},
		}

		copied := copyValue(original).(Map)

		// Verify it's a deep copy
		originalUsers := original["users"].(Array)
		originalFirstUser := originalUsers[0].(Map)
		originalFirstUser["name"] = NewString("Modified")

		copiedUsers := copied["users"].(Array)
		copiedFirstUser := copiedUsers[0].(Map)
		if copiedFirstUser["name"].(*stringData).String() != "Alice" {
			t.Error("Deep nested modification should not affect copy")
		}
	})
}

// TestCopyConcurrency verifies that Copy() prevents race conditions
func TestCopyConcurrency(t *testing.T) {
	original := Map{
		"key1": NewString("value1"),
		"key2": Map{
			"nested": NewString("nested_value"),
		},
	}

	// Create a copy
	copied := original.Copy()

	// Simulate concurrent modifications on original
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			original["key1"] = NewString("modified")
			original["new_key"] = NewNumberFromInteger(i)
		}
		done <- true
	}()

	// Simultaneously access the copy (should not race)
	go func() {
		for i := 0; i < 100; i++ {
			_ = copied["key1"]
			_ = copied["key2"]
		}
		done <- true
	}()

	<-done
	<-done

	// Verify copy is unaffected by modifications to original
	if copied["key1"].(*stringData).String() != "value1" {
		t.Error("Copy should be unaffected by concurrent modifications to original")
	}
	if _, exists := copied["new_key"]; exists {
		t.Error("Copy should not have new keys added to original")
	}
}

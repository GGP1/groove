package scan

// TODO: test
// var columns = []string{"string", "number", "string_slice", "time", "boolean"}

// type test struct {
// 	String      string       `json:"string,omitempty"`
// 	Number      int          `json:"number,omitempty"`
// 	StringSlice []string     `json:"string_slice,omitempty" db:"string_slice"`
// 	EmbeddedPtr *embeddedPtr `json:"embedded,omitempty"`
// 	Embedded    embedded
// }

// type embedded struct {
// 	Boolean *bool `json:"boolean,omitempty"`
// }

// type embeddedPtr struct {
// 	Time time.Time `json:"time,omitempty"`
// }

// func TestRows(t *testing.T) {
// 	dest := test{
// 		EmbeddedPtr: &embeddedPtr{},
// 	}
// 	expected := []interface{}{
// 		&dest.String,
// 		&dest.Number,
// 		&dest.StringSlice,
// 		&dest.EmbeddedPtr.Time,
// 		&dest.Embedded.Boolean,
// 	}

// 	got, err := Rows(&dest, columns)
// 	assert.NoError(t, err)

// 	assert.Equal(t, expected, got)
// }

// func TestGetFieldsErrors(t *testing.T) {
// 	t.Run("Not pointer", func(t *testing.T) {
// 		_, err := GetFields([]test{}, columns)
// 		assert.Error(t, err)
// 	})
// 	t.Run("Nil", func(t *testing.T) {
// 		_, err := GetFields(nil, columns)
// 		assert.Error(t, err)
// 	})
// 	t.Run("Not a struct", func(t *testing.T) {
// 		_, err := GetFields(new(string), columns)
// 		assert.Error(t, err)
// 	})
// }

// func BenchmarkGetFields(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		_, _ = GetFields(&test{}, columns)
// 	}
// }

// func BenchmarkManual(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		dest := &test{
// 			EmbeddedPtr: &embeddedPtr{},
// 		}
// 		values := make([]interface{}, 0, len(columns))
// 		for _, c := range columns {
// 			switch c {
// 			case "string":
// 				values = append(values, &dest.String)
// 			case "number":
// 				values = append(values, &dest.Number)
// 			case "string_slice":
// 				values = append(values, &dest.StringSlice)
// 			case "time":
// 				values = append(values, &dest.EmbeddedPtr.Time)
// 			case "boolean":
// 				values = append(values, &dest.Embedded.Boolean)
// 			}
// 		}
// 	}
// }
